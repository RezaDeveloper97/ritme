package metrics_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/metrics"
	"github.com/ritme/backend-go/internal/metrics/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func TestMAEOf(t *testing.T) {
	// predicted 07-29, actual 08-02 → error_days 4 (signed, actual − predicted); early starts count as |error|.
	m := metrics.MAEOf([]store.ListResolvedFirstPredictionsRow{
		{UserID: 1, ErrorDays: sql.NullInt16{Int16: 4, Valid: true}},
		{UserID: 2, ErrorDays: sql.NullInt16{Int16: -2, Valid: true}},
		{UserID: 2, ErrorDays: sql.NullInt16{}},
	})
	assert.Equal(t, 2, m.Samples)
	assert.Equal(t, 2, m.Users)
	assert.InDelta(t, 3, m.Mean, 1e-9)
	assert.InDelta(t, 50, m.Within2Pct, 1e-9)
}

func TestRUIOf_ResolvedWithinSevenDays(t *testing.T) {
	const mask = uint32(1) << 4
	start := civildate.MustParse("2026-09-20")
	var pts []metrics.SnapshotPoint
	for u := uint64(1); u <= 10; u++ {
		resolve := 6 // users 1..5 clear the warning on day 6
		if u > 5 {
			resolve = 8 // users 6..10 on day 8
		}
		for d := 0; d <= resolve; d++ {
			p := metrics.SnapshotPoint{UserID: u, Day: start.AddDays(d), Warnings: mask, Rev: 2}
			if d == resolve {
				p.Warnings, p.Rev = 0, 3
			}
			pts = append(pts, p)
		}
	}
	r := metrics.RUIOf(pts, mask, start.AddDays(-10), start.AddDays(10))
	require.NotNil(t, r.Pct)
	assert.InDelta(t, 50, *r.Pct, 0)
	assert.Equal(t, 5, r.Count)

	// Cleared without a newer engine_rev (no user input, e.g. the warning aged out) does not count.
	for i := range pts {
		pts[i].Rev = 2
	}
	r = metrics.RUIOf(pts, mask, start.AddDays(-10), start.AddDays(10))
	assert.InDelta(t, 0, *r.Pct, 0)

	// An episode outside [from, to] is not counted; fewer than k users → suppressed.
	r = metrics.RUIOf(pts, mask, start.AddDays(1), start.AddDays(10))
	assert.True(t, r.Suppressed)
	assert.Nil(t, r.Pct)
}

func TestRetentionOf_PlusMinusOneDay(t *testing.T) {
	signupDay := time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)
	today := signupDay.AddDate(0, 0, 9)
	signup := map[uint64]time.Time{}
	var days []metrics.ActivityDay
	for u := uint64(1); u <= 10; u++ {
		signup[u] = signupDay
	}
	days = append(days,
		metrics.ActivityDay{UserID: 1, Offset: 6, Logged: true}, // day 7 − 1
		metrics.ActivityDay{UserID: 2, Offset: 8, Logged: true}, // day 7 + 1
		metrics.ActivityDay{UserID: 3, Offset: 9, Logged: true}, // day 7 + 2: outside
		metrics.ActivityDay{UserID: 4, Offset: 7, Logged: false},
	)
	r := metrics.RetentionOf(today, "log_saved", signup, days)
	require.Len(t, r.Cohorts, 1)
	assert.InDelta(t, 20, *r.Cohorts[0].Cells[2], 0, "users 1 and 2 of 10")
	assert.Nil(t, r.Cohorts[0].Cells[3], "day 14 not reached")
	r = metrics.RetentionOf(today, "any", signup, days)
	assert.InDelta(t, 30, *r.Cohorts[0].Cells[2], 0, "a session-only day counts for «any»")

	// Day 7 needs the whole window (day 8) reached.
	r = metrics.RetentionOf(signupDay.AddDate(0, 0, 7), "log_saved", signup, days)
	assert.Nil(t, r.Cohorts[0].Cells[2])
}

func TestFunnelOf(t *testing.T) {
	facts := make([]metrics.FunnelFacts, 0, 12)
	for i := 0; i < 12; i++ {
		facts = append(facts, metrics.FunnelFacts{Age: 10, OptInOpen: i < 3, Onboarded: i < 9, FirstLog: i < 6, LogD7: i < 4})
	}
	f := metrics.FunnelOf(30, facts)
	got := map[string]metrics.FunnelStep{}
	for _, s := range f.Steps {
		got[s.Key] = s
	}
	assert.Equal(t, 3, *got["install"].Count)
	assert.Equal(t, "opt_in_analytics", got["install"].Source)
	assert.Equal(t, 12, *got["otp_verified"].Count)
	assert.Equal(t, 9, *got["onboarding_done"].Count)
	assert.InDelta(t, 50, *got["first_log"].Pct, 0)
	assert.Equal(t, 4, *got["log_d7"].Count)
	assert.Nil(t, got["log_d30"].Pct, "age 10 < 31")
	assert.False(t, got["log_d30"].Suppressed, "nobody mature: empty, not suppressed")

	small := metrics.FunnelOf(30, facts[:5])
	assert.True(t, small.Steps[1].Suppressed)
	assert.Nil(t, small.Steps[1].Count)
}

func TestAICostShare(t *testing.T) {
	// 150 USD × 600 000 IRR/USD = 90 000 000 IRR over 600 000 000 IRR revenue → 15 %.
	assert.InDelta(t, 15, *metrics.AICostShare(150, 600_000, 600_000_000), 0)
	assert.Nil(t, metrics.AICostShare(150, 0, 600_000_000), "no rate set")
	assert.Nil(t, metrics.AICostShare(150, 600_000, 0), "no revenue")
}

func TestDistributionOf(t *testing.T) {
	d := metrics.DistributionOf("2026-10-11", 20, []string{"high", "medium", "low", "unknown"}, map[string]int{"high": 14, "low": 6})
	require.NotNil(t, d.Of)
	assert.Equal(t, []string{"high", "medium", "low", "unknown"}, []string{d.Buckets[0].Key, d.Buckets[1].Key, d.Buckets[2].Key, d.Buckets[3].Key})
	assert.InDelta(t, 30, *d.Buckets[2].Pct, 0)
	assert.True(t, d.Buckets[2].Suppressed)
	assert.Nil(t, d.Buckets[2].Count)
	assert.Equal(t, 0, *d.Buckets[1].Count)

	small := metrics.DistributionOf("2026-10-11", 9, []string{"high"}, map[string]int{"high": 9})
	assert.True(t, small.Suppressed)
	assert.Empty(t, small.Buckets)
}

func TestMetricValues_V1SnapshotKeepsOldKeys(t *testing.T) {
	v := 120.0
	p := &metrics.Payload{Version: 1, DAU: metrics.Value{Value: &v}, Engine: metrics.Engine{Suppressed: true}}
	got := metrics.MetricValues(p)
	assert.InDelta(t, 120, got["dau"], 0)
	_, ok := got["loggers"]
	assert.False(t, ok, "not in a v1 snapshot")
	for _, k := range metrics.MetricKeys {
		assert.NotEmpty(t, k)
	}
}
