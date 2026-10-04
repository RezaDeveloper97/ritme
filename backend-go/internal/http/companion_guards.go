package http

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/care"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Companion security guards (B-N4-08b, docs/security/bloom-companion.md CMP-M2 / CMP-M3 / CMP-L1).
const (
	// CompanionReadsPerUser caps a companion's section reads (GET /companion/home and
	// GET /companions/links/{id}/sections/{section}) per CompanionAcceptWindow (CMP-L1).
	CompanionReadsPerUser = 300
	// CompanionDelegatedReadsPerUser caps delegated single-record reads (GET /care/medications|appointments/{id}
	// with for_user_id) per CompanionAcceptWindow, so a companion cannot walk the owner's record ids (CMP-M2).
	CompanionDelegatedReadsPerUser = 120

	// Failed-accept circuit breaker (CMP-M3). Every refused accept (unknown, expired, used, locked, wrong number…) of
	// any caller counts. CompanionAcceptFailuresPerMinute refusals within a minute, or CompanionAcceptFailuresPerHour
	// within an hour, pause POST /companions/accept for everyone for CompanionAcceptBreakerCooldown (429
	// too_many_requests, the throttles' generic answer) and log an error-level alert. Legitimate typos at today's
	// volume stay far below; a blind guesser spread over many accounts / IPs (the CMP-M3 scenario: ~1,000
	// guesses/hour) trips it within minutes and is held to a few hundred guesses per cool-down. Trade-off (accepted):
	// anyone can pause accepts for 15 minutes with ~30 wrong codes; the owner can still share the code later.
	CompanionAcceptFailuresPerMinute = 30
	CompanionAcceptFailuresPerHour   = 200
	CompanionAcceptBreakerCooldown   = 15 * time.Minute
)

// companionReadThrottle is the per-user limit of companion section reads (no-op without Redis).
func companionReadThrottle(d *Deps) fiber.Handler {
	return companionThrottle(d, "companion-read", CompanionReadsPerUser, auth.ThrottleIdentity)
}

// delegatedReadThrottle limits GETs that name another account in for_user_id; the owner's own reads pass untouched.
func delegatedReadThrottle(d *Deps) fiber.Handler {
	limit := companionThrottle(d, "companion-delegated-read", CompanionDelegatedReadsPerUser, auth.ThrottleIdentity)
	return func(c fiber.Ctx) error {
		raw, ok := validation.Query(c).Get(care.ForUserField)
		if !ok || raw == nil || phpval.ToString(raw) == "" {
			return c.Next()
		}
		return limit(c)
	}
}

// breakerScript counts one refusal in the minute and hour windows and opens the breaker (resetting both counters)
// when either reaches its threshold.
//
//	KEYS[1] minute counter  KEYS[2] hour counter  KEYS[3] open flag
//	ARGV[1] per-minute max  ARGV[2] per-hour max  ARGV[3] cool-down seconds
//
// Returns 1 when this refusal tripped the breaker.
var breakerScript = redis.NewScript(`
local m = redis.call('INCR', KEYS[1])
if m == 1 then redis.call('EXPIRE', KEYS[1], 60) end
local h = redis.call('INCR', KEYS[2])
if h == 1 then redis.call('EXPIRE', KEYS[2], 3600) end
if m >= tonumber(ARGV[1]) or h >= tonumber(ARGV[2]) then
  redis.call('DEL', KEYS[1], KEYS[2])
  if redis.call('SET', KEYS[3], '1', 'EX', tonumber(ARGV[3]), 'NX') then
    return 1
  end
end
return 0
`)

// redisAcceptBreaker is companion.AcceptBreaker on Redis (global keys, shared by every API instance).
type redisAcceptBreaker struct {
	c                *cache.Client
	perMinute, perHr int
	cooldown         time.Duration
}

func (b redisAcceptBreaker) key(k string) string { return b.c.Key("companion-accept-breaker:" + k) }

func (b redisAcceptBreaker) Open(ctx context.Context) (bool, int, error) {
	ttl, err := b.c.Redis().TTL(ctx, b.key("open")).Result()
	if err != nil {
		return false, 0, fmt.Errorf("companion breaker: %w", err)
	}
	if ttl <= 0 {
		return false, 0, nil
	}
	return true, max(1, int(ttl/time.Second)), nil
}

func (b redisAcceptBreaker) Failed(ctx context.Context) (bool, error) {
	n, err := breakerScript.Run(ctx, b.c.Redis(), []string{b.key("min"), b.key("hour"), b.key("open")},
		b.perMinute, b.perHr, int(b.cooldown/time.Second)).Int()
	if err != nil {
		return false, fmt.Errorf("companion breaker: %w", err)
	}
	return n == 1, nil
}

// companionAcceptBreaker is the failed-accept breaker; without Redis (unit tests) there is none.
func companionAcceptBreaker(d *Deps) companion.AcceptBreaker {
	if d.Cache == nil {
		return nil
	}
	return redisAcceptBreaker{c: d.Cache, perMinute: CompanionAcceptFailuresPerMinute,
		perHr: CompanionAcceptFailuresPerHour, cooldown: CompanionAcceptBreakerCooldown}
}
