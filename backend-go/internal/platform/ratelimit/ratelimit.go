// Package ratelimit is Laravel's `throttle:N,M` middleware (Illuminate\Routing\Middleware\
// ThrottleRequests over Illuminate\Cache\RateLimiter) on Redis:
//
//   - fixed window: the first hit starts a window of `decay`; the counter and a timer key
//     (holding the unix time the window ends) expire together;
//   - a request is rejected when the counter has reached maxAttempts and the timer still
//     exists; rejected requests are not counted;
//   - the key is sha1(user id) for an authenticated request, else sha1("|" + client IP)
//     (route domain is always empty here). Like Laravel's unnamed limiters, the key does
//     NOT include the route: every throttled route shares one counter per user/IP, each
//     comparing it against its own limit;
//   - success: X-RateLimit-Limit and X-RateLimit-Remaining on the response (also on error
//     responses the handler returns); rejection: the framework 429 "Too Many Attempts."
//     with Retry-After, X-RateLimit-Reset, X-RateLimit-Limit and X-RateLimit-Remaining: 0.
//
// "Now" is the request clock (clock.FromContext), like Carbon::now() in Laravel.
package ratelimit

import (
	"context"
	"crypto/sha1" //nolint:gosec // G505: key derivation identical to Laravel, not a security boundary
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// Limiter is RateLimiter on the Go Redis prefix.
type Limiter struct {
	cache *cache.Client
	base  clock.Clock
}

// New returns a limiter. base is the fallback clock when a request carries none.
func New(c *cache.Client, base clock.Clock) *Limiter {
	return &Limiter{cache: c, base: base}
}

// Result is the outcome of one Attempt.
type Result struct {
	Allowed   bool
	Hits      int   // counter value after this attempt (unchanged when rejected)
	Remaining int   // max(0, maxAttempts - Hits); 0 when rejected
	ResetAt   int64 // unix time the window ends (the timer value)
}

// RetryAfter is availableIn(): seconds until the window ends (never negative).
func (r Result) RetryAfter(now time.Time) int {
	return int(max(0, r.ResetAt-now.Unix()))
}

// attemptScript is tooManyAttempts() followed by hit() in one atomic step.
//
//	KEYS[1] counter  KEYS[2] timer   ARGV[1] maxAttempts  ARGV[2] decay seconds  ARGV[3] now
//
// Returns {allowed(0/1), hits, timer}.
var attemptScript = redis.NewScript(`
local max = tonumber(ARGV[1])
local decay = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local attempts = tonumber(redis.call('GET', KEYS[1]) or '0')
if attempts >= max then
  local timer = redis.call('GET', KEYS[2])
  if timer then
    return {0, attempts, tonumber(timer)}
  end
  redis.call('DEL', KEYS[1])
end
redis.call('SET', KEYS[2], now + decay, 'EX', decay, 'NX')
local added = redis.call('SET', KEYS[1], 0, 'EX', decay, 'NX')
local hits = redis.call('INCR', KEYS[1])
if (not added) and hits == 1 then
  redis.call('SET', KEYS[1], 1, 'EX', decay)
end
return {1, hits, tonumber(redis.call('GET', KEYS[2]) or (now + decay))}
`)

// Attempt counts one request against key (already hashed or not; it is used verbatim
// under the "throttle:" namespace).
func (l *Limiter) Attempt(ctx context.Context, key string, maxAttempts int, decay time.Duration, now time.Time) (Result, error) {
	decaySec := int64(decay / time.Second)
	if decaySec < 1 {
		decaySec = 1
	}
	counter := l.cache.Key("throttle:" + key)
	timer := counter + ":timer"
	out, err := attemptScript.Run(ctx, l.cache.Redis(), []string{counter, timer}, maxAttempts, decaySec, now.Unix()).Int64Slice()
	if err != nil {
		return Result{}, fmt.Errorf("ratelimit: %w", err)
	}
	if len(out) != 3 {
		return Result{}, fmt.Errorf("ratelimit: unexpected script reply %v", out)
	}
	res := Result{Allowed: out[0] == 1, Hits: int(out[1]), ResetAt: out[2]}
	if res.Allowed {
		res.Remaining = max(0, maxAttempts-res.Hits)
	}
	return res, nil
}

// Identity returns the authenticated user's id for the request, or "" for a guest.
// auth.ThrottleIdentity is the implementation for routes behind auth.RequireUser.
type Identity func(c fiber.Ctx) string

// Signature is ThrottleRequests::resolveRequestSignature: sha1(user id) when the request is
// authenticated, else sha1("|" + client IP) (trusted X-Forwarded-For, see cmd/api).
func Signature(c fiber.Ctx, identity Identity) string {
	var src string
	if identity != nil {
		src = identity(c)
	}
	if src == "" {
		src = "|" + c.IP()
	}
	sum := sha1.Sum([]byte(src)) //nolint:gosec // G401: same identifier as Laravel
	return hex.EncodeToString(sum[:])
}

// Middleware is `throttle:maxAttempts,decayMinutes` (decay = decayMinutes minutes).
// identity may be nil for guest-only routes.
func (l *Limiter) Middleware(maxAttempts int, decay time.Duration, identity Identity) fiber.Handler {
	return l.middleware("", maxAttempts, decay, identity)
}

// Named is Middleware on a counter of its own: the key is name + ":" + Signature, so it never
// shares the unnamed per-user/IP counter of the Laravel-compatible throttles (a burst of
// writes must not use up refresh-session's 10/min). Same headers and 429 as Middleware.
func (l *Limiter) Named(name string, maxAttempts int, decay time.Duration, identity Identity) fiber.Handler {
	if name == "" {
		panic("ratelimit: Named needs a name")
	}
	return l.middleware(name+":", maxAttempts, decay, identity)
}

func (l *Limiter) middleware(prefix string, maxAttempts int, decay time.Duration, identity Identity) fiber.Handler {
	limit := strconv.Itoa(maxAttempts)
	return func(c fiber.Ctx) error {
		now := clock.FromContext(c, l.base).Now()
		res, err := l.Attempt(c.Context(), prefix+Signature(c, identity), maxAttempts, decay, now)
		if err != nil {
			return err
		}
		if !res.Allowed {
			retry := res.RetryAfter(now)
			return httpx.TooManyRequests(retry, maxAttempts, 0, now.Unix()+int64(retry))
		}
		err = c.Next()
		c.Set("X-RateLimit-Limit", limit)
		c.Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
		return err
	}
}
