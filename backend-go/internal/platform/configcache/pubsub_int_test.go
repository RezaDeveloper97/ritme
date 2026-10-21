package configcache_test

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/configcache"
)

// Two hubs on one Redis play two API processes: a change published by one reaches the other's caches.
// Integration: `make test-int PKG=./internal/platform/configcache/...`.
func TestPubSub_SecondProcessSeesTheChange(t *testing.T) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set")
	}
	prefix := "ritme-go-test-" + time.Now().Format("150405.000000") + ":"
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	newClient := func() *cache.Client {
		c := cache.NewFromClient(redis.NewClient(&redis.Options{Addr: addr}), prefix)
		t.Cleanup(func() { _ = c.Close() })
		return c
	}
	a := configcache.NewHubAt(time.Hour, time.Now)
	b := configcache.NewHubAt(time.Hour, time.Now)
	ca, cb := newClient(), newClient()
	a.Start(ctx, ca, nil)
	b.Start(ctx, cb, nil)

	var table atomic.Int64
	table.Store(1)
	load := func(context.Context) (int64, error) { return table.Load(), nil }
	inA := configcache.New("w", []string{"widgets"}, load, configcache.WithHub(a))
	inB := configcache.New("w", []string{"widgets"}, load, configcache.WithHub(b))
	v, _ := inA.Get(ctx)
	require.Equal(t, int64(1), v)
	v, _ = inB.Get(ctx)
	require.Equal(t, int64(1), v)

	// The subscription is asynchronous: wait until B is listening.
	require.Eventually(t, func() bool {
		n, err := ca.Redis().PubSubNumSub(ctx, ca.Key(configcache.Channel)).Result()
		return err == nil && n[ca.Key(configcache.Channel)] >= 2
	}, 5*time.Second, 20*time.Millisecond)

	table.Store(2) // the admin write commits…
	a.Changed(ctx, "widgets")
	v, _ = inA.Get(ctx)
	assert.Equal(t, int64(2), v, "same process: next request")
	assert.Eventually(t, func() bool { v, _ := inB.Get(ctx); return v == 2 }, 5*time.Second, 10*time.Millisecond,
		"second process: via ritme-go:config-changed")
}

// With Redis gone, Changed still invalidates locally and never fails the admin write.
func TestPubSub_RedisDownIsLocalOnly(t *testing.T) {
	c := cache.NewFromClient(redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: -1}), "ritme-go:")
	t.Cleanup(func() { _ = c.Close() })
	h := configcache.NewHubAt(time.Hour, time.Now)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	h.Start(ctx, c, nil)
	var table atomic.Int64
	cc := configcache.New("w", []string{"widgets"}, func(context.Context) (int64, error) { return table.Load(), nil }, configcache.WithHub(h))
	_, _ = cc.Get(ctx)
	table.Store(5)
	h.Changed(ctx, "widgets")
	v, err := cc.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(5), v)
}
