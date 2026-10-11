package metrics_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/overview"
	"github.com/ritme/backend-go/internal/metrics"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

var now = time.Date(2026, 10, 11, 12, 0, 0, 0, civildate.Tehran)

func seed(e *admintest.Env, users int) {
	for i := 0; i < users; i++ {
		id := 100 + i
		e.Exec("INSERT INTO users (id, mobile, created_at, updated_at) VALUES (?, ?, ?, ?)", id, fmt.Sprintf("0990000%04d", id), "2026-09-01 10:00:00", "2026-09-01 10:00:00")
		// Five confirmed cycles, 28 days apart; the 4th is 30 days long.
		start := now.AddDate(0, 0, -150)
		for c, l := range []int{28, 28, 28, 30, 28} {
			e.Exec("INSERT INTO cycle_histories (user_id, period_start_date, cycle_length, is_confirmed, is_estimated, created_at, updated_at) VALUES (?, ?, ?, 1, 0, NOW(), NOW())",
				id, start.AddDate(0, 0, 28*c).Format("2006-01-02"), l)
		}
	}
}

func TestCompute_AggregatesSeededData(t *testing.T) {
	e := admintest.New(t)
	seed(e, 12)
	// Yesterday (10-10): users 100..104 refreshed a token, 103/104 also logged; the day before: user 100 only.
	for i := 0; i < 5; i++ {
		e.Exec("INSERT INTO oauth_access_tokens (id, user_id, client_id, revoked, created_at, updated_at) VALUES (?, ?, 'c', 0, ?, ?)", fmt.Sprintf("t%d", i), 100+i, "2026-10-10 09:00:00", "2026-10-10 09:00:00")
	}
	e.Exec("INSERT INTO oauth_access_tokens (id, user_id, client_id, revoked, created_at, updated_at) VALUES ('t9', 100, 'c', 0, '2026-10-09 09:00:00', '2026-10-09 09:00:00')")
	for _, u := range []int{103, 104, 105} {
		e.Exec("INSERT INTO daily_health_logs (user_id, log_date, created_at, updated_at) VALUES (?, '2026-10-10', NOW(), NOW())", u)
	}
	e.Exec("UPDATE users SET created_at = '2026-10-11 08:00:00' WHERE id IN (100, 101)")
	// Modes: 12 default cycle users; 10 of them ttc.
	for i := 0; i < 10; i++ {
		e.Exec("INSERT INTO user_life_profiles (user_id, life_mode, created_at, updated_at) VALUES (?, 'ttc', NOW(), NOW())", 100+i)
	}
	// Plus: one purchased 3-month plan, 1 200 000 rials → 40 000 toman / month.
	e.Exec("INSERT INTO plus_invoices (reference, user_id, duration_months, status, subtotal_rials, discount_rials, vat_rate_bps, vat_rials, total_rials, expires_at, created_at, updated_at) VALUES ('R1', 100, 3, 'paid', 1200000, 0, 1000, 120000, 1320000, '2026-12-01 00:00:00', NOW(), NOW())")
	inv := e.Int("SELECT id FROM plus_invoices WHERE reference = 'R1'")
	e.Exec("INSERT INTO plus_subscriptions (user_id, invoice_id, status, source, starts_at, ends_at, created_at, updated_at) VALUES (100, ?, 'active', 'purchase', '2026-10-01 00:00:00', '2027-01-01 00:00:00', NOW(), NOW())", inv)
	e.Exec("INSERT INTO plus_trials (user_id, started_at, ends_at, created_at, updated_at) VALUES (101, '2026-09-20 00:00:00', '2026-09-27 00:00:00', NOW(), NOW())")

	svc := metrics.New(e.DB, admintest.Quiet)
	p, err := svc.Compute(t.Context(), now)
	require.NoError(t, err)

	assert.InDelta(t, 6, *p.DAU.Value, 0) // 100..104 by token, 105 by log
	assert.InDelta(t, 1, *p.DAU.Prev, 0)
	assert.InDelta(t, 6, *p.MAU.Value, 0)
	assert.InDelta(t, 100, *p.Stickiness.Value, 0)
	assert.InDelta(t, 2, *p.SignupsToday.Value, 0)
	assert.InDelta(t, 1, *p.PlusActive.Value, 0)
	require.NotNil(t, p.MRRToman)
	assert.EqualValues(t, 40000, *p.MRRToman)
	assert.True(t, p.TrialToPaid.Suppressed, "1 trial is below k=10")
	assert.Nil(t, p.TrialToPaid.Pct)

	modes := map[string]metrics.ModeCount{}
	for _, m := range p.Modes {
		modes[m.Mode] = m
	}
	require.NotNil(t, modes["ttc"].Count)
	assert.Equal(t, 10, *modes["ttc"].Count)
	assert.True(t, modes["cycle"].Suppressed, "2 users < k")
	assert.Nil(t, modes["cycle"].Count)
	assert.True(t, modes["companion"].Suppressed)

	require.Len(t, p.Series, metrics.SeriesDays)
	last := p.Series[len(p.Series)-2]
	assert.Equal(t, "2026-10-10", last.Date)
	assert.Equal(t, 6, last.DAU)
	assert.Equal(t, 3, last.Loggers)

	// Engine: rows 3-5 of 12 users each; errors 0, 2, 0 → mean 0.7, all within ±2.
	assert.False(t, p.Engine.Suppressed)
	assert.Equal(t, 12, p.Engine.Users)
	assert.InDelta(t, 0.7, *p.Engine.MeanErrorDays, 0.001)
	assert.InDelta(t, 100, *p.Engine.Within2DaysPct, 0)
	assert.Equal(t, "healthy", *p.Engine.Status)
	assert.Nil(t, p.Engine.Confidence, "confidence is not stored: never faked")
	assert.Empty(t, p.Alerts)
	assert.Nil(t, p.OKR)

	// No per-user data in the stored document.
	raw, err := json.Marshal(p)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "user_id")
}

func TestEngine_SmallCohortSuppressed(t *testing.T) {
	e := admintest.New(t)
	seed(e, 9)
	p, err := metrics.New(e.DB, admintest.Quiet).Compute(t.Context(), now)
	require.NoError(t, err)
	assert.True(t, p.Engine.Suppressed)
	assert.Nil(t, p.Engine.MeanErrorDays)
	assert.Zero(t, p.Engine.Users)
}

func TestOverview_EndpointAndPermissions(t *testing.T) {
	e := admintest.New(t)
	svc := metrics.New(e.DB, admintest.Quiet)
	overview.New(svc, admintest.Quiet).Routes(e.Route(), e.Kit)

	assert.Equal(t, 401, e.Anonymous().Get("/overview").Status)
	r := e.As(admintest.EditorID).Get("/overview")
	require.Equal(t, 200, r.Status)
	assert.Nil(t, r.Data()["metrics"], "no snapshot before the first run")

	require.NoError(t, svc.Refresh(t.Context(), now))
	require.NoError(t, svc.Refresh(t.Context(), now), "refresh is an upsert for the day")
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM metric_snapshots"))
	r = e.As(admintest.EditorID).Get("/overview")
	require.Equal(t, 200, r.Status)
	assert.Equal(t, "2026-10-11", r.Data()["day"])
	m := r.Obj("metrics")
	require.NotNil(t, m)
	assert.Len(t, m["series"], metrics.SeriesDays)
}
