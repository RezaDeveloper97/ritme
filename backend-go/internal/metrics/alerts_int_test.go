package metrics_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/metrics"
)

func ensureRule(e *admintest.Env, key, cmp, thr string, window int, minVol any) {
	e.Exec("INSERT INTO metric_alert_rules (metric_key, comparator, threshold, window_days, min_volume, enabled, created_at, updated_at) "+
		"VALUES (?, ?, ?, ?, ?, 1, NOW(), NOW()) ON DUPLICATE KEY UPDATE comparator = VALUES(comparator), threshold = VALUES(threshold), "+
		"window_days = VALUES(window_days), min_volume = VALUES(min_volume), enabled = 1", key, cmp, thr, window, minVol)
}

func openCount(e *admintest.Env, key string) int {
	return e.Int("SELECT COUNT(*) FROM metric_alerts a JOIN metric_alert_rules r ON r.id = a.rule_id WHERE r.metric_key = ? AND a.state = 'open'", key)
}

func TestAlerts_SafetyOpensOnceAndResolves(t *testing.T) {
	e := admintest.New(t)
	e.Exec("DELETE FROM metric_alerts")
	ensureRule(e, metrics.RuleSafety, "gt", "0", 1, nil)
	svc := metrics.New(e.DB, admintest.Quiet)

	// No run yet: unknown, nothing opens.
	require.NoError(t, svc.EvaluateAlerts(t.Context(), now))
	assert.Equal(t, 0, openCount(e, metrics.RuleSafety))

	e.Exec("INSERT INTO safety_scenario_runs (run_id, scenario, passed, duration_ms, created_at) VALUES ('RUN1', 's1', 0, 1, ?)", now.Add(-2*time.Hour))
	require.NoError(t, svc.EvaluateAlerts(t.Context(), now))
	require.NoError(t, svc.EvaluateAlerts(t.Context(), now)) // same data: still one
	assert.Equal(t, 1, openCount(e, metrics.RuleSafety))

	p, err := svc.Compute(t.Context(), now)
	require.NoError(t, err)
	require.NoError(t, svc.Refresh(t.Context(), now))
	_ = p
	open, done, err := svc.AlertLists(t.Context())
	require.NoError(t, err)
	require.Len(t, open, 1)
	assert.Equal(t, "critical", open[0].Severity)
	assert.Empty(t, done)

	// A green newer run resolves it.
	e.Exec("INSERT INTO safety_scenario_runs (run_id, scenario, passed, duration_ms, created_at) VALUES ('RUN2', 's1', 1, 1, ?)", now.Add(-time.Hour))
	require.NoError(t, svc.EvaluateAlerts(t.Context(), now))
	assert.Equal(t, 0, openCount(e, metrics.RuleSafety))
	open, done, err = svc.AlertLists(t.Context())
	require.NoError(t, err)
	assert.Empty(t, open)
	require.Len(t, done, 1)
	assert.Equal(t, "resolved", done[0].State)
}

func TestAlerts_CTRDropPerVersionAndMinVolume(t *testing.T) {
	e := admintest.New(t)
	e.Exec("DELETE FROM metric_alerts")
	ensureRule(e, metrics.RuleCTR, "drop_vs_mean", "0.3", 7, 200)
	svc := metrics.New(e.DB, admintest.Quiet)
	// Yesterday is 2026-10-10. Version 1: 7 prior days at 10 % (1000 shown), yesterday 6.9 % (a 31 % drop).
	// Version 2: same drop but only 100 shown yesterday (below min_volume 200).
	row := func(day string, rule, ver, shown, clicked int) {
		e.Exec("INSERT INTO message_event_daily (`day`, rule_id, version, variant, slot, event, `count`) VALUES (?, ?, ?, 'a', 's', 'impression', ?)", day, rule, ver, shown)
		e.Exec("INSERT INTO message_event_daily (`day`, rule_id, version, variant, slot, event, `count`) VALUES (?, ?, ?, 'a', 's', 'click', ?)", day, rule, ver, clicked)
	}
	for d := 3; d <= 9; d++ {
		row(fmt.Sprintf("2026-10-%02d", d), 1, 1, 1000, 100)
		row(fmt.Sprintf("2026-10-%02d", d), 2, 1, 1000, 100)
	}
	row("2026-10-10", 1, 1, 1000, 69)
	row("2026-10-10", 2, 1, 100, 6)
	require.NoError(t, svc.EvaluateAlerts(t.Context(), now))
	require.NoError(t, svc.EvaluateAlerts(t.Context(), now))
	assert.Equal(t, 1, openCount(e, metrics.RuleCTR))
	assert.Equal(t, "r1v1", e.String("SELECT subject FROM metric_alerts WHERE state = 'open'"))

	// Recovery: a later day back at 10 % resolves it.
	row("2026-10-11", 1, 1, 1000, 100)
	require.NoError(t, svc.EvaluateAlerts(t.Context(), now.AddDate(0, 0, 1)))
	assert.Equal(t, 0, openCount(e, metrics.RuleCTR))
}

func TestAlerts_DisabledRuleResolves(t *testing.T) {
	e := admintest.New(t)
	e.Exec("DELETE FROM metric_alerts")
	ensureRule(e, metrics.RuleSafety, "gt", "0", 1, nil)
	svc := metrics.New(e.DB, admintest.Quiet)
	e.Exec("INSERT INTO safety_scenario_runs (run_id, scenario, passed, duration_ms, created_at) VALUES ('RUN9', 's1', 0, 1, ?)", now.Add(-time.Hour))
	require.NoError(t, svc.EvaluateAlerts(t.Context(), now))
	assert.Equal(t, 1, openCount(e, metrics.RuleSafety))
	e.Exec("UPDATE metric_alert_rules SET enabled = 0 WHERE metric_key = ?", metrics.RuleSafety)
	require.NoError(t, svc.EvaluateAlerts(t.Context(), now))
	assert.Equal(t, 0, openCount(e, metrics.RuleSafety))
}
