package state_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/state"
)

// Budget (T-M9-31): a warm For (L1 hit) ≤ 2 µs.
func BenchmarkFor_L1Hit(b *testing.B) {
	st := newFakeStore()
	st.setPeriods(uid, 24, 28)
	pr, _ := newProvider(b, st, newFakeKV(), noVerify())
	ctx := context.Background()
	_, err := pr.For(ctx, uid, 1, p1)
	require.NoError(b, err)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := pr.For(ctx, uid, 1, p1); err != nil {
			b.Fatal(err)
		}
	}
}

// Budget (T-M9-31): an L2 hit + Prepare of 24 periods ≤ 400 µs. In-memory L2 (decode + Prepare + the cache
// machinery); BenchmarkFor_L2HitPrepare24_Redis adds the real Redis round trip.
func BenchmarkFor_L2HitPrepare24(b *testing.B) {
	benchL2Hit(b, newFakeKV())
}

// benchL2Hit alternates two params fingerprints over a one-entry L1, so every call misses L1 and reads L2.
func benchL2Hit(b *testing.B, kv state.KV) {
	st := newFakeStore()
	st.setPeriods(uid, 24, 28)
	o := noVerify()
	o.L1Entries = 1
	pr, _ := newProvider(b, st, kv, o)
	ctx := context.Background()
	pa := state.Params{Engine: engine.DefaultParams(), FP: "a"}
	pb := state.Params{Engine: engine.DefaultParams(), FP: "b"}
	s, err := pr.For(ctx, uid, 1, pa)
	require.NoError(b, err)
	require.Len(b, s.Periods(), 24)
	loads := st.loads.Load()
	b.ReportAllocs()
	b.ResetTimer()
	i := 0
	for b.Loop() {
		p := pa
		if i%2 == 0 {
			p = pb
		}
		i++
		if _, err := pr.For(ctx, uid, 1, p); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	if st.loads.Load() != loads {
		b.Fatalf("database reached: %d loads", st.loads.Load()-loads)
	}
}
