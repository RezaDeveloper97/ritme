package pregnancyalerts_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-M9-05: severe BP is urgent in both engines; emergency rules fire without their copy row.

func TestEmergency_SevereBPBothEngines(t *testing.T) {
	h := setup(t)
	uid, tok := h.user(t, "09120000950", true)
	r := h.do(t, http.MethodPost, "/api/v1/pregnancy/weekly", tok, "fa",
		`{"pregnancy_week":13,"log_date":"2026-09-23","systolic_pressure":165,"diastolic_pressure":112}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	v1, _ := r.data()["alerts"].([]any)
	require.Len(t, v1, 1)
	assert.Equal(t, "emergency", v1[0].(map[string]any)["alert_level"])
	assert.Contains(t, v1[0].(map[string]any)["message"], "۱۱۵")

	list := alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", tok, "en", ""))
	bp := byRule(list, "bp_high")
	require.Len(t, bp, 1)
	assert.Equal(t, "urgent", bp[0]["level"])
	assert.Equal(t, "Your blood pressure is in the severe range for pregnancy", bp[0]["title"])
	assert.Contains(t, bp[0]["contact"], "115")
	assert.Equal(t, 1, h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ? AND alert_type = 'v2:bp_high'`, uid))
}

func TestEmergency_RulesWithoutCopyRow(t *testing.T) {
	for name, stmt := range map[string]string{
		"disabled": "UPDATE message_contents SET payload = JSON_SET(payload, '$.enabled', false) WHERE `group` = 'pregnancy_alert' AND item_key IN ('critical_symptom', 'bp_high')",
		"deleted":  "DELETE FROM message_contents WHERE `group` = 'pregnancy_alert' AND item_key IN ('critical_symptom', 'bp_high')",
	} {
		t.Run(name, func(t *testing.T) {
			h := setup(t)
			h.exec(stmt)
			uid, tok := h.user(t, "09120000951", true)
			r := h.do(t, http.MethodPost, "/api/v1/pregnancy/symptoms", tok, "en",
				`{"log_date":"2026-09-23","has_fluid_leakage":true,"fluid_leakage_severity":"mild"}`)
			require.Equal(t, http.StatusCreated, r.status, r.raw)
			r = h.do(t, http.MethodPost, "/api/v1/pregnancy/weekly", tok, "en",
				`{"pregnancy_week":13,"log_date":"2026-09-23","systolic_pressure":150,"diastolic_pressure":95}`)
			require.Equal(t, http.StatusCreated, r.status, r.raw)
			r = h.do(t, http.MethodPost, "/api/v1/pregnancy/weekly", tok, "en",
				`{"pregnancy_week":13,"log_date":"2026-09-22","systolic_pressure":170,"diastolic_pressure":100}`)
			require.Equal(t, http.StatusCreated, r.status, r.raw)

			for _, loc := range []string{"fa", "en"} {
				list := alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", tok, loc, ""))
				crit := byRule(list, "critical_symptom")
				require.Len(t, crit, 1, loc)
				assert.Equal(t, "urgent", crit[0]["level"], loc)
				assert.NotEmpty(t, crit[0]["title"], loc)
				assert.Contains(t, crit[0]["contact"], map[string]string{"fa": "۱۱۵", "en": "115"}[loc], loc)
				bp := byRule(list, "bp_high")
				require.Len(t, bp, 1, "%s: only the severe tier fires while the row is off", loc)
				assert.Equal(t, "urgent", bp[0]["level"], loc)
			}
			assert.Equal(t, 1, h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ? AND alert_type = 'v2:critical_symptom'`, uid))
		})
	}
}
