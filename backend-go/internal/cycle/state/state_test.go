package state_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/inputs"
	"github.com/ritme/backend-go/internal/cycle/state"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// now is a Tehran noon; the fake store does not filter by day, so it only bounds the marker window.
var now = time.Date(2026, 10, 21, 12, 0, 0, 0, civildate.Tehran)

const uid = 1004

// fakeStore is the four loader queries over in-memory periods. loads counts the database loads (one profile query
// per Load).
type fakeStore struct {
	mu      sync.Mutex
	periods map[uint64][]store.ListEngineInputPeriodsRow
	loads   atomic.Int64
	gate    chan struct{} // when set, the profile query waits for it (stampede test)
}

func newFakeStore() *fakeStore {
	return &fakeStore{periods: map[uint64][]store.ListEngineInputPeriodsRow{}}
}

// setPeriods replaces uid's periods with n 28-day cycles of 5 days ending before now.
func (s *fakeStore) setPeriods(u uint64, n, cycleLen int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := make([]store.ListEngineInputPeriodsRow, 0, n)
	first := civildate.FromTime(now).AddDays(-n * cycleLen)
	for i := range n {
		start := first.AddDays(i * cycleLen)
		rows = append(rows, store.ListEngineInputPeriodsRow{
			UserID: u, PeriodStartDate: start, PeriodEndDate: civildate.NullDate{Date: start.AddDays(4), Valid: true},
			Source: "user_logged", IsConfirmed: true,
		})
	}
	s.periods[u] = rows
}

func (s *fakeStore) ListEngineInputProfiles(_ context.Context, ids []uint64) ([]store.ListEngineInputProfilesRow, error) {
	s.loads.Add(1)
	if s.gate != nil {
		<-s.gate
	}
	out := make([]store.ListEngineInputProfilesRow, 0, len(ids))
	for _, id := range ids {
		out = append(out, store.ListEngineInputProfilesRow{UserID: id, LengthsAuto: true})
	}
	return out, nil
}

func (s *fakeStore) ListEngineInputPeriods(_ context.Context, ids []uint64) ([]store.ListEngineInputPeriodsRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []store.ListEngineInputPeriodsRow
	for _, id := range ids {
		out = append(out, s.periods[id]...)
	}
	return out, nil
}

func (s *fakeStore) ListEngineInputSkips(context.Context, []uint64) ([]store.ListEngineInputSkipsRow, error) {
	return nil, nil
}

func (s *fakeStore) ListEngineInputMarkers(context.Context, store.ListEngineInputMarkersParams) ([]store.ListEngineInputMarkersRow, error) {
	return nil, nil
}

// fakeKV is an in-memory Redis that records every key read and can fail.
type fakeKV struct {
	mu    sync.Mutex
	m     map[string]string
	reads []string
	fail  error
}

func newFakeKV() *fakeKV { return &fakeKV{m: map[string]string{}} }

func (k *fakeKV) Get(_ context.Context, key string) (string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.reads = append(k.reads, key)
	if k.fail != nil {
		return "", k.fail
	}
	v, ok := k.m[key]
	if !ok {
		return "", cache.ErrMiss
	}
	return v, nil
}

func (k *fakeKV) Set(_ context.Context, key string, value any, _ time.Duration) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.fail != nil {
		return k.fail
	}
	switch v := value.(type) {
	case []byte:
		k.m[key] = string(v)
	case string:
		k.m[key] = v
	}
	return nil
}

func (k *fakeKV) put(key, v string) { k.mu.Lock(); k.m[key] = v; k.mu.Unlock() }

func (k *fakeKV) get(key string) string { k.mu.Lock(); defer k.mu.Unlock(); return k.m[key] }

func (k *fakeKV) takeReads() []string {
	k.mu.Lock()
	defer k.mu.Unlock()
	r := k.reads
	k.reads = nil
	return r
}

var p1 = state.Params{Engine: engine.DefaultParams(), FP: "p1"}

// p2 is a clinical-config change: another luteal default, another fingerprint.
var p2 = func() state.Params {
	p := engine.DefaultParams()
	p.Luteal.Default = 12
	return state.Params{Engine: p, FP: "p1:cfg2"}
}()

// newProvider builds a provider with counted Prepare calls; verify follows opts.
func newProvider(t testing.TB, st inputs.Querier, kv state.KV, opts state.Options) (*state.Provider, *atomic.Int64) {
	t.Helper()
	opts.Build = "test"
	pr, err := state.New(st, kv, clock.At(now), opts)
	require.NoError(t, err)
	var prepares atomic.Int64
	state.SetPrepare(pr, func(in engine.Input, p engine.Params) *engine.State {
		prepares.Add(1)
		return engine.Prepare(in, p)
	})
	return pr, &prepares
}

func noVerify() state.Options { o := state.DefaultOptions(); o.Verify = false; return o }

