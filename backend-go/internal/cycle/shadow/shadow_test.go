package shadow

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/platform/cache"
)

func mustDay(t *testing.T, y int, m time.Month, d int) engine.Day {
	t.Helper()
	day, ok := engine.MakeDay(y, m, d)
	require.True(t, ok)
	return day
}

// v6Of resolves the §2.35 example history (last start 07-01, 29-day cycles, O = 07-16) on target.
func v6Of(t *testing.T, target, today engine.Day, hormonal bool) engine.Status {
	t.Helper()
	in := engine.Input{Hormonal: hormonal}
	for _, s := range []engine.Day{mustDay(t, 2026, 4, 4), mustDay(t, 2026, 5, 3), mustDay(t, 2026, 6, 1), mustDay(t, 2026, 7, 1)} {
		in.Periods = append(in.Periods, engine.Period{Start: s, End: s + 4})
	}
	return engine.Prepare(in, engine.DefaultParams()).Resolve(target, today)
}

// servedV1 is a served v1 `data` object with the given cycle_view fields over a neutral base.
func servedV1(t *testing.T, cv map[string]any, calc map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"calculation": calc, "cycle_view": cv})
	require.NoError(t, err)
	return raw
}

// asV1 is the v1.1 view of st with no differences (Classify then reports new_fields_only).
func asV1(st *engine.Status) map[string]any {
	day := func(d engine.Day) any {
		if !d.IsSet() {
			return nil
		}
		return d.String()
	}
	opt := func(o engine.Opt[int]) any {
		if v, ok := o.Get(); ok {
			return v
		}
		return nil
	}
	a := &st.Anchors
	return map[string]any{
		"cycle_day": opt(st.CycleDay), "main_phase": st.MainPhase.String(), "subphase": st.Subphase.String(),
		"fertility_level": st.FertilityLevel.String(), "days_to_ovulation": opt(st.DaysToOvulation),
		"days_to_period": opt(st.DaysToPeriod), "days_late": opt(st.DaysLate),
		"anchors": map[string]any{
			"current_period_start": day(a.CurrentPeriodStart), "current_period_start_source": a.CurrentPeriodStartSource.String(),
			"current_period_end": day(a.CurrentPeriodEnd), "current_period_end_source": a.CurrentPeriodEndSource.String(),
			"current_period_end_is_confirmed": a.CurrentPeriodEndIsConfirmed,
			"predicted_next_period_start":     day(a.PredictedNextPeriodStart), "estimated_ovulation_date": day(a.EstimatedOvulationDate),
		},
		"metrics": map[string]any{
			"effective_cycle_length": st.Metrics.EffectiveCycleLength, "effective_period_length": st.Metrics.EffectivePeriodLength,
			"cycle_variability": opt(st.Metrics.CycleVariability),
		},
		"confidence": st.Confidence.String(), "confidence_reasons": reasons(st.ConfidenceReasons),
		"data_quality": st.DataQuality.String(), "resolution_source": st.ResolutionSource.String(),
		"requires_user_input": st.RequiresUserInput, "warnings": warnings(st.Warnings),
	}
}

