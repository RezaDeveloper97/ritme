package analysis

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

func testKey(uid uint64) CacheKey {
	return CacheKey{UserID: uid, AnalysisRev: 3, EngineRev: 2, Endpoint: EndpointSummary, Range: DefaultRange,
		Plus: "d0l0p0", Today: civildate.MustParse("2026-09-23"), Locale: "fa", ConfigFP: "abc"}
}

func miniCache(t *testing.T) (*Cache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewCache(cache.NewFromClient(rdb, "ritme-go:"), "b1", nil), mr
}

func TestCacheKey_FormatAndTag(t *testing.T) {
	k := testKey(1000)
	assert.Equal(t, "ana:1000:3:2:summary:6m:d0l0p0:2026-09-23:abc:fa:b1", k.redisKey("b1"))
	assert.Equal(t, `W/"urs.a3.e2-2026-09-23-summary-6m-d0l0p0-fa-abc-b1"`, k.Tag("b1"))

	// Every part separates bodies: uid, both revisions, endpoint, range, plus, day, locale, config, build.
	seen := map[string]bool{k.redisKey("b1"): true, k.redisKey("b2"): true}
	for _, mut := range []func(*CacheKey){
		func(k *CacheKey) { k.UserID = 1001 }, func(k *CacheKey) { k.AnalysisRev++ }, func(k *CacheKey) { k.EngineRev++ },
		func(k *CacheKey) { k.Endpoint = "cycle" }, func(k *CacheKey) { k.Range = Range1Y }, func(k *CacheKey) { k.Plus = "d1l1p1" },
		func(k *CacheKey) { k.Today = k.Today.AddDays(1) }, func(k *CacheKey) { k.Locale = "en" }, func(k *CacheKey) { k.ConfigFP = "def" },
	} {
		c := testKey(1000)
		mut(&c)
		rk := c.redisKey("b1")
		assert.False(t, seen[rk], rk)
		seen[rk] = true
		assert.NotEqual(t, k.Tag("b1"), c.Tag("b1"))
	}
	// A client-chosen locale never breaks the header.
	k.Locale = `f"a x`
	assert.Equal(t, `W/"urs.a3.e2-2026-09-23-summary-6m-d0l0p0-f_a_x-abc-b1"`, k.Tag("b1"))
}

func TestCache_MissThenHit(t *testing.T) {
	c, mr := miniCache(t)
	ctx := context.Background()
	calls := 0
	compute := func(context.Context) (any, error) { calls++; return jsonx.Obj("n", calls), nil }

	b, err := c.Body(ctx, testKey(1), compute)
	require.NoError(t, err)
	assert.JSONEq(t, `{"success":true,"data":{"n":1}}`, string(b))
	assert.True(t, c.Cached(ctx, testKey(1)))
	ttl := mr.TTL("ritme-go:" + testKey(1).redisKey("b1"))
	assert.Equal(t, CacheTTL, ttl)

	b, err = c.Body(ctx, testKey(1), compute)
	require.NoError(t, err)
	assert.JSONEq(t, `{"success":true,"data":{"n":1}}`, string(b), "served from Redis")
	assert.Equal(t, 1, calls)

	// Another user never reads it.
	b, err = c.Body(ctx, testKey(2), compute)
	require.NoError(t, err)
	assert.JSONEq(t, `{"success":true,"data":{"n":2}}`, string(b))
}

func TestCache_ErrorsAreNotCached(t *testing.T) {
	c, _ := miniCache(t)
	ctx := context.Background()
	boom := errors.New("boom")
	_, err := c.Body(ctx, testKey(1), func(context.Context) (any, error) { return nil, boom })
	require.ErrorIs(t, err, boom)
	assert.False(t, c.Cached(ctx, testKey(1)))
}

func TestCache_RedisDownFallsBackToCompute(t *testing.T) {
	c, mr := miniCache(t)
	mr.Close()
	calls := 0
	for range 2 {
		b, err := c.Body(context.Background(), testKey(1), func(context.Context) (any, error) { calls++; return 1, nil })
		require.NoError(t, err)
		assert.JSONEq(t, `{"success":true,"data":1}`, string(b))
	}
	assert.Equal(t, 2, calls)
}

func TestCache_SingleflightPerKey(t *testing.T) {
	c, _ := miniCache(t)
	var calls atomic.Int32
	release := make(chan struct{})
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.Body(context.Background(), testKey(1), func(context.Context) (any, error) {
				calls.Add(1)
				<-release
				return 1, nil
			})
			assert.NoError(t, err)
		}()
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()
	assert.LessOrEqual(t, calls.Load(), int32(2), "concurrent misses of one key compute once (a late arrival may hit Redis)")
}

func TestEntitlements_PlusPart(t *testing.T) {
	assert.Equal(t, "d0l0p0", entitlements{}.plusPart())
}
