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
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ritme/backend-go/internal/platform/config"
)

// ErrMiss is returned by Get when the key does not exist.
var ErrMiss = errors.New("cache: miss")

// Client is a prefixed Redis client. Use Key() when calling Redis() directly.
type Client struct {
	rdb    *redis.Client
	prefix string
}

// New builds a client; it does not dial. Call Ping to fail fast.
func New(cfg config.Redis) *Client {
	return NewFromClient(redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	}), cfg.Prefix)
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

// Close closes the underlying client.
func (c *Client) Close() error { return c.rdb.Close() }
