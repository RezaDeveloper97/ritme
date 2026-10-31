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

// seed writes `users` users, each with one resolved `first` prediction (basis 3 cycles) whose next real start was
// 20 days ago: errors 0, 2, 0, 0 … by user (every 2nd user is 2 days late) → mean ≈ 0.9 at ≤ 12 users, all within ±2.
func seed(e *admintest.Env, users int) {
	for i := 0; i < users; i++ {
		id := 100 + i
		e.Exec("INSERT INTO users (id, mobile, created_at, updated_at) VALUES (?, ?, ?, ?)", id, fmt.Sprintf("0990000%04d", id), "2026-09-01 10:00:00", "2026-09-01 10:00:00")
		errDays := 0
		if i%2 == 1 {
			errDays = 2
		}
		predict(e, id, "2026-08-24", "first", 3, errDays)
	}
}

// predict writes a resolved prediction: cycle_start, predicted next = actual − err, actual = cycle_start + 28.
func predict(e *admintest.Env, uid int, cycleStart, kind string, basis, errDays int) {
	cs := parseDay(cycleStart)
	actual := cs.AddDate(0, 0, 28)
	e.Exec("INSERT INTO engine_prediction_log (user_id, cycle_start, kind, predicted_next, basis_cycles, ovulation_source, confidence, actual_start, error_days, created_at) "+
		"VALUES (?, ?, ?, ?, ?, 'calendar', 'medium', ?, ?, NOW())", uid, cycleStart, kind, actual.AddDate(0, 0, -errDays).Format("2006-01-02"), basis, actual.Format("2006-01-02"), errDays)
}

func parseDay(s string) time.Time { t, _ := time.Parse("2006-01-02", s); return t }

func activity(e *admintest.Env, uid int, day string, session, logged int) {
	e.Exec("INSERT INTO user_activity_days (user_id, `day`, `session`, `logged`) VALUES (?, ?, ?, ?)", uid, day, session, logged)
}

func TestCompute_AggregatesSeededData(t *testing.T) {
	e := admintest.New(t)
	seed(e, 12)
	// Yesterday (10-10): users 100..105 had a session, 103..105 also logged; the day before: user 100 only. A
	// legacy token refresh or daily log alone no longer counts (T-M10-30: sessions come from user_activity_days).
	for i := 0; i < 6; i++ {
		logged := 0
		if i >= 3 {
			logged = 1
		}
		activity(e, 100+i, "2026-10-10", 1, logged)
	}
	activity(e, 100, "2026-10-09", 1, 0)
	e.Exec("INSERT INTO oauth_access_tokens (id, user_id, client_id, revoked, created_at, updated_at) VALUES ('t9', 110, 'c', 0, '2026-10-10 09:00:00', '2026-10-10 09:00:00')")
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

	assert.InDelta(t, 6, *p.DAU.Value, 0) // 100..105 by session (110's token refresh is not a session)
	assert.InDelta(t, 1, *p.DAU.Prev, 0)
	assert.InDelta(t, 3, *p.Loggers.Value, 0)
	assert.True(t, p.Loggers15d.Suppressed, "MAU 6 < k")
	assert.Equal(t, metrics.PayloadVersion, p.Version)
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

	// Engine: one first prediction per user, errors 0 / 2 alternating → mean 1.0, all within ±2.
	assert.False(t, p.Engine.Suppressed)
	assert.Equal(t, 12, p.Engine.Users)
	assert.InDelta(t, 1.0, *p.Engine.MeanErrorDays, 0.001)
	assert.InDelta(t, 100, *p.Engine.Within2DaysPct, 0)
	assert.Equal(t, "healthy", *p.Engine.Status)
	require.NotNil(t, p.Engine.Confidence)
	assert.Nil(t, p.Engine.Confidence.Of, "no engine snapshot yet: no distribution, never faked")
	require.NotNil(t, p.KPI)
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