func TestDefaultOptions_VerifyOnInTests(t *testing.T) {
	o := state.DefaultOptions()
	assert.True(t, o.Enabled)
	assert.True(t, o.Verify, "go test runs with verify on")
	assert.Equal(t, 20000, o.L1Entries)
	assert.Equal(t, 1.0, o.VerifySample)
}

// Coherence: a write bumps the revision, the next For misses L1 and L2 and returns the new state; the old
// revision's key is never read again.
func TestFor_RevisionBumpMissesBothLayers(t *testing.T) {
	ctx := context.Background()
	st, kv := newFakeStore(), newFakeKV()
	st.setPeriods(uid, 6, 28)
	pr, _ := newProvider(t, st, kv, noVerify())

	s1, err := pr.For(ctx, uid, 1, p1)
	require.NoError(t, err)
	assert.Len(t, s1.Periods(), 6)
	assert.Equal(t, []string{state.L2Key(uid, 1)}, kv.takeReads())
	again, err := pr.For(ctx, uid, 1, p1)
	require.NoError(t, err)
	assert.Same(t, s1, again, "L1 hit: the shared state")
	assert.Empty(t, kv.takeReads())

	// The write (T-M9-30 bumps engine_rev in its transaction).
	st.setPeriods(uid, 7, 28)
	s2, err := pr.For(ctx, uid, 2, p1)
	require.NoError(t, err)
	assert.Len(t, s2.Periods(), 7, "the new state")
	assert.Equal(t, int64(2), st.loads.Load(), "L2 missed too: reloaded from the database")
	reads := kv.takeReads()
	assert.Equal(t, []string{state.L2Key(uid, 2)}, reads)
	for _, k := range reads {
		assert.False(t, strings.HasSuffix(k, ":1"), "old rev key read: %s", k)
	}
	assert.NotEmpty(t, kv.get(state.L2Key(uid, 2)), "the new snapshot is stored")
	assert.Len(t, s1.Periods(), 6, "the old state is untouched (immutable)")
}

// Stampede: 50 concurrent For calls for one cold user → 1 database load and 1 Prepare.
func TestFor_StampedeIsOneLoadOnePrepare(t *testing.T) {
	st, kv := newFakeStore(), newFakeKV()
	st.setPeriods(uid, 12, 28)
	st.gate = make(chan struct{})
	pr, prepares := newProvider(t, st, kv, noVerify())

	const n = 50
	var wg sync.WaitGroup
	states := make([]*engine.State, n)
	errs := make([]error, n)
	var started sync.WaitGroup
	started.Add(n)
	for i := range n {
		wg.Go(func() {
			started.Done()
			states[i], errs[i] = pr.For(context.Background(), uid, 3, p1)
		})
	}
	started.Wait()
	time.Sleep(20 * time.Millisecond) //nolint:sleep // lets the 50 calls reach the flight; the single-load assertion holds for any scheduling
	close(st.gate)
	wg.Wait()

	for i := range n {
		require.NoError(t, errs[i])
		assert.Same(t, states[0], states[i])
	}
	assert.Equal(t, int64(1), st.loads.Load(), "one database load")
	assert.Equal(t, int64(1), prepares.Load(), "one Prepare")
}

// Redis down → served from the database; corrupt JSON or an unknown snapshot version → reload (and rewrite).
func TestFor_L2FailuresFallBackToTheDatabase(t *testing.T) {
	ctx := context.Background()
	t.Run("redis down", func(t *testing.T) {
		st, kv := newFakeStore(), newFakeKV()
		st.setPeriods(uid, 5, 28)
		kv.fail = errors.New("dial tcp: connection refused")
		pr, _ := newProvider(t, st, kv, noVerify())
		s, err := pr.For(ctx, uid, 1, p1)
		require.NoError(t, err)
		assert.Len(t, s.Periods(), 5)
		assert.Equal(t, int64(1), st.loads.Load())
	})
	for name, raw := range map[string]string{
		"corrupt json":    `{"v":1,"periods":[{"start":`,
		"unknown version": `{"v":99}`,
		"bad date":        `{"v":1,"periods":[{"start":"2026-13-40","end":null,"source":"user_logged"}],"declared":{},"mode":"cycle"}`,
	} {
		t.Run(name, func(t *testing.T) {
			st, kv := newFakeStore(), newFakeKV()
			st.setPeriods(uid, 5, 28)
			kv.put(state.L2Key(uid, 1), raw)
			pr, _ := newProvider(t, st, kv, noVerify())
			s, err := pr.For(ctx, uid, 1, p1)
			require.NoError(t, err)
			assert.Len(t, s.Periods(), 5)
			assert.Equal(t, int64(1), st.loads.Load(), "reloaded from the database")
			_, err = inputs.Decode([]byte(kv.get(state.L2Key(uid, 1))))
			assert.NoError(t, err, "the snapshot is rewritten")
		})
	}
}

