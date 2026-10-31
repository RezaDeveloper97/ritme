package metrics_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/overview"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/metrics"
	"github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	platformmetrics "github.com/ritme/backend-go/internal/platform/metrics"
)

func addUsers(e *admintest.Env, from, n int, created string) {
	for i := 0; i < n; i++ {
		id := from + i
		e.Exec("INSERT INTO users (id, mobile, created_at, updated_at) VALUES (?, ?, ?, ?)", id, fmt.Sprintf("0991000%04d", id), created, created)
	}
}

func event(e *admintest.Env, uid int, name, day string) {
	e.Exec("INSERT INTO analytics_events (user_id, name, occurred_on, occurred_at, created_at) VALUES (?, ?, ?, ?, NOW())", uid, name, day, day+" 10:00:00")
}

func snapshot(e *admintest.Env, uid int, day, mode, confidence, quality string, warnings uint32, rev int) {
	e.Exec("INSERT INTO engine_daily_snapshots (user_id, `day`, main_phase, subphase, confidence, data_quality, warnings_mask, mode, engine_rev, created_at) "+
		"VALUES (?, ?, 'luteal', 'mid_luteal', ?, ?, ?, ?, ?, NOW())", uid, day, confidence, quality, warnings, mode, rev)
}

func refreshed(t *testing.T, e *admintest.Env) *metrics.Payload {
	t.Helper()
	svc := metrics.New(e.DB, admintest.Quiet)
	require.NoError(t, svc.Refresh(t.Context(), now))
	var p metrics.Payload
	require.NoError(t, json.Unmarshal([]byte(e.String("SELECT payload FROM metric_snapshots ORDER BY `day` DESC LIMIT 1")), &p))
	require.NotNil(t, p.KPI)
	return &p
}

// Funnel from server facts: 12 signups on 2026-09-12 (29 days before `now`), 8 onboarded, 6 logged, 5 of them on day
// 7 ±1; install from opt-in analytics; day 30 not mature.
func TestKPI_FunnelAndRetentionFromServerFacts(t *testing.T) {
	e := admintest.New(t)
	addUsers(e, 200, 12, "2026-09-12 10:00:00")
	addUsers(e, 300, 5, "2026-09-19 10:00:00") // a second, small cohort
	for i := 0; i < 4; i++ {
		event(e, 200+i, "app_open", "2026-09-12")
	}
	for i := 0; i < 8; i++ {
		e.Exec("INSERT INTO user_life_profiles (user_id, life_mode, onboarding_completed_at, created_at, updated_at) VALUES (?, 'cycle', '2026-09-12 11:00:00', NOW(), NOW())", 200+i)
	}
	for i := 0; i < 12; i++ {
		activity(e, 200+i, "2026-09-12", 1, 0) // D0 session
	}
	for i := 0; i < 6; i++ {
		day := "2026-09-19" // day 7
		if i == 5 {
			day = "2026-09-25" // day 13: outside day 7 ±1, inside day 14 ±1
		}
		activity(e, 200+i, day, 1, 1)
	}
	activity(e, 200, "2026-09-13", 1, 0) // D1 session only
	activity(e, 201, "2026-09-18", 1, 1) // day 6: also day 7 ±1 for 201 (already counted)

	p := refreshed(t, e)
	f := p.KPI.Funnels["30"]
	steps := map[string]metrics.FunnelStep{}
	for _, s := range f.Steps {
		steps[s.Key] = s
	}
	assert.Equal(t, "opt_in_analytics", steps["install"].Source)
	assert.Equal(t, 4, *steps["install"].Count)
	assert.Equal(t, 17, *steps["otp_verified"].Count)
	assert.InDelta(t, 100, *steps["otp_verified"].Pct, 0)
	assert.Equal(t, 8, *steps["onboarding_done"].Count)
	assert.Equal(t, 6, *steps["first_log"].Count)
	assert.Equal(t, 5, *steps["log_d7"].Count, "users 200..204 logged within day 7 ±1")
	assert.Equal(t, 17, *steps["log_d7"].Of, "both cohorts are ≥ 8 days old")
	assert.Nil(t, steps["log_d30"].Pct, "nobody is 31 days old yet")

	r := p.KPI.Retention["log_saved"]
	require.Len(t, r.Cohorts, 2)
	a := r.Cohorts[0]
	assert.Equal(t, "2026-09-12", a.WeekStart)
	assert.InDelta(t, 0, *a.Cells[0], 0, "D0 (±1: days 0..1): no log")
	assert.InDelta(t, 100*5.0/12, *a.Cells[2], 0.1, "D7 ±1")
	assert.InDelta(t, 100*1.0/12, *a.Cells[3], 0.1, "D14 ±1 catches the day-13 log")
	assert.True(t, r.Cohorts[1].Suppressed)
	anyRow := p.KPI.Retention["any"].Cohorts[0]
	assert.InDelta(t, 100, *anyRow.Cells[0], 0, "every member had a session on day 0")

	// The endpoint picks the variant and reads ≤ 2 queries.
	k, err := metrics.New(e.DB, admintest.Quiet).KPI(t.Context(), now, metrics.KPIOptions{Days: 30, Event: "log_saved"})
	require.NoError(t, err)
	assert.Equal(t, "log_saved", k.Retention.Event)
	assert.Equal(t, 30, k.Funnel.Days)
	raw, err := json.Marshal(k)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "user_id")
}

