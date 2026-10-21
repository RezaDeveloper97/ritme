package vitals_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-M9-05: vitals in pregnancy use the pregnancy table and hand the reading to the pregnancy alert engine.

// pregnant makes the user pregnant: LMP 2026-04-01 → 26+6 on the fixed clock (2026-10-06).
func (e *env) pregnant(t *testing.T, user uint64) {
	t.Helper()
	_, err := e.db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2026-04-01', NOW(), NOW())`, user)
	require.NoError(t, err)
}

type pregAlert struct{ level, typ, trigger string }

func (e *env) pregnancyAlerts(t *testing.T, user uint64) []pregAlert {
	t.Helper()
	rows, err := e.db.Query(`SELECT alert_level, alert_type, trigger_symptoms FROM pregnancy_alerts WHERE user_id = ? ORDER BY id`, user)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var out []pregAlert
	for rows.Next() {
		var a pregAlert
		require.NoError(t, rows.Scan(&a.level, &a.typ, &a.trigger))
		out = append(out, a)
	}
	require.NoError(t, rows.Err())
	return out
}

func TestVitals_PregnancySevereBP(t *testing.T) {
	e := setup(t)
	sara, tok := e.user(t, "09120000001")
	e.pregnant(t, sara)

	body := map[string]any{"type": "bp", "systolic": 165, "diastolic": 112, "measured_at": "2026-10-06 08:10"}
	r := e.do(t, http.MethodPost, "/api/v1/vitals/readings", tok, "en", body)
	require.Equal(t, http.StatusCreated, r.status, r.body)
	reading := r.data()["reading"].(map[string]any)
	assert.Equal(t, "crisis", reading["classification"].(map[string]any)["code"], "the pregnancy severe range")
	assert.Equal(t, true, reading["urgent"])
	alert, ok := r.data()["alert"].(map[string]any)
	require.True(t, ok, r.body)
	assert.Equal(t, "pregnancy_bp_high", alert["rule"])
	assert.Equal(t, "urgent", alert["level"])
	assert.Equal(t, "Your blood pressure is in the severe range for pregnancy", alert["title"])
	assert.Contains(t, alert["what_we_saw"], "165/112")
	actions := alert["actions"].([]any)
	require.NotEmpty(t, actions)
	assert.Equal(t, "115", actions[0].(map[string]any)["phone"])

	got := e.pregnancyAlerts(t, sara)
	require.Len(t, got, 1, "one alert row, never two")
	assert.Equal(t, "emergency", got[0].level)
	assert.Equal(t, "v2:bp_high", got[0].typ)

	// The same reading again: still the modal, still one row.
	r = e.do(t, http.MethodPost, "/api/v1/vitals/readings", tok, "fa", body)
	require.Equal(t, http.StatusCreated, r.status, r.body)
	alert = r.data()["alert"].(map[string]any)
	assert.Contains(t, alert["contact"], "۱۱۵")
	assert.Len(t, e.pregnancyAlerts(t, sara), 1)

	// 139/89 in pregnancy: normal, nothing raised (non-pregnant ACC/AHA would say stage 1).
	r = e.do(t, http.MethodPost, "/api/v1/vitals/readings", tok, "en", map[string]any{"type": "bp", "systolic": 139, "diastolic": 89})
	assert.Equal(t, "normal", r.data()["reading"].(map[string]any)["classification"].(map[string]any)["code"])
	assert.Nil(t, r.data()["alert"])
	// 140/80 → the high tier (urgent level from pregnancy.bp.high.urgent = 1).
	r = e.do(t, http.MethodPost, "/api/v1/vitals/readings", tok, "en", map[string]any{"type": "bp", "systolic": 140, "diastolic": 80})
	assert.Equal(t, "stage2", r.data()["reading"].(map[string]any)["classification"].(map[string]any)["code"])
	assert.Equal(t, "pregnancy_bp_high", r.data()["alert"].(map[string]any)["rule"])
	assert.Len(t, e.pregnancyAlerts(t, sara), 2)
}

func TestVitals_NotPregnantUnchanged(t *testing.T) {
	e := setup(t)
	sara, tok := e.user(t, "09120000001")
	r := e.do(t, http.MethodPost, "/api/v1/vitals/readings", tok, "en", map[string]any{"type": "bp", "systolic": 160, "diastolic": 100})
	require.Equal(t, http.StatusCreated, r.status, r.body)
	assert.Equal(t, "stage2", r.data()["reading"].(map[string]any)["classification"].(map[string]any)["code"])
	assert.Nil(t, r.data()["alert"])
	assert.Empty(t, e.pregnancyAlerts(t, sara))
}

func TestVitals_PregnancyGDM(t *testing.T) {
	e := setup(t)
	sara, tok := e.user(t, "09120000001")
	e.pregnant(t, sara)
	post := func(v float64, ctx string) map[string]any {
		r := e.do(t, http.MethodPost, "/api/v1/vitals/readings", tok, "en", map[string]any{
			"type": "glucose", "value": v, "unit": "mg_dl", "context": ctx, "method": "glucometer"})
		require.Equal(t, http.StatusCreated, r.status, r.body)
		assert.Nil(t, r.data()["alert"], "follow_up: no urgent modal")
		return r.data()["reading"].(map[string]any)["classification"].(map[string]any)
	}
	assert.Equal(t, "in_range", post(119, "after_meal")["code"])
	assert.Empty(t, e.pregnancyAlerts(t, sara))
	assert.Equal(t, "high", post(120, "after_meal")["code"], "2 h ≥ 120")
	got := e.pregnancyAlerts(t, sara)
	require.Len(t, got, 1)
	assert.Equal(t, "warning", got[0].level)
	assert.Equal(t, "v2:sugar_high", got[0].typ)
	assert.Equal(t, "in_range", post(94, "fasting")["code"])
	assert.Equal(t, "high", post(95, "fasting")["code"])
	assert.Len(t, e.pregnancyAlerts(t, sara), 2)
}
