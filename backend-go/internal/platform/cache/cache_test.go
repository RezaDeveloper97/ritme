package cache

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyPrefix(t *testing.T) {
	c := NewFromClient(redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}), "ritme-go:")
	assert.Equal(t, "ritme-go:languages.registry", c.Key("languages.registry"))
}

// Integration: runs with `make test-int PKG=./internal/platform/cache/...`.
func TestRoundTrip(t *testing.T) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set")
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	c := NewFromClient(rdb, "ritme-go-test:")
	t.Cleanup(func() { _ = c.Close() })
	ctx := context.Background()

	require.NoError(t, c.Ping(ctx))
	require.NoError(t, c.Set(ctx, "k", "v", time.Minute))

	raw, err := rdb.Get(ctx, "ritme-go-test:k").Result()
	require.NoError(t, err)
	assert.Equal(t, "v", raw, "stored under the prefix")

	v, err := c.Get(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, "v", v)

	require.NoError(t, c.Delete(ctx, "k"))
	_, err = c.Get(ctx, "k")
	assert.ErrorIs(t, err, ErrMiss)
}
