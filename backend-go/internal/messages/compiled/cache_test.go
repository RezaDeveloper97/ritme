package compiled_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

func row(id uint64, prio uint16, title string) store.MessageRule {
	return store.MessageRule{
		ID: id, Level: rules.LevelInfo, Kind: rules.KindContent, Priority: prio, Placement: "daily_advice",
		Content: `{"a":{"title":"` + title + `","body":"b"}}`, Conditions: `[]`, Exceptions: `[]`, AbSplit: 50,
		AudiencePercent: 100, Status: rules.StatusApproved, IsActive: true, Version: 1,
	}
}

// table is a fake message_rules table shared by the "processes" of a test.
type table struct {
	mu    sync.Mutex
	rows  []store.MessageRule
	fail  error
	loads atomic.Int64
}

func (t *table) set(rows ...store.MessageRule) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.rows = rows
}

func (t *table) load(context.Context) ([]store.MessageRule, error) {
	t.loads.Add(1)
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.fail != nil {
		return nil, t.fail
	}
	return append([]store.MessageRule(nil), t.rows...), nil
}

func title(t *testing.T, c *compiled.Cache, tb *table) string {
	t.Helper()
	comp, err := c.Get(context.Background(), tb.load)
	require.NoError(t, err)
	ws := comp.Select("daily_advice", rules.Facts{}, nil, 7, tNow)
	require.NotEmpty(t, ws)
	return ws[0].Rule.Content.A.Title
}

func TestCache_SameProcessSeesTheEditOnTheNextRequest(t *testing.T) {
	h := configcache.NewHub()
	c := compiled.New(configcache.WithHub(h))
	tb := &table{}
	tb.set(row(1, 300, "v1"))
	require.Equal(t, "v1", title(t, c, tb))
	rev := c.Current().Rev
	require.Equal(t, "v1", title(t, c, tb))
	assert.Equal(t, int64(1), tb.loads.Load(), "fresh snapshot: no reload")

	tb.set(row(1, 300, "v2"))
	h.Changed(context.Background(), rules.Table) // the admin write commits
	assert.Equal(t, "v2", title(t, c, tb))
	assert.Greater(t, c.Current().Rev, rev)
}

func TestCache_SecondProcessThroughPubSub(t *testing.T) {
	mr := miniredis.RunT(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	client := func() *cache.Client {
		cl := cache.NewFromClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), "ritme-go:")
		t.Cleanup(func() { _ = cl.Close() })
		return cl
	}
	ha, hb := configcache.NewHub(), configcache.NewHub()
	ca, cb := client(), client()
	ha.Start(ctx, ca, nil)
	hb.Start(ctx, cb, nil)
	inA := compiled.New(configcache.WithHub(ha), configcache.WithTTL(time.Hour))
	inB := compiled.New(configcache.WithHub(hb), configcache.WithTTL(time.Hour))
	tb := &table{}
	tb.set(row(1, 300, "v1"))
	require.Equal(t, "v1", title(t, inA, tb))
	require.Equal(t, "v1", title(t, inB, tb))
	require.Eventually(t, func() bool {
		n, err := ca.Redis().PubSubNumSub(ctx, ca.Key(configcache.Channel)).Result()
		return err == nil && n[ca.Key(configcache.Channel)] >= 2
	}, 5*time.Second, 10*time.Millisecond)

	tb.set(row(1, 300, "v2"))
	ha.Changed(ctx, rules.Table)
	assert.Equal(t, "v2", title(t, inA, tb))
	assert.Eventually(t, func() bool { return title(t, inB, tb) == "v2" }, 5*time.Second, 10*time.Millisecond,
		"second process: via ritme-go:config-changed")
}

func TestCache_TTLWithoutPubSub(t *testing.T) {
	h := configcache.NewHub() // never started: no Redis
	c := compiled.New(configcache.WithHub(h), configcache.WithTTL(40*time.Millisecond))
	tb := &table{}
	tb.set(row(1, 300, "v1"))
	require.Equal(t, "v1", title(t, c, tb))
	tb.set(row(1, 300, "v2")) // written by another process; the message is lost
	assert.Equal(t, "v1", title(t, c, tb))
	assert.Eventually(t, func() bool { return title(t, c, tb) == "v2" }, 2*time.Second, 10*time.Millisecond)
}

func counterValue(t *testing.T, name string) float64 {
	t.Helper()
	mfs, err := metrics.Registry().Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() == name {
			return mf.GetMetric()[0].GetCounter().GetValue()
		}
	}
	t.Fatalf("metric %s not registered", name)
	return 0
}

func TestCache_BrokenRowIsSkippedAndCounted(t *testing.T) {
	c := compiled.New(configcache.WithHub(configcache.NewHub()))
	broken := row(2, 700, "broken")
	broken.Conditions = `[{"op":"and","items":[` // truncated JSON: FromRow would read it as "no conditions"
	tb := &table{}
	tb.set(row(1, 300, "ok"), broken, row(3, 200, "ok3"))
	before := counterValue(t, "messages_rules_compile_errors_total")
	comp, err := c.Get(context.Background(), tb.load)
	require.NoError(t, err)
	require.Len(t, comp.Errors, 1)
	assert.Equal(t, uint64(2), comp.Errors[0].RuleID)
	assert.InDelta(t, before+1, counterValue(t, "messages_rules_compile_errors_total"), 0)
	ws := comp.Select("daily_advice", rules.Facts{}, nil, 7, tNow)
	assert.Equal(t, []win{{1, "a"}, {3, "a"}}, wins(ws), "the other rules are served")
}

