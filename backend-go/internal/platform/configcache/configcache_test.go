package configcache_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/configcache"
)

// clock is a settable test clock.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) add(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func newClock() *clock { return &clock{t: time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)} }

// source is a table with a version and a load counter.
type source struct {
	val   atomic.Int64
	loads atomic.Int64
	fail  atomic.Bool
}

func (s *source) load(context.Context) (int64, error) {
	s.loads.Add(1)
	if s.fail.Load() {
		return 0, errors.New("db down")
	}
	return s.val.Load(), nil
}

func TestDefaultDisabledUnderGoTest(t *testing.T) {
	assert.False(t, configcache.Default.Enabled(), "integration tests of other packages must keep reading the DB")
}

func TestGet_CachesUntilInvalidated(t *testing.T) {
	ck := newClock()
	h := configcache.NewHubAt(time.Minute, ck.now)
	src := &source{}
	src.val.Store(1)
	c := configcache.New("t", []string{"widgets"}, src.load, configcache.WithHub(h))
	ctx := context.Background()

	for range 5 {
		v, err := c.Get(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), v)
	}
	assert.Equal(t, int64(1), src.loads.Load(), "one load, then the snapshot")

	src.val.Store(2)
	v, _ := c.Get(ctx)
	assert.Equal(t, int64(1), v, "stale until someone says the table changed")

	h.Invalidate("other_table")
	v, _ = c.Get(ctx)
	assert.Equal(t, int64(1), v, "another table's change is not ours")

	h.Changed(ctx, "widgets") // no bus: local only
	v, _ = c.Get(ctx)
	assert.Equal(t, int64(2), v, "visible on the next Get of the same process")
	assert.Equal(t, int64(2), src.loads.Load())
}

func TestGet_TTLSafetyNet(t *testing.T) {
	ck := newClock()
	h := configcache.NewHubAt(configcache.DefaultTTL, ck.now)
	src := &source{}
	src.val.Store(1)
	c := configcache.New("t", []string{"widgets"}, src.load, configcache.WithHub(h))
	ctx := context.Background()
	_, _ = c.Get(ctx)
	src.val.Store(2)

	ck.add(59 * time.Second)
	v, _ := c.Get(ctx)
	assert.Equal(t, int64(1), v)
	ck.add(time.Second)
	v, _ = c.Get(ctx)
	assert.Equal(t, int64(2), v, "reloaded after 60 s without any invalidation")
}

func TestGet_FailedReloadKeepsSnapshotAndBacksOff(t *testing.T) {
	ck := newClock()
	h := configcache.NewHubAt(time.Minute, ck.now)
	src := &source{}
	src.val.Store(1)
	c := configcache.New("t", []string{"widgets"}, src.load, configcache.WithHub(h))
	ctx := context.Background()
	_, _ = c.Get(ctx)

	src.fail.Store(true)
	h.Invalidate("widgets")
	v, err := c.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), v, "previous snapshot while the DB is down")
	loads := src.loads.Load()
	_, _ = c.Get(ctx)
	_, _ = c.Get(ctx)
	assert.Equal(t, loads, src.loads.Load(), "no retry storm inside the backoff")

	src.fail.Store(false)
	src.val.Store(3)
	ck.add(6 * time.Second)
	v, _ = c.Get(ctx)
	assert.Equal(t, int64(3), v, "retried after the backoff")
}

func TestGet_FirstLoadErrorGoesToCaller(t *testing.T) {
	h := configcache.NewHubAt(time.Minute, newClock().now)
	src := &source{}
	src.fail.Store(true)
	c := configcache.New("t", []string{"widgets"}, src.load, configcache.WithHub(h))
	_, err := c.Get(context.Background())
	require.Error(t, err)
}

func TestGet_DisabledHubAlwaysLoads(t *testing.T) {
	h := configcache.NewHubAt(time.Minute, newClock().now)
	h.SetEnabled(false)
	src := &source{}
	c := configcache.New("t", []string{"widgets"}, src.load, configcache.WithHub(h))
	_, _ = c.Get(context.Background())
	_, _ = c.Get(context.Background())
	assert.Equal(t, int64(2), src.loads.Load())
}