// MAE fixture (A-05): first prediction 07-29 vs actual 08-02 → error 4; a pregnancy cycle, a hormonal cycle and a
// basis-1 prediction are excluded; `last` rows are not used.
func TestKPI_EngineMAEFromPredictionLog(t *testing.T) {
	e := admintest.New(t)
	addUsers(e, 500, 13, "2026-06-01 10:00:00")
	for i := 0; i < 10; i++ {
		e.Exec("INSERT INTO engine_prediction_log (user_id, cycle_start, kind, predicted_next, basis_cycles, ovulation_source, confidence, actual_start, error_days, created_at) "+
			"VALUES (?, '2026-07-04', 'first', '2026-07-29', 2, 'calendar', 'medium', '2026-08-02', 4, NOW())", 500+i)
		e.Exec("INSERT INTO engine_prediction_log (user_id, cycle_start, kind, predicted_next, basis_cycles, ovulation_source, confidence, actual_start, error_days, created_at) "+
			"VALUES (?, '2026-07-04', 'last', '2026-08-02', 2, 'calendar', 'medium', '2026-08-02', 0, NOW())", 500+i)
	}
	predictRow := func(uid, basis, errDays int) {
		e.Exec("INSERT INTO engine_prediction_log (user_id, cycle_start, kind, predicted_next, basis_cycles, ovulation_source, confidence, actual_start, error_days, created_at) "+
			"VALUES (?, '2026-07-04', 'first', '2026-07-20', ?, 'calendar', 'low', '2026-08-02', ?, NOW())", uid, basis, errDays)
	}
	predictRow(510, 3, 13)
	snapshot(e, 510, "2026-07-15", "pregnancy", "unknown", "insufficient", 0, 3) // pregnant during the cycle
	predictRow(511, 3, 13)
	snapshot(e, 511, "2026-07-10", "cycle", "low", "partial", uint32(1)<<engine.WarnHormonalContraceptionActive, 3)
	predictRow(512, 1, 13) // built from one real cycle

	p := refreshed(t, e)
	require.False(t, p.Engine.Suppressed)
	assert.Equal(t, 10, p.Engine.Samples)
	assert.InDelta(t, 4, *p.Engine.MeanErrorDays, 0)
	assert.InDelta(t, 0, *p.Engine.Within2DaysPct, 0)
	assert.Equal(t, "watch", *p.Engine.Status)
	v := metrics.MetricValues(p)
	assert.InDelta(t, 4, v["engine_mae"], 0)
	assert.InDelta(t, 0, v["accuracy_2d"], 0)
}