func TestCache_SnapshotFailureKeepsThePreviousSnapshot(t *testing.T) {
	h := configcache.NewHub()
	c := compiled.New(configcache.WithHub(h))
	tb := &table{}
	tb.set(row(1, 300, "v1"))
	require.Equal(t, "v1", title(t, c, tb))
	first := c.Current()

	// A whole-snapshot compile failure (duplicate ids)…
	tb.set(row(1, 300, "v2"), row(1, 400, "dup"))
	h.Changed(context.Background(), rules.Table)
	before := counterValue(t, "messages_rules_compile_snapshot_errors_total")
	assert.Equal(t, "v1", title(t, c, tb))
	assert.Same(t, first, c.Current())
	assert.InDelta(t, before+1, counterValue(t, "messages_rules_compile_snapshot_errors_total"), 0)

	// …and a failed load (fresh cache, so the 5 s retry backoff of the first does not hide it).
	c2 := compiled.New(configcache.WithHub(h))
	tb.set(row(1, 300, "v1"))
	require.Equal(t, "v1", title(t, c2, tb))
	tb.fail = errors.New("db down")
	h.Changed(context.Background(), rules.Table)
	assert.Equal(t, "v1", title(t, c2, tb))

	// With no previous snapshot the error reaches the caller.
	_, err := compiled.New(configcache.WithHub(h)).Get(context.Background(), tb.load)
	require.Error(t, err)

	// Every row broken is a snapshot failure too.
	bad := row(5, 300, "x")
	bad.Content = `{`
	_, err = compiled.Compile([]store.MessageRule{bad})
	require.ErrorIs(t, err, compiled.ErrSnapshot)
}

// The served content is the approved snapshot of a head waiting for re-approval (T-M10-02 via rules.ServedRows).
func TestCompile_ServesTheApprovedSnapshot(t *testing.T) {
	head := row(1, 300, "head")
	head.Status, head.Version = rules.StatusPending, 2
	head.ApprovedVersion.Int32, head.ApprovedVersion.Valid = 1, true
	q := snapQuerier{snap: store.ListServedMessageRuleSnapshotsRow{RuleID: 1, Version: 1,
		Snapshot: `{"content":{"a":{"title":"approved","body":"b"}}}`}}
	rows, err := rules.ServedRows(context.Background(), q, []store.MessageRule{head})
	require.NoError(t, err)
	comp, err := compiled.Compile(rows)
	require.NoError(t, err)
	ws := comp.Select("daily_advice", rules.Facts{}, nil, 7, tNow)
	require.Len(t, ws, 1)
	assert.Equal(t, "approved", ws[0].Rule.Content.A.Title)
}

type snapQuerier struct {
	store.Querier
	snap store.ListServedMessageRuleSnapshotsRow
}

func (q snapQuerier) ListServedMessageRuleSnapshots(context.Context) ([]store.ListServedMessageRuleSnapshotsRow, error) {
	return []store.ListServedMessageRuleSnapshotsRow{q.snap}, nil
}

// 50 goroutines select while admin writes swap the snapshot (run with -race).
func TestCache_AtomicSwapUnderConcurrentSelect(t *testing.T) {
	h := configcache.NewHub()
	c := compiled.New(configcache.WithHub(h))
	tb := &table{}
	mk := func(gen int) []store.MessageRule {
		out := make([]store.MessageRule, 0, 20)
		for i := range 20 {
			out = append(out, row(uint64(i+1), uint16(100+i), fmt.Sprintf("g%d", gen))) //nolint:gosec // small
		}
		return out
	}
	tb.set(mk(0)...)
	ctx := context.Background()
	_, err := c.Get(ctx, tb.load)
	require.NoError(t, err)

	stop := make(chan struct{})
	var wg sync.WaitGroup
	var selects atomic.Int64
	for g := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				comp, err := c.Get(ctx, tb.load)
				if err != nil {
					t.Error(err)
					return
				}
				ws := comp.Select("daily_advice", rules.Facts{}, nil, uint64(g), tNow) //nolint:gosec // small
				if len(ws) != 3 || ws[0].Rule.ID != 20 {
					t.Errorf("winners %v", wins(ws))
					return
				}
				selects.Add(1)
			}
		}()
	}
	for gen := 1; gen <= 30; gen++ {
		tb.set(mk(gen)...)
		h.Changed(ctx, rules.Table)
		time.Sleep(2 * time.Millisecond)
	}
	close(stop)
	wg.Wait()
	assert.Positive(t, selects.Load())
	assert.GreaterOrEqual(t, c.Current().Rev, uint64(2))
	assert.Equal(t, "g30", title(t, c, tb))
}