// Run with -race: readers never block on, nor race with, a reload; concurrent stale readers share one load.
func TestGet_ConcurrentDuringReload(t *testing.T) {
	h := configcache.NewHubAt(time.Minute, time.Now)
	var loads atomic.Int64
	release := make(chan struct{})
	var version atomic.Int64
	load := func(context.Context) (map[string]int64, error) {
		loads.Add(1)
		if loads.Load() > 1 {
			<-release
		}
		return map[string]int64{"v": version.Load()}, nil
	}
	c := configcache.New("t", []string{"widgets"}, load, configcache.WithHub(h))
	ctx := context.Background()
	_, err := c.Get(ctx)
	require.NoError(t, err)

	version.Store(1)
	h.Invalidate("widgets")
	var wg sync.WaitGroup
	results := make([]int64, 64)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			m, err := c.Get(ctx)
			if err == nil {
				results[i] = m["v"]
			}
		}(i)
	}
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()
	assert.Equal(t, int64(2), loads.Load(), "singleflight: one reload for 64 stale readers")
	for _, r := range results {
		assert.Equal(t, int64(1), r, "every reader after the invalidation sees the new content")
	}

	// Fast path under contention with invalidations sprinkled in.
	var wg2 sync.WaitGroup
	stop := make(chan struct{})
	for range 8 {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_, _ = c.Get(ctx)
				}
			}
		}()
	}
	for i := range 20 {
		version.Store(int64(10 + i))
		h.Invalidate("widgets")
		time.Sleep(time.Millisecond)
	}
	close(stop)
	wg2.Wait()
	m, _ := c.Get(ctx)
	assert.Equal(t, int64(29), m["v"])
}

func TestFingerprint_ChangesOnlyWithContent(t *testing.T) {
	h := configcache.NewHubAt(time.Minute, newClock().now)
	val := map[string]string{"a": "1"}
	var mu sync.Mutex
	load := func(context.Context) (map[string]string, error) {
		mu.Lock()
		defer mu.Unlock()
		out := map[string]string{}
		for k, v := range val {
			out[k] = v
		}
		return out, nil
	}
	c := configcache.New("t", []string{"widgets"}, load, configcache.WithHub(h))
	ctx := context.Background()
	empty := h.Fingerprint("widgets")
	_, _ = c.Get(ctx)
	fp1 := h.Fingerprint("widgets")
	assert.NotEqual(t, empty, fp1)
	assert.NotEmpty(t, c.Fingerprint())

	h.Invalidate("widgets") // reload, same content
	_, _ = c.Get(ctx)
	assert.Equal(t, fp1, h.Fingerprint("widgets"), "a reload of identical content keeps the fingerprint")

	mu.Lock()
	val["a"] = "2"
	mu.Unlock()
	h.Invalidate("widgets")
	_, _ = c.Get(ctx)
	fp2 := h.Fingerprint("widgets")
	assert.NotEqual(t, fp1, fp2)
	assert.Equal(t, fp2, h.Fingerprint("widgets", "untouched"), "a table nothing loaded adds nothing")
	assert.Equal(t, h.Fingerprint("a", "widgets"), h.Fingerprint("widgets", "a"), "order-independent")
}

func TestKeyed_OneCachePerKey(t *testing.T) {
	h := configcache.NewHubAt(time.Minute, newClock().now)
	k := configcache.NewKeyed[string]("contents", []string{"message_contents"}, configcache.WithHub(h))
	ctx := context.Background()
	var loads atomic.Int64
	load := func(key string) configcache.Loader[string] {
		return func(context.Context) (string, error) { loads.Add(1); return "rows-" + key, nil }
	}
	for range 3 {
		v, err := k.Get(ctx, "fa", load("fa"))
		require.NoError(t, err)
		assert.Equal(t, "rows-fa", v)
		v, _ = k.Get(ctx, "en", load("en"))
		assert.Equal(t, "rows-en", v)
	}
	assert.Equal(t, int64(2), loads.Load())
	h.Invalidate("message_contents")
	_, _ = k.Get(ctx, "fa", load("fa"))
	assert.Equal(t, int64(3), loads.Load())
}

func BenchmarkGet_Fresh(b *testing.B) {
	h := configcache.NewHubAt(time.Minute, time.Now)
	c := configcache.New("t", []string{"a", "b"}, func(context.Context) (int, error) { return 1, nil }, configcache.WithHub(h))
	ctx := context.Background()
	_, _ = c.Get(ctx)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = c.Get(ctx)
		}
	})
}
