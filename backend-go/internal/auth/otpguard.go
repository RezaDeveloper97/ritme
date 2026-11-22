package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"

	"github.com/ritme/backend-go/internal/auth/sms"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

// OTPGuard is the OTP protection of T-M12-01 (S-01/S-02) and T-M12-01b (R-02) on Redis:
//
//   - send caps per (number, client network): at most SendShortMax codes per SendShortWindow and SendDailyMax per
//     SendDailyWindow (sliding), plus a ceiling per number across every network (SendGlobalMax per SendDailyWindow);
//   - failure budget across codes per (number, network): FailureMax wrong codes per FailureWindow lock that network out
//     of the number for LockFor, during which even the right code is refused from it; FailureGlobalMax wrong codes per
//     FailureGlobalWindow across every network lock the number everywhere for LockFor;
//   - send lock: one SET-NX-style lock per number around delete/insert/dispatch, so parallel sends dispatch one SMS.
//
// The network is ratelimit.ClientNet (IPv4 /32, IPv6 /64): a stranger who uses up the caps or trips the lock from her
// network does not stop the owner on another one (R-02) until the much higher global ceiling. Keys never carry a
// user id, so deleting the account and signing up again resets nothing.
//
// Every decision compares against the request clock (X-Test-Now in tests), never Redis' own clock: key TTLs only
// collect garbage. Keys hold sha256(mobile) and sha256(network), never the number or the address.
type OTPGuard struct {
	cache  *cache.Client
	lim    config.OTPLimits
	logger *slog.Logger
}

// otpLockouts counts numbers locked by the failure budget (T-M12-01).
var otpLockouts = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "auth_otp_lockouts_total",
	Help: "Mobile numbers locked after too many wrong OTP codes.",
})

func init() { metrics.Registry().MustRegister(otpLockouts) }

// NewOTPGuard returns the guard (nil cache → nil guard: no guard, tests only).
func NewOTPGuard(c *cache.Client, lim config.OTPLimits, logger *slog.Logger) *OTPGuard {
	if c == nil {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &OTPGuard{cache: c, lim: lim, logger: logger}
}

// OTPRefusal is a refused send or verify: the limit that was hit and the seconds until it lifts.
type OTPRefusal struct {
	Limit      int
	RetryAfter int
}

// key is the per-number (all networks) key of kind.
func (g *OTPGuard) key(kind, mobile string) string {
	sum := sha256.Sum256([]byte(mobile))
	return g.cache.Key("otp:" + kind + ":" + hex.EncodeToString(sum[:]))
}

// netKey is the per-(number, network) key of kind; net is a ratelimit.ClientNet bucket.
func (g *OTPGuard) netKey(kind, mobile, net string) string {
	m := sha256.Sum256([]byte(mobile))
	n := sha256.Sum256([]byte(net))
	return g.cache.Key("otp:" + kind + ":" + hex.EncodeToString(m[:]) + ":net:" + hex.EncodeToString(n[:16]))
}

// noCeiling turns an unset (0) global ceiling into "never full" for the Lua scripts.
func noCeiling(n int) int {
	if n < 1 {
		return math.MaxInt32
	}
	return n
}

func ms(d time.Duration) int64 { return d.Milliseconds() }

// ceilSec is milliseconds rounded up to whole seconds (at least 1).
func ceilSec(msLeft int64) int { return int(max(1, (msLeft+999)/1000)) }

func member(now time.Time) (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: otp guard: %w", err)
	}
	return fmt.Sprintf("%d-%s", now.UnixMilli(), hex.EncodeToString(b)), nil
}

// lockScript takes the send lock unless an unexpired one exists.
//
//	KEYS[1] lock   ARGV[1] now ms   ARGV[2] ttl ms   → {acquired(0/1), lock end ms}
var lockScript = redis.NewScript(`
local now = tonumber(ARGV[1])
local ttl = tonumber(ARGV[2])
local v = tonumber(redis.call('GET', KEYS[1]) or '0')
if v > now then
  return {0, v}
end
redis.call('SET', KEYS[1], now + ttl, 'PX', ttl)
return {1, now + ttl}
`)