// Config change (another params fingerprint) → re-Prepare from the same L2 snapshot, no database.
func TestFor_ParamsChangeReprepareFromL2(t *testing.T) {
	ctx := context.Background()
	st, kv := newFakeStore(), newFakeKV()
	st.setPeriods(uid, 8, 28)
	pr, prepares := newProvider(t, st, kv, noVerify())

	a, err := pr.For(ctx, uid, 4, p1)
	require.NoError(t, err)
	state.PurgeL1(pr) // e.g. another instance, or an evicted entry: only L2 is warm
	_ = kv.takeReads()

	b, err := pr.For(ctx, uid, 4, p2)
	require.NoError(t, err)
	assert.Equal(t, int64(1), st.loads.Load(), "no database load")
	assert.Equal(t, int64(2), prepares.Load())
	assert.Equal(t, []string{state.L2Key(uid, 4)}, kv.takeReads())
	assert.Equal(t, 12, b.Params().Luteal.Default)
	assert.Equal(t, 14, a.Params().Luteal.Default)

	// Without purging, a new fingerprint is an L1 miss and still never reaches the database.
	c, err := pr.For(ctx, uid, 4, state.Params{Engine: engine.DefaultParams(), FP: "p1:cfg3"})
	require.NoError(t, err)
	assert.NotSame(t, a, c)
	assert.Equal(t, int64(1), st.loads.Load())
}

// Verify mode catches an injected stale L1 entry: counted, and the fresh state is served and cached.
func TestFor_VerifyCatchesStaleL1(t *testing.T) {
	ctx := context.Background()
	st, kv := newFakeStore(), newFakeKV()
	st.setPeriods(uid, 6, 28)
	pr, _ := newProvider(t, st, kv, state.DefaultOptions()) // verify on (go test)

	fresh, err := pr.For(ctx, uid, 5, p1)
	require.NoError(t, err)
	before := state.VerifyMismatches()
	_, err = pr.For(ctx, uid, 5, p1) // a verified hit that matches
	require.NoError(t, err)
	assert.Equal(t, before, state.VerifyMismatches())

	stale := engine.Prepare(engine.Input{}, p1.Engine)
	state.InjectL1(pr, uid, 5, p1.FP, stale)
	got, err := pr.For(ctx, uid, 5, p1)
	require.NoError(t, err)
	assert.Equal(t, before+1, state.VerifyMismatches(), "mismatch counted")
	assert.NotSame(t, stale, got)
	assert.Len(t, got.Periods(), 6, "the fresh state is served")
	assert.Len(t, fresh.Periods(), 6)
	again, err := pr.For(ctx, uid, 5, p1)
	require.NoError(t, err)
	assert.Same(t, got, again, "and replaces the stale entry")
	assert.Equal(t, before+1, state.VerifyMismatches())
}

// A verified hit with VerifySample 0 never reloads.
func TestFor_VerifySample(t *testing.T) {
	ctx := context.Background()
	st := newFakeStore()
	st.setPeriods(uid, 3, 28)
	o := state.DefaultOptions()
	o.VerifySample = 0
	pr, _ := newProvider(t, st, nil, o)
	for range 5 {
		_, err := pr.For(ctx, uid, 1, p1)
		require.NoError(t, err)
	}
	assert.Equal(t, int64(1), st.loads.Load())
}

// CYCLE_ENGINE_CACHE=off bypasses both layers.
func TestFor_Disabled(t *testing.T) {
	ctx := context.Background()
	st, kv := newFakeStore(), newFakeKV()
	st.setPeriods(uid, 4, 28)
	o := state.DefaultOptions()
	o.Enabled = false
	pr, prepares := newProvider(t, st, kv, o)
	a, err := pr.For(ctx, uid, 1, p1)
	require.NoError(t, err)
	b, err := pr.For(ctx, uid, 1, p1)
	require.NoError(t, err)
	assert.NotSame(t, a, b)
	assert.Equal(t, int64(2), st.loads.Load())
	assert.Equal(t, int64(2), prepares.Load())
	assert.Empty(t, kv.takeReads())
	assert.Empty(t, kv.get(state.L2Key(uid, 1)))
}

// A state from L2 equals the one prepared straight from the database (the loader output is normalized through the
// snapshot format), so verify never flags an L2 hit falsely.
func TestFor_L2AndDBStatesAreEqual(t *testing.T) {
	ctx := context.Background()
	st, kv := newFakeStore(), newFakeKV()
	st.setPeriods(uid, 10, 29)
	pr, _ := newProvider(t, st, kv, state.DefaultOptions())
	_, err := pr.For(ctx, uid, 1, p1)
	require.NoError(t, err)
	state.PurgeL1(pr)
	before := state.VerifyMismatches()
	_, err = pr.For(ctx, uid, 1, p1) // L2 hit, verified against the database
	require.NoError(t, err)
	assert.Equal(t, before, state.VerifyMismatches())
}

func TestNew_RejectsBadSample(t *testing.T) {
	o := state.DefaultOptions()
	o.VerifySample = 1.5
	_, err := state.New(newFakeStore(), nil, nil, o)
	require.Error(t, err)
}