// Distributions over the newest snapshot day and the requires_user_input episodes (resolved by an input on day 6
// counts, on day 8 does not).
func TestKPI_DistributionsAndRUI(t *testing.T) {
	e := admintest.New(t)
	addUsers(e, 600, 20, "2026-06-01 10:00:00")
	rui := uint32(1) << engine.WarnInsufficientAnchorData
	for i := 0; i < 20; i++ {
		conf := "high"
		if i < 4 {
			conf = "low"
		}
		snapshot(e, 600+i, "2026-10-11", "cycle", conf, "good", 0, 9)
		// Episode starts 2026-09-20; users 600..609 clear it on day 6 with a write (rev 2 → 3), 610..619 on day 8.
		resolve := 6
		if i >= 10 {
			resolve = 8
		}
		for d := 0; d <= resolve; d++ {
			day := time.Date(2026, 9, 20+d, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
			if d < resolve {
				snapshot(e, 600+i, day, "cycle", "low", "poor", rui, 2)
			} else {
				snapshot(e, 600+i, day, "cycle", "medium", "partial", 0, 3)
			}
		}
	}
	p := refreshed(t, e)
	c := p.KPI.Distributions.Confidence
	require.NotNil(t, c.Of)
	assert.Equal(t, 20, *c.Of)
	assert.Equal(t, "2026-10-11", *c.Day)
	assert.InDelta(t, 20, metrics.MetricValues(p)["low_confidence_share"], 0)
	for _, b := range c.Buckets {
		if b.Key == "low" {
			assert.True(t, b.Suppressed, "4 users < k: share kept, count hidden")
			assert.Nil(t, b.Count)
		}
	}
	require.NotNil(t, p.KPI.RUI.Pct)
	assert.InDelta(t, 50, *p.KPI.RUI.Pct, 0)
	assert.Equal(t, 20, *p.KPI.RUI.Of)
}

// AI cost / Plus revenue with an admin-set FX rate; without the rate the share is null.
func TestKPI_AICostShare(t *testing.T) {
	e := admintest.New(t)
	addUsers(e, 700, 10, "2026-06-01 10:00:00")
	for i := 0; i < 10; i++ {
		e.Exec("INSERT INTO ai_usage_logs (user_id, feature, op, provider, model, cost_micros, ok, created_at) VALUES (?, 'lab_analysis', 'x', 'p', 'm', 1000000, 1, '2026-10-05 10:00:00')", 700+i)
		e.Exec("INSERT INTO plus_invoices (reference, user_id, duration_months, status, subtotal_rials, discount_rials, vat_rate_bps, vat_rials, total_rials, expires_at, paid_at, created_at, updated_at) "+
			"VALUES (?, ?, 1, 'paid', 11000000, 1000000, 1000, 1000000, 11000000, '2026-12-01 00:00:00', '2026-10-05 10:00:00', NOW(), NOW())", fmt.Sprintf("R%d", i), 700+i)
	}
	p := refreshed(t, e)
	assert.Nil(t, p.KPI.AICost.SharePct, "no FX rate set")
	assert.InDelta(t, 10, *p.KPI.AICost.CostUSD, 0)

	e.Exec("INSERT INTO plus_settings (`key`, `value`, created_at, updated_at) VALUES (?, '1000000', NOW(), NOW())", metrics.FXRateSetting)
	p = refreshed(t, e)
	// 10 USD × 1 000 000 IRR/USD = 10 000 000 IRR over 100 000 000 IRR revenue → 10 %.
	require.NotNil(t, p.KPI.AICost.SharePct)
	assert.InDelta(t, 10, *p.KPI.AICost.SharePct, 0)
	assert.InDelta(t, 10, metrics.MetricValues(p)["ai_cost_share"], 0)
}

// The KPI endpoint reads two snapshots (latest + previous period) and nothing else; deltas compare them.
func TestKPI_EndpointReadsTwoSnapshots(t *testing.T) {
	e := admintest.New(t)
	seed(e, 12)
	svc := metrics.New(e.DB, admintest.Quiet)
	require.NoError(t, svc.Refresh(t.Context(), now.AddDate(0, 0, -30)))
	activity(e, 100, "2026-10-10", 1, 1)
	require.NoError(t, svc.Refresh(t.Context(), now))

	counted, err := db.OpenDSN(t.Context(), testdb.DSN(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = counted.Close() })
	ctx, q := platformmetrics.WithQueryCounter(context.Background())
	k, err := metrics.New(counted, admintest.Quiet).KPI(ctx, now, metrics.KPIOptions{Days: 30})
	require.NoError(t, err)
	assert.LessOrEqual(t, q.Load(), int64(2))
	require.NotNil(t, k.Previous)
	assert.Equal(t, "2026-09-11", *k.Previous)
	d := k.Deltas["dau"]
	require.NotNil(t, d.Delta)
	assert.InDelta(t, 1, *d.Delta, 0)
	assert.Contains(t, k.Deltas, "message_ctr")
	assert.Nil(t, k.Family)

	overview.New(svc, admintest.Quiet).Routes(e.Route(), e.Kit)
	assert.Equal(t, 401, e.Anonymous().Get("/kpi").Status)
	r := e.As(admintest.EditorID).Get("/kpi")
	require.Equal(t, 200, r.Status)
	assert.NotNil(t, r.Data()["funnel"])
	assert.Equal(t, 422, e.As(admintest.EditorID).Get("/kpi?days=7").Status)
	assert.Equal(t, 422, e.As(admintest.EditorID).Get("/kpi?event=symptom").Status)
}

func TestKPI_NoSnapshotYet(t *testing.T) {
	e := admintest.New(t)
	k, err := metrics.New(e.DB, admintest.Quiet).KPI(t.Context(), now, metrics.KPIOptions{})
	require.NoError(t, err)
	assert.Nil(t, k.Day)
	assert.Empty(t, k.Retention.Cohorts)
	assert.Empty(t, k.Funnel.Steps)
}