// AcquireSend takes the per-number send lock. ok=false → another send holds it; retry = seconds until it ends.
func (g *OTPGuard) AcquireSend(ctx context.Context, mobile string, now time.Time) (ok bool, retry int, err error) {
	out, err := lockScript.Run(ctx, g.cache.Redis(), []string{g.key("send_lock", mobile)}, now.UnixMilli(), ms(g.lim.SendLock)).Int64Slice()
	if err != nil {
		return false, 0, fmt.Errorf("auth: otp send lock: %w", err)
	}
	if out[0] == 1 {
		return true, 0, nil
	}
	return false, ceilSec(out[1] - now.UnixMilli()), nil
}

// ReleaseSend drops the send lock after a send that did not go out (cap refusal, DB error).
func (g *OTPGuard) ReleaseSend(ctx context.Context, mobile string) {
	if err := g.cache.Redis().Del(context.WithoutCancel(ctx), g.key("send_lock", mobile)).Err(); err != nil {
		g.logger.ErrorContext(ctx, "auth: otp send lock release", slog.String("error", err.Error()))
	}
}

// sendCapScript is the sliding send windows: refuse when any is full, else record the send in both sets.
//
//	KEYS[1] zset of the (number, network) sends   KEYS[2] zset of the number's sends on every network
//	ARGV: now ms, short window ms, short max, long window ms, long max, global max (on the long window), member
//	→ {allowed(0/1), limit hit, ms until it lifts}
var sendCapScript = redis.NewScript(`
local now = tonumber(ARGV[1])
local sw, smax = tonumber(ARGV[2]), tonumber(ARGV[3])
local lw, lmax = tonumber(ARGV[4]), tonumber(ARGV[5])
local gmax = tonumber(ARGV[6])
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now - math.max(sw, lw))
redis.call('ZREMRANGEBYSCORE', KEYS[2], '-inf', now - lw)
local function full(key, win, maxn)
  local inwin = redis.call('ZRANGEBYSCORE', key, '(' .. (now - win), '+inf', 'WITHSCORES')
  local n = #inwin / 2
  if n < maxn then return nil end
  -- the window frees a slot when its (n-maxn+1)-th oldest send leaves it
  return tonumber(inwin[(n - maxn + 1) * 2]) + win - now
end
local wait = full(KEYS[2], lw, gmax)
if wait then return {0, gmax, wait} end
wait = full(KEYS[1], lw, lmax)
if wait then return {0, lmax, wait} end
wait = full(KEYS[1], sw, smax)
if wait then return {0, smax, wait} end
redis.call('ZADD', KEYS[1], now, ARGV[7])
redis.call('PEXPIRE', KEYS[1], math.max(sw, lw))
redis.call('ZADD', KEYS[2], now, ARGV[7])
redis.call('PEXPIRE', KEYS[2], lw)
return {1, 0, 0}
`)

// CountSend checks the send caps of (mobile, net) and the number's global ceiling and, when under all of them, records
// this send. net is ratelimit.ClientNet of the client. A refused send is not counted anywhere.
func (g *OTPGuard) CountSend(ctx context.Context, mobile, net string, now time.Time) (*OTPRefusal, error) {
	m, err := member(now)
	if err != nil {
		return nil, err
	}
	out, err := sendCapScript.Run(ctx, g.cache.Redis(), []string{g.netKey("sends", mobile, net), g.key("sends", mobile)},
		now.UnixMilli(), ms(g.lim.SendShortWindow), g.lim.SendShortMax, ms(g.lim.SendDailyWindow), g.lim.SendDailyMax,
		noCeiling(g.lim.SendGlobalMax), m).Int64Slice()
	if err != nil {
		return nil, fmt.Errorf("auth: otp send cap: %w", err)
	}
	if out[0] == 1 {
		return nil, nil
	}
	return &OTPRefusal{Limit: int(out[1]), RetryAfter: ceilSec(out[2])}, nil
}

