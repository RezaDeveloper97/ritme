// Package cache is the port of CycleEngineCache (backend/app/Services/HealthEngine/
// CycleEngineCache.php): a per-user Redis cache of cycle-engine results (/cycle/today,
// /cycle/date/{date}, /cycle/month/{y}/{m}).
//
// The key is content-addressed:
//
//	ritme-go:cycle-engine:{uid}:{calc_version}:{locale}:{today}:{scope}:{hash(inputs)}
//
// where inputs are the engine's actual inputs (profile row, every cycle_histories row, the daily
// logs of the range and, when tips are part of the result, the recommendations signature), so a
// cached value can never outlive a write, whichever code path made it. Values are the computed
// JSON (unescaped; the response encoder applies the endpoint's JSON flags), so a hit returns
// exactly the bytes a miss would have produced. Correctness never depends on the cache: any Redis
// or encoding failure falls back to computing directly, and a disabled cache always computes.
//
// The Go namespace is separate from Laravel's (`ritme-go:` prefix, sha256 instead of xxh128 over
// PHP serialize()); the two stacks never read each other's entries.
package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	platformcache "github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// TTL bounds dead entries; keys roll over with the date anyway (CycleEngineCache::TTL_SECONDS).
const TTL = 24 * time.Hour

// schema is bumped when the cached payload shape changes, so old entries are ignored.
const schema = 1

// Key identifies one cached computation.
type Key struct {
	UserID uint64
	// Version is user_profiles.calculation_version (0 without a profile).
	Version int64
	Locale  string
	// Today is the Tehran calendar day of the request clock.
	Today civildate.Date
	// Scope names what is computed incl. its arguments ("day:2026-09-23 00:00:00",
	// "month:2026-09:calendar").
	Scope string
	// Inputs are the engine inputs; anything encoding/json can marshal deterministically.
	Inputs any
}

// String renders the key (without the Redis prefix).
func (k Key) String() (string, error) {
	payload, err := json.Marshal(struct {
		Schema int
		Inputs any
	}{schema, k.Inputs})
	if err != nil {
		return "", fmt.Errorf("cycle cache: hash inputs: %w", err)
	}
	sum := sha256.Sum256(payload)
	return strings.Join([]string{
		"cycle-engine",
		strconv.FormatUint(k.UserID, 10),
		strconv.FormatInt(k.Version, 10),
		k.Locale,
		k.Today.String(),
		k.Scope,
		hex.EncodeToString(sum[:16]),
	}, ":"), nil
}

// Cache is the engine cache. A nil *Cache or a disabled one always computes.
type Cache struct {
	client  *platformcache.Client
	enabled bool
	logger  *slog.Logger
}

// New returns the cache over client; enabled=false turns it into a pass-through (the contract
// suite runs with it on and off).
func New(client *platformcache.Client, enabled bool, logger *slog.Logger) *Cache {
	if logger == nil {
		logger = slog.Default()
	}
	return &Cache{client: client, enabled: enabled && client != nil, logger: logger}
}

// Enabled reports whether lookups go to Redis.
func (c *Cache) Enabled() bool { return c != nil && c.enabled }

// Remember returns the cached JSON for key, or computes, stores and returns it. Cache failures
// are logged and never fail the request; compute errors are returned as is.
func (c *Cache) Remember(ctx context.Context, key Key, compute func() ([]byte, error)) ([]byte, error) {
	if !c.Enabled() {
		return compute()
	}
	k, err := key.String()
	if err != nil {
		c.warn(ctx, err)
		return compute()
	}
	full := c.client.Key(k)

	cached, err := c.client.Redis().Get(ctx, full).Bytes()
	switch {
	case err == nil && json.Valid(cached):
		return cached, nil
	case err != nil && !errors.Is(err, redis.Nil):
		c.warn(ctx, err)
		return compute()
	}

	value, err := compute()
	if err != nil {
		return nil, err
	}
	if err := c.client.Redis().Set(ctx, full, value, TTL).Err(); err != nil {
		c.warn(ctx, err)
	}
	return value, nil
}

func (c *Cache) warn(ctx context.Context, err error) {
	c.logger.WarnContext(ctx, "CycleEngineCache: cache unavailable, computing directly", slog.String("error", err.Error()))
}
