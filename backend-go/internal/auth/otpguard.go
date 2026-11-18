package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"

	"github.com/ritme/backend-go/internal/auth/sms"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

// OTPGuard is the per-number OTP protection of T-M12-01 (S-01/S-02) on Redis:
//
//   - send caps: at most SendShortMax codes per SendShortWindow and SendDailyMax per SendDailyWindow (sliding);
//   - failure budget across codes: FailureMax wrong codes per FailureWindow lock the number for LockFor, during which
//     even the right code is refused;
//   - send lock: one SET-NX-style lock per number around delete/insert/dispatch, so parallel sends dispatch one SMS.
//
// Every decision compares against the request clock (X-Test-Now in tests), never Redis' own clock: key TTLs only
// collect garbage. Keys hold sha256(mobile), never the number.
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

func (g *OTPGuard) key(kind, mobile string) string {
	sum := sha256.Sum256([]byte(mobile))
	return g.cache.Key("otp:" + kind + ":" + hex.EncodeToString(sum[:]))
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

// sendCapScript is the two sliding send windows: refuse when either is full, else record the send.
//
//	KEYS[1] zset of sends   ARGV: now ms, short window ms, short max, long window ms, long max, member
//	→ {allowed(0/1), limit hit, ms until it lifts}
var sendCapScript = redis.NewScript(`
local now = tonumber(ARGV[1])
local sw, smax = tonumber(ARGV[2]), tonumber(ARGV[3])
local lw, lmax = tonumber(ARGV[4]), tonumber(ARGV[5])
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now - math.max(sw, lw))
local function full(win, maxn)
  local inwin = redis.call('ZRANGEBYSCORE', KEYS[1], '(' .. (now - win), '+inf', 'WITHSCORES')
  local n = #inwin / 2
  if n < maxn then return nil end
  -- the window frees a slot when its (n-maxn+1)-th oldest send leaves it
  return tonumber(inwin[(n - maxn + 1) * 2]) + win - now
end
local wait = full(lw, lmax)
if wait then return {0, lmax, wait} end
wait = full(sw, smax)
if wait then return {0, smax, wait} end
redis.call('ZADD', KEYS[1], now, ARGV[6])
redis.call('PEXPIRE', KEYS[1], math.max(sw, lw))
return {1, 0, 0}
`)

// CountSend checks the per-number send caps and, when under both, records this send.
func (g *OTPGuard) CountSend(ctx context.Context, mobile string, now time.Time) (*OTPRefusal, error) {
	m, err := member(now)
	if err != nil {
		return nil, err
	}
	out, err := sendCapScript.Run(ctx, g.cache.Redis(), []string{g.key("sends", mobile)}, now.UnixMilli(),
		ms(g.lim.SendShortWindow), g.lim.SendShortMax, ms(g.lim.SendDailyWindow), g.lim.SendDailyMax, m).Int64Slice()
	if err != nil {
		return nil, fmt.Errorf("auth: otp send cap: %w", err)
	}
	if out[0] == 1 {
		return nil, nil
	}
	return &OTPRefusal{Limit: int(out[1]), RetryAfter: ceilSec(out[2])}, nil
}

// Locked reports whether the failure budget has locked the number (and for how long).
func (g *OTPGuard) Locked(ctx context.Context, mobile string, now time.Time) (*OTPRefusal, error) {
	v, err := g.cache.Redis().Get(ctx, g.key("locked", mobile)).Int64()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("auth: otp lock: %w", err)
	}
	if v <= now.UnixMilli() {
		return nil, nil
	}
	return &OTPRefusal{Limit: g.lim.FailureMax, RetryAfter: ceilSec(v - now.UnixMilli())}, nil
}

// failureScript records one wrong code and locks the number when the window holds max failures.
//
//	KEYS[1] zset of failures  KEYS[2] lock   ARGV: now ms, window ms, max, lock ms, member  → {failures, locked now(0/1)}
var failureScript = redis.NewScript(`
local now, win, maxn, lock = tonumber(ARGV[1]), tonumber(ARGV[2]), tonumber(ARGV[3]), tonumber(ARGV[4])
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now - win)
redis.call('ZADD', KEYS[1], now, ARGV[5])
redis.call('PEXPIRE', KEYS[1], win)
local n = redis.call('ZCOUNT', KEYS[1], '(' .. (now - win), '+inf')
if n >= maxn then
  redis.call('SET', KEYS[2], now + lock, 'PX', lock)
  return {n, 1}
end
return {n, 0}
`)

// RecordFailure counts one wrong code against the number's budget; reaching it locks the number for LockFor.
func (g *OTPGuard) RecordFailure(ctx context.Context, mobile string, now time.Time) error {
	m, err := member(now)
	if err != nil {
		return err
	}
	out, err := failureScript.Run(ctx, g.cache.Redis(), []string{g.key("failures", mobile), g.key("locked", mobile)},
		now.UnixMilli(), ms(g.lim.FailureWindow), g.lim.FailureMax, ms(g.lim.LockFor), m).Int64Slice()
	if err != nil {
		return fmt.Errorf("auth: otp failure budget: %w", err)
	}
	if out[1] == 1 {
		otpLockouts.Inc()
		g.logger.WarnContext(ctx, "auth: OTP failure budget exhausted, number locked",
			slog.String("mobile", sms.MaskMobile(mobile)), slog.Int64("failures", out[0]),
			slog.Duration("locked_for", g.lim.LockFor))
	}
	return nil
}
