// Package cache wraps the go-redis v9 client with the Go service's key prefix.
//
// Go and Laravel share one Redis during the strangler period. Laravel's entries are
// PHP-serialized under ritme-cache-/ritme-database- prefixes and are never read here;
// every Go key goes through Key() and lands under REDIS_PREFIX (default "ritme-go:").
package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

// ErrMiss is returned by Get when the key does not exist.
var ErrMiss = errors.New("cache: miss")

// Client is a prefixed Redis client. Use Key() when calling Redis() directly.
type Client struct {
	rdb    *redis.Client
	prefix string
}

// New builds a client; it does not dial. Call Ping to fail fast. Pool size and socket timeouts come from cfg
// (REDIS_POOL_SIZE / REDIS_*_TIMEOUT, T-M9-10); zero fields take the config defaults.
func New(cfg config.Redis) *Client {
	return NewFromClient(redis.NewClient(Options(cfg)), cfg.Prefix)
}

// Options maps cfg to the go-redis options New uses.
func Options(cfg config.Redis) *redis.Options {
	cfg = cfg.WithDefaults()
	return &redis.Options{
		Addr:         cfg.Addr(),
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}
}

// NewFromClient wraps an existing go-redis client (tests, asynq sharing).
func NewFromClient(rdb *redis.Client, prefix string) *Client {
	return &Client{rdb: rdb, prefix: prefix}
}

// Key returns the prefixed key.
func (c *Client) Key(key string) string { return c.prefix + key }

// Prefix returns the configured prefix.
func (c *Client) Prefix() string { return c.prefix }

// Redis exposes the raw client for commands not wrapped here. Always prefix keys with Key().
func (c *Client) Redis() *redis.Client { return c.rdb }

// Ping checks the connection.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("cache: ping: %w", err)
	}
	return nil
}

// Get returns the value of key, or ErrMiss.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	v, err := c.rdb.Get(ctx, c.Key(key)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrMiss
	}
	return v, err
}

// Set stores value under key; ttl 0 means no expiry.
func (c *Client) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return c.rdb.Set(ctx, c.Key(key), value, ttl).Err()
}

// Delete removes keys.
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	prefixed := make([]string, len(keys))
	for i, k := range keys {
		prefixed[i] = c.Key(k)
	}
	return c.rdb.Del(ctx, prefixed...).Err()
}

// Cache-lookup counters: cache_requests_total{cache,result} (T-M9-07).

// Hit counts one hit of the named cache.
func Hit(name string) { metrics.IncCache(name, metrics.CacheHit) }

// Miss counts one miss of the named cache.
func Miss(name string) { metrics.IncCache(name, metrics.CacheMiss) }

// Error counts one failed lookup (Redis error, undecodable value) of the named cache.
func Error(name string) { metrics.IncCache(name, metrics.CacheError) }

// Record counts a lookup from its outcome: nil → hit, ErrMiss or redis.Nil → miss, anything else → error.
func Record(name string, err error) {
	switch {
	case err == nil:
		Hit(name)
	case errors.Is(err, ErrMiss) || errors.Is(err, redis.Nil):
		Miss(name)
	default:
		Error(name)
	}
}

// TrackGets counts every GET this client sends for keys under namespace (unprefixed, e.g. "cycle-engine:") as a
// lookup of the named cache. It instruments callers that read through Redis() directly — the cycle-engine cache
// (internal/cycle/cache) — without touching them. Call it once at start-up, before serving.
func (c *Client) TrackGets(namespace, name string) {
	c.rdb.AddHook(getTracker{prefix: c.Key(namespace), name: name})
}

// getTracker is the go-redis hook behind TrackGets.
type getTracker struct{ prefix, name string }

func (getTracker) DialHook(next redis.DialHook) redis.DialHook { return next }

func (t getTracker) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if t.matches(cmd) {
			Record(t.name, err)
		}
		return err
	}
}

func (t getTracker) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		err := next(ctx, cmds)
		for _, cmd := range cmds {
			if t.matches(cmd) {
				Record(t.name, cmd.Err())
			}
		}
		return err
	}
}

func (t getTracker) matches(cmd redis.Cmder) bool {
	if cmd.Name() != "get" {
		return false
	}
	args := cmd.Args()
	if len(args) < 2 {
		return false
	}
	key, ok := args[1].(string)
	return ok && strings.HasPrefix(key, t.prefix)
}

// Close closes the underlying client.
func (c *Client) Close() error { return c.rdb.Close() }