// lockedUntil reads a lock key: its end in Unix ms, 0 when there is none.
func (g *OTPGuard) lockedUntil(ctx context.Context, key string) (int64, error) {
	v, err := g.cache.Redis().Get(ctx, key).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("auth: otp lock: %w", err)
	}
	return v, nil
}

// Locked reports whether the failure budget has locked net out of the number, or the global ceiling has locked the
// number on every network (and for how long; the later of the two when both hold).
func (g *OTPGuard) Locked(ctx context.Context, mobile, net string, now time.Time) (*OTPRefusal, error) {
	nowMs := now.UnixMilli()
	var out *OTPRefusal
	for _, l := range []struct {
		key   string
		limit int
	}{{g.netKey("locked", mobile, net), g.lim.FailureMax}, {g.key("locked", mobile), g.lim.FailureGlobalMax}} {
		v, err := g.lockedUntil(ctx, l.key)
		if err != nil {
			return nil, err
		}
		if v <= nowMs {
			continue
		}
		if r := ceilSec(v - nowMs); out == nil || r > out.RetryAfter {
			out = &OTPRefusal{Limit: l.limit, RetryAfter: r}
		}
	}
	return out, nil
}

// failureScript records one wrong code for (number, network) and for the number on every network, and locks the
// network (resp. the whole number) when its window holds its max failures.
//
//	KEYS[1] zset (number, network)  KEYS[2] lock (number, network)  KEYS[3] zset number  KEYS[4] lock number
//	ARGV: now ms, window ms, max, lock ms, member, global window ms, global max
//	→ {failures (number, network), network locked now(0/1), failures number, number locked now(0/1)}
var failureScript = redis.NewScript(`
local now, win, maxn, lock = tonumber(ARGV[1]), tonumber(ARGV[2]), tonumber(ARGV[3]), tonumber(ARGV[4])
local gwin, gmax = tonumber(ARGV[6]), tonumber(ARGV[7])
local function count(key, w, m, lockKey)
  redis.call('ZREMRANGEBYSCORE', key, '-inf', now - w)
  redis.call('ZADD', key, now, ARGV[5])
  redis.call('PEXPIRE', key, w)
  local n = redis.call('ZCOUNT', key, '(' .. (now - w), '+inf')
  if n >= m then
    redis.call('SET', lockKey, now + lock, 'PX', lock)
    return n, 1
  end
  return n, 0
end
local n, l = count(KEYS[1], win, maxn, KEYS[2])
local gn, gl = count(KEYS[3], gwin, gmax, KEYS[4])
return {n, l, gn, gl}
`)

// RecordFailure counts one wrong code against the (number, network) budget and the number's global ceiling;
// reaching either locks (that network out of the number, resp. the number everywhere) for LockFor.
func (g *OTPGuard) RecordFailure(ctx context.Context, mobile, net string, now time.Time) error {
	m, err := member(now)
	if err != nil {
		return err
	}
	gwin := g.lim.FailureGlobalWindow
	if gwin <= 0 {
		gwin = g.lim.FailureWindow
	}
	out, err := failureScript.Run(ctx, g.cache.Redis(),
		[]string{g.netKey("failures", mobile, net), g.netKey("locked", mobile, net), g.key("failures", mobile), g.key("locked", mobile)},
		now.UnixMilli(), ms(g.lim.FailureWindow), g.lim.FailureMax, ms(g.lim.LockFor), m, ms(gwin),
		noCeiling(g.lim.FailureGlobalMax)).Int64Slice()
	if err != nil {
		return fmt.Errorf("auth: otp failure budget: %w", err)
	}
	if out[1] == 1 || out[3] == 1 {
		otpLockouts.Inc()
		scope := "network"
		if out[3] == 1 {
			scope = "number"
		}
		g.logger.WarnContext(ctx, "auth: OTP failure budget exhausted, number locked",
			slog.String("mobile", sms.MaskMobile(mobile)), slog.String("scope", scope),
			slog.Int64("failures", out[0]), slog.Int64("failures_all_networks", out[2]),
			slog.Duration("locked_for", g.lim.LockFor))
	}
	return nil
}
