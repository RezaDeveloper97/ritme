package metrics_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/overview"
	"github.com/ritme/backend-go/internal/metrics"
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

func TestKPI_FunnelCohortsAndSuppression(t *testing.T) {
	e := admintest.New(t)
	// Cohort A (week of Sat 2026-09-12): 12 users. Cohort B (week of 2026-09-19): 5 users, below k.
	addUsers(e, 200, 12, "2026-09-12 10:00:00")
	addUsers(e, 300, 5, "2026-09-19 10:00:00")
	for i := 0; i < 12; i++ {
		event(e, 200+i, "app_open", "2026-09-12")
	}
	for i := 0; i < 8; i++ {
		event(e, 200+i, "onboarding_done", "2026-09-12")
	}
	for i := 0; i < 3; i++ {
		event(e, 200+i, "app_open", "2026-09-13") // D1
	}
	for i := 0; i < 6; i++ {
		event(e, 200+i, "log_saved", "2026-09-19") // D7
	}
	event(e, 300, "app_open", "2026-09-19")

	k, err := metrics.New(e.DB, admintest.Quiet).KPI(t.Context(), now, metrics.KPIOptions{})
	require.NoError(t, err)
	require.NotNil(t, k.EventsSince)
	assert.Equal(t, "2026-09-12", *k.EventsSince)

	steps := map[string]metrics.FunnelStep{}
	for _, s := range k.Funnel.Steps {
		steps[s.Key] = s
	}
	assert.InDelta(t, 100, *steps["signup"].Pct, 0)
	assert.Equal(t, 17, *steps["signup"].Of)
	assert.Equal(t, 13, *steps["app_open"].Count) // 12 + user 300
	assert.Equal(t, 8, *steps["onboarding_done"].Count)
	assert.Equal(t, 6, *steps["first_log"].Count)
	assert.Equal(t, 6, *steps["log_d7"].Count)
	assert.Nil(t, steps["log_d30"].Pct, "nobody is 30 days old yet")

	require.Len(t, k.Retention.Cohorts, 2)
	a, b := k.Retention.Cohorts[0], k.Retention.Cohorts[1]
	assert.Equal(t, "2026-09-12", a.WeekStart)
	require.NotNil(t, a.Size)
	assert.Equal(t, 12, *a.Size)
	assert.InDelta(t, 100, *a.Cells[0], 0) // D0
	assert.InDelta(t, 25, *a.Cells[1], 0)  // D1
	assert.InDelta(t, 50, *a.Cells[2], 0)  // D7 (any event: the 6 loggers)
	assert.Nil(t, a.Cells[5], "D60 not reached")
	assert.True(t, b.Suppressed, "5 users < k")
	assert.Nil(t, b.Size)
	for _, c := range b.Cells {
		assert.Nil(t, c)
	}

	// log_saved only: D0 is 0 % (nobody logged on signup day).
	k, err = metrics.New(e.DB, admintest.Quiet).KPI(t.Context(), now, metrics.KPIOptions{Event: "log_saved"})
	require.NoError(t, err)
	assert.InDelta(t, 0, *k.Retention.Cohorts[0].Cells[0], 0)
	assert.InDelta(t, 50, *k.Retention.Cohorts[0].Cells[2], 0)

	raw, err := json.Marshal(k)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "user_id")
}

func TestKPI_NoEventsYet(t *testing.T) {
	e := admintest.New(t)
	k, err := metrics.New(e.DB, admintest.Quiet).KPI(t.Context(), now, metrics.KPIOptions{})
	require.NoError(t, err)
	assert.Nil(t, k.EventsSince)
	assert.Empty(t, k.Retention.Cohorts)
	for _, s := range k.Funnel.Steps {
		assert.Nil(t, s.Pct)
	}
}

func TestCompute_DAUUsesEvents(t *testing.T) {
	e := admintest.New(t)
	addUsers(e, 400, 3, "2026-09-01 10:00:00")
	for i := 0; i < 3; i++ {
		event(e, 400+i, "app_open", "2026-10-10") // no token, no log: only events
	}
	p, err := metrics.New(e.DB, admintest.Quiet).Compute(t.Context(), now)
	require.NoError(t, err)
	assert.InDelta(t, 3, *p.DAU.Value, 0)
}

func TestKPI_Endpoint(t *testing.T) {
	e := admintest.New(t)
	overview.New(metrics.New(e.DB, admintest.Quiet), admintest.Quiet).Routes(e.Route(), e.Kit)
	assert.Equal(t, 401, e.Anonymous().Get("/kpi").Status)
	r := e.As(admintest.EditorID).Get("/kpi")
	require.Equal(t, 200, r.Status)
	assert.NotNil(t, r.Data()["funnel"])
	assert.Equal(t, 422, e.As(admintest.EditorID).Get("/kpi?days=7").Status)
	assert.Equal(t, 422, e.As(admintest.EditorID).Get("/kpi?event=symptom").Status)
}
