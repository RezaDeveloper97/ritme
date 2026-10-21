package cache

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/metrics"
)

// cacheCount reads cache_requests_total{cache=name,result=result}.
func cacheCount(t *testing.T, name, result string) float64 {
	t.Helper()
	mfs, err := metrics.Registry().Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() != "cache_requests_total" {
			continue
		}
		for _, m := range mf.GetMetric() {
			got := map[string]string{}
			for _, lp := range m.GetLabel() {
				got[lp.GetName()] = lp.GetValue()
			}
			if got["cache"] == name && got["result"] == result {
				return m.GetCounter().GetValue()
			}
		}
	}
	return 0
}

func TestRecord(t *testing.T) {
	Record("rec_test", nil)
	Record("rec_test", ErrMiss)
	Record("rec_test", redis.Nil)
	Record("rec_test", errors.New("boom"))
	Hit("rec_test")
	assert.InDelta(t, 2, cacheCount(t, "rec_test", metrics.CacheHit), 0)
	assert.InDelta(t, 2, cacheCount(t, "rec_test", metrics.CacheMiss), 0)
	assert.InDelta(t, 1, cacheCount(t, "rec_test", metrics.CacheError), 0)
}

// TrackGets counts the GETs of one namespace sent through the raw client (how internal/cycle/cache reads), and
// nothing else.
func TestTrackGets(t *testing.T) {
	mr := miniredis.RunT(t)
	c := NewFromClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), "ritme-go:")
	t.Cleanup(func() { _ = c.Close() })
	c.TrackGets("cycle-engine:", "track_test")
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "cycle-engine:1:a", "{}", 0))
	_, err := c.Redis().Get(ctx, c.Key("cycle-engine:1:a")).Bytes()
	require.NoError(t, err)
	_, err = c.Redis().Get(ctx, c.Key("cycle-engine:1:b")).Bytes()
	require.ErrorIs(t, err, redis.Nil)
	_, err = c.Get(ctx, "other:1") // another namespace: not counted
	require.ErrorIs(t, err, ErrMiss)

	pipe := c.Redis().Pipeline()
	pipe.Get(ctx, c.Key("cycle-engine:1:a"))
	_, _ = pipe.Exec(ctx)

	assert.InDelta(t, 2, cacheCount(t, "track_test", metrics.CacheHit), 0)
	assert.InDelta(t, 1, cacheCount(t, "track_test", metrics.CacheMiss), 0)

	mr.Close() // Redis down → error
	_, _ = c.Redis().Get(ctx, c.Key("cycle-engine:1:a")).Bytes()
	assert.InDelta(t, 1, cacheCount(t, "track_test", metrics.CacheError), 0)
}