func TestClassify(t *testing.T) {
	today := mustDay(t, 2026, 7, 18)

	t.Run("identical shared fields → new_fields_only", func(t *testing.T) {
		st := v6Of(t, today, today, false)
		d, err := Classify(servedV1(t, asV1(&st), map[string]any{"cycle_day": 18}), &st)
		require.NoError(t, err)
		assert.Equal(t, []Class{ClassNewFieldsOnly}, d.Classes)
	})
	t.Run("O+1 inside the v1 window O−5..O+1", func(t *testing.T) {
		o1 := mustDay(t, 2026, 7, 17)
		st := v6Of(t, o1, today, false)
		cv := asV1(&st)
		cv["predictions"] = map[string]any{"fertile_window_end": "2026-07-17"}
		d, err := Classify(servedV1(t, cv, nil), &st)
		require.NoError(t, err)
		assert.Equal(t, []Class{ClassOPlusOne}, d.Classes)
		assert.Equal(t, true, d.Fields["v1_window_covers_day"])
	})
	t.Run("late: the legacy calculation wraps", func(t *testing.T) {
		late := mustDay(t, 2026, 8, 9) // next 07-30 → 10 days late
		st := v6Of(t, late, late, false)
		require.Equal(t, 10, st.DaysLate.Or(0))
		d, err := Classify(servedV1(t, asV1(&st), map[string]any{"cycle_day": 11}), &st)
		require.NoError(t, err)
		assert.Contains(t, d.Classes, ClassLateness)
		assert.Equal(t, [2]any{float64(11), 40}, toPair(d.Fields["cycle_day"]))
	})
	t.Run("warnings, confidence, phase, other", func(t *testing.T) {
		st := v6Of(t, today, today, false)
		cv := asV1(&st)
		cv["warnings"] = []string{"prediction_based_output"}
		cv["confidence"] = "medium"
		cv["main_phase"] = "fertile"
		cv["days_to_period"] = 3
		d, err := Classify(servedV1(t, cv, nil), &st)
		require.NoError(t, err)
		assert.Equal(t, []Class{ClassWarnings, ClassConfidence, ClassPhase, ClassOther}, d.Classes)
		assert.Equal(t, []string{"days_to_period"}, d.Fields["other"])
		assert.Equal(t, []string{"prediction_based_output"}, d.Fields["warnings_removed"])
	})
	t.Run("onboarding anchor and suppression", func(t *testing.T) {
		st := v6Of(t, today, today, true)
		cv := asV1(&st)
		cv["fertility_level"] = "low"
		cv["anchors"].(map[string]any)["current_period_start_source"] = "onboarding_declared"
		d, err := Classify(servedV1(t, cv, nil), &st)
		require.NoError(t, err)
		assert.Contains(t, d.Classes, ClassOnboardingAnchor)
		assert.Contains(t, d.Classes, ClassSuppression)
		assert.Equal(t, "hormonal_contraception", d.Fields["suppression_reason"])
	})
	t.Run("garbage v1 is an error", func(t *testing.T) {
		st := v6Of(t, today, today, false)
		_, err := Classify([]byte("{"), &st)
		require.Error(t, err)
	})
}

func toPair(v any) [2]any {
	p := v.([2]any)
	if n, ok := p[0].(*int); ok && n != nil {
		p[0] = float64(*n)
	}
	if n, ok := p[1].(*int); ok && n != nil {
		p[1] = *n
	}
	return p
}

// The pool runs at most PoolSize jobs and drops the rest without blocking; samples hash the user id.
func TestPool_BoundedDropAndSamples(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	rc := cache.NewFromClient(rdb, "ritme-go-shadow-test:")

	today := mustDay(t, 2026, 7, 18)
	st := v6Of(t, today, today, false)
	release := make(chan struct{})
	var running atomic.Int32
	pool := New(func(context.Context, uint64, uint64, engine.Day, engine.Day) (engine.Status, error) {
		running.Add(1)
		<-release
		return st, nil
	}, rc, nil)

	v1 := servedV1(t, asV1(&st), nil)
	start := time.Now()
	accepted := 0
	for range 10 {
		if pool.Submit(Job{UserID: 4242, Target: today, Today: today, V1: v1}) {
			accepted++
		}
	}
	assert.Less(t, time.Since(start), 100*time.Millisecond, "Submit never blocks")
	assert.Equal(t, PoolSize, accepted, "two slots, the rest dropped")
	close(release)
	pool.Wait()
	assert.Equal(t, int32(PoolSize), running.Load())

	raw, err := rdb.LRange(context.Background(), rc.Key(SamplesKey), 0, -1).Result()
	require.NoError(t, err)
	require.Len(t, raw, PoolSize)
	for _, s := range raw {
		assert.NotContains(t, s, "4242")
		var sm Sample
		require.NoError(t, json.Unmarshal([]byte(s), &sm))
		assert.Len(t, sm.User, 16)
		assert.Equal(t, 0, sm.DateOffset)
	}
	assert.Greater(t, mr.TTL(rc.Key(SamplesKey)), 6*24*time.Hour)

	var out strings.Builder
	require.NoError(t, Report(context.Background(), rc, &out, 0))
	assert.Contains(t, out.String(), "2 comparisons")
	assert.Contains(t, out.String(), "new_fields_only          2")
}

// The sample list is capped at MaxSamples.
func TestRecord_CapsSamples(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	rc := cache.NewFromClient(rdb, "ritme-go-shadow-cap:")
	p := New(nil, rc, nil)
	for i := range MaxSamples + 5 {
		require.NoError(t, p.record(context.Background(), Job{UserID: uint64(i)}, Diff{Classes: []Class{ClassOther}})) //nolint:gosec // test ids
	}
	n, err := rdb.LLen(context.Background(), rc.Key(SamplesKey)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(MaxSamples), n)
}
