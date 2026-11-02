package migrations_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/pregnancyalerts/inboxalerts"
	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const pregnancyV1AlertsInboxVersion int64 = 114 // 00114_pregnancy_v1_alerts_inbox.sql

// T-M10-08c: Up moves every legacy v1 pregnancy_alerts row (each shape a v1 writer stored: Laravel's
// PregnancyAlertService and the retired Go rules, incl. the day-log flow alert) into message_inbox on its preg.* rule
// with the evidence its copy renders; the state mapping keeps an unread v1 emergency an unread emergency (also behind an
// acknowledged open row of its rule, which becomes history), archives dismissed / > 30 days / history-only rows, skips
// v2 rows and rows already moved, and uses the code-held id of a deleted locked rule. Down removes the moved rows and
// restores the superseded row.
func TestPregnancyV1AlertsInboxMigration(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })
	exec := func(q string, args ...any) {
		t.Helper()
		_, err := db.ExecContext(ctx, q, args...)
		require.NoError(t, err)
	}
	_, err = p.DownTo(ctx, pregnancyV1AlertsInboxVersion-1)
	require.NoError(t, err)

	exec(`INSERT INTO users (id, name, mobile, created_at, updated_at) VALUES (9301, 'A', '09120009301', NOW(), NOW()),
		(9302, 'B', '09120009302', NOW(), NOW()), (9303, 'C', '09120009303', NOW(), NOW())`)
	add := func(id, user uint64, level, typ, title, trigger string, read, dismissed, days int) {
		t.Helper()
		var trig any
		if trigger != "" {
			trig = trigger
		}
		exec(`INSERT INTO pregnancy_alerts (id, user_id, alert_level, alert_type, title, message, pregnancy_week,
			trigger_symptoms, is_read, is_dismissed, read_at, dismissed_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, 'm', 22, ?, ?, ?, IF(? = 1, NOW() - INTERVAL ? DAY, NULL), IF(? = 1, NOW(), NULL),
			NOW() - INTERVAL ? DAY, NOW() - INTERVAL ? DAY)`,
			id, user, level, typ, title, trig, read, dismissed, read, days, dismissed, days, days)
	}
	sb := "symptom_based"
	add(101, 9301, "emergency", sb, "Emergency: Heavy Bleeding", `{"bleeding":"severe"}`, 0, 0, 2)
	add(102, 9301, "warning", sb, "Warning: Spotting", `{"spotting":"mild"}`, 0, 0, 1)                        // newer, lower: history
	add(103, 9301, "info", sb, "Note: Light Spotting", `{"spotting":"mild"}`, 0, 0, 1)                        // no rule raises it today
	add(104, 9301, "warning", sb, "Warning: High Blood Pressure", `{"systolic":150,"diastolic":95}`, 1, 0, 3) // read
	add(105, 9301, "warning", sb, "Warning: High Blood Pressure", `{"systolic":165,"diastolic":100}`, 0, 0, 3)
	add(106, 9301, "emergency", sb, "Emergency: Severe Blood Pressure", `{"systolic":170,"diastolic":112,"severe":true}`, 0, 1, 2)
	add(107, 9301, "warning", sb, "Warning: High Blood Sugar", `{"fasting":"98.00","post_meal":null}`, 0, 0, 5)
	add(108, 9301, "emergency", sb, "Emergency: No Fetal Movement", `{"fetal_movement":"none"}`, 0, 0, 40) // > 30 days
	add(109, 9301, "warning", sb, "Warning: Reduced Fetal Movement", `{"fetal_movement":"reduced"}`, 1, 0, 4)
	add(110, 9301, "warning", sb, "Note: Mental Health", "", 0, 0, 2)
	add(111, 9301, "warning", sb, "هشدار: علائم متعدد شدید", `{"multiple_severe":true}`, 0, 0, 0)
	add(112, 9301, "info", "missing_data", "Reminder: Log Symptoms", "", 0, 0, 1)
	add(113, 9301, "emergency", "v2:critical_symptom", "v2", `{"rule":"critical_symptom"}`, 0, 0, 1) // a v2 row: 00113's
	add(114, 9301, "warning", sb, "Warning: Bleeding", `{"bleeding":"mild","flow":"low","source":"day_log"}`, 0, 0, 1)
	exec(`INSERT INTO pregnancy_symptom_logs (user_id, log_date, nausea_severity, vomiting_severity, fatigue_severity,
		headache_severity, created_at, updated_at) VALUES (9301, CURDATE(), 'severe', 'severe', 'severe', 'severe', NOW(), NOW())`)
	// Already moved by an earlier run: skipped.
	exec(`INSERT INTO message_inbox (user_id, rule_id, version, rule_key, kind, level, state, legacy_alert_id, created_at, updated_at)
		VALUES (9301, 1, 1, 'preg.critical_symptom', 'safety', 'follow_soon', 'resolved', 114, NOW(), NOW())`)

	// 9302: acknowledged open rows (the v6 inbox) and newer unread v1 rows of the same rules.
	var critID, gdmID uint64
	require.NoError(t, db.QueryRowContext(ctx, "SELECT id FROM message_rules WHERE rule_key = 'preg.critical_symptom'").Scan(&critID))
	require.NoError(t, db.QueryRowContext(ctx, "SELECT id FROM message_rules WHERE rule_key = 'preg.gdm'").Scan(&gdmID))
	exec(`INSERT INTO message_inbox (user_id, rule_id, version, rule_key, kind, level, state, evidence, first_at, last_at,
		acked_at, created_at, updated_at) VALUES
		(9302, ?, 1, 'preg.critical_symptom', 'safety', 'follow_soon', 'acknowledged', '{"vars":{"preg_week":22}}', NOW() - INTERVAL 3 DAY,
		NOW() - INTERVAL 3 DAY, NOW() - INTERVAL 3 DAY, NOW(), NOW()),
		(9302, ?, 1, 'preg.gdm', 'clinical', 'worth_followup', 'acknowledged', NULL, NOW() - INTERVAL 3 DAY,
		NOW() - INTERVAL 3 DAY, NOW() - INTERVAL 3 DAY, NOW(), NOW())`, critID, gdmID)
	add(201, 9302, "emergency", sb, "Emergency: Fluid Leakage", `{"fluid_leakage":"moderate"}`, 0, 0, 1)
	add(202, 9302, "warning", sb, "Warning: High Blood Sugar", `{"fasting":"101.00","post_meal":"150.00"}`, 0, 0, 1)

	// 9303: the locked fetal-movement rule row was deleted; the emergency still moves, on the code-held id.
	exec("DELETE FROM message_rules WHERE rule_key = 'preg.fetal_movement'")
	add(301, 9303, "emergency", sb, "Emergency: No Fetal Movement", `{"fetal_movement":"none"}`, 0, 0, 1)

	_, err = p.UpTo(ctx, pregnancyV1AlertsInboxVersion)
	require.NoError(t, err)

	type row struct {
		key, state, level, kind string
		acked                   bool
		vars                    map[string]any
		ruleID                  uint64
	}
	read := func() map[uint64]row {
		out := map[uint64]row{}
		rows, err := db.QueryContext(ctx, `SELECT legacy_alert_id, rule_key, state, level, kind, acked_at IS NOT NULL, evidence, rule_id
			FROM message_inbox WHERE legacy_alert_id IS NOT NULL AND legacy_alert_id <> 114`)
		require.NoError(t, err)
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var id uint64
			var r row
			var ev string
			require.NoError(t, rows.Scan(&id, &r.key, &r.state, &r.level, &r.kind, &r.acked, &ev, &r.ruleID))
			var doc struct {
				Vars map[string]any `json:"vars"`
			}
			require.NoError(t, json.Unmarshal([]byte(ev), &doc))
			r.vars = doc.Vars
			out[id] = r
		}
		require.NoError(t, rows.Err())
		return out
	}
	got := read()
	require.Len(t, got, 15, "every v1 row but the v2 one and the one already moved")
	assert.NotContains(t, got, uint64(113))

	assert.Equal(t, "preg.critical_symptom", got[101].key)
	assert.Equal(t, "active", got[101].state, "the unread emergency is the open row")
	assert.Equal(t, "follow_soon", got[101].level)
	assert.Equal(t, []any{"bleeding"}, got[101].vars["preg_symptoms"])
	assert.Equal(t, 22.0, got[101].vars["preg_week"])
	assert.NotEmpty(t, got[101].vars["preg_symptom_on"])
	assert.Equal(t, "resolved", got[102].state)
	assert.Equal(t, []any{"spotting"}, got[102].vars["preg_symptoms"])
	assert.Equal(t, "archived", got[103].state, "light spotting: history only")

	assert.Equal(t, row{"preg.bp_high", "acknowledged", "worth_followup", "clinical", true,
		map[string]any{"preg_week": 22.0, "preg_bp_sys": "150", "preg_bp_dia": "95", "preg_bp_on": got[104].vars["preg_bp_on"]},
		got[104].ruleID}, got[104])
	assert.Equal(t, "preg.bp_severe", got[105].key, "≥ 160 is severe even when v1 called it a warning")
	assert.Equal(t, "active", got[105].state)
	assert.Equal(t, "safety", got[105].kind)
	assert.Equal(t, "preg.bp_severe", got[106].key)
	assert.Equal(t, "archived", got[106].state, "dismissed")
	assert.True(t, got[106].acked)

	assert.Equal(t, "preg.gdm", got[107].key)
	assert.Equal(t, "active", got[107].state)
	assert.Equal(t, "98.00", got[107].vars["preg_fasting"])
	assert.Equal(t, "—", got[107].vars["preg_post_meal"])
	assert.Equal(t, "preg.fetal_movement", got[108].key)
	assert.Equal(t, "archived", got[108].state, "> 30 days")
	assert.Equal(t, "acknowledged", got[109].state)
	assert.Equal(t, "reduced", got[109].vars["preg_fetal_status"])
	assert.Equal(t, "follow_soon", got[109].level, "reduced movement is urgent today")
	assert.Equal(t, "preg.depression", got[110].key)
	assert.Equal(t, "active", got[110].state)
	assert.NotEmpty(t, got[110].vars["preg_mood_on"])
	assert.Equal(t, "preg.severe_symptoms", got[111].key)
	assert.Equal(t, 4.0, got[111].vars["preg_severe_count"], "counted from the day's symptom log")
	assert.Equal(t, "preg.weight_missing", got[112].key)
	assert.Equal(t, "archived", got[112].state, "missing_data: no rule, history")
	assert.Equal(t, 1, countRows(t, db, "SELECT COUNT(*) FROM message_inbox WHERE legacy_alert_id = 114"), "already moved")

	// 9302: the unread emergency is active, the acknowledged row it replaced is history; the warning stays history.
	assert.Equal(t, "active", got[201].state)
	assert.Equal(t, []any{"fluid_leakage"}, got[201].vars["preg_symptoms"])
	assert.Equal(t, "resolved", got[202].state)
	var supersededState, superseded string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT state, JSON_VALUE(evidence, '$.v1_superseded') FROM message_inbox
		WHERE user_id = 9302 AND rule_key = 'preg.critical_symptom' AND legacy_alert_id IS NULL`).Scan(&supersededState, &superseded))
	assert.Equal(t, "resolved", supersededState)
	assert.Equal(t, "acknowledged", superseded)
	assert.Equal(t, 1, countRows(t, db, `SELECT COUNT(*) FROM message_inbox WHERE user_id = 9302 AND rule_key = 'preg.gdm'
		AND legacy_alert_id IS NULL AND state = 'acknowledged'`))

	// 9303: the code-held id of the deleted locked rule.
	assert.Equal(t, inboxalerts.CodeRuleID("preg.fetal_movement"), got[301].ruleID)
	assert.Equal(t, row{"preg.fetal_movement", "active", "follow_soon", "safety", false, got[301].vars, got[301].ruleID}, got[301])
	for _, key := range []string{"preg.critical_symptom", "preg.fetal_movement", "preg.bp_severe"} {
		d, ok := inboxalerts.DefOf(key)
		require.True(t, ok)
		assert.True(t, d.Locked, key)
	}

	// Open rows: one per (user, rule).
	assert.Zero(t, countRows(t, db, `SELECT COUNT(*) FROM (SELECT user_id, rule_id FROM message_inbox WHERE open_flag = 1
		GROUP BY user_id, rule_id HAVING COUNT(*) > 1) d`))
	assert.Zero(t, countRows(t, db, "SELECT COUNT(*) FROM message_inbox WHERE state = 'v1_staged'"))
	assert.Equal(t, 17, countRows(t, db, "SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id IN (9301, 9302, 9303)"), "the old rows stay")

	// Down: the moved rows go, the superseded row is acknowledged again; Up again gives the same rows.
	_, err = p.DownTo(ctx, pregnancyV1AlertsInboxVersion-1)
	require.NoError(t, err)
	assert.Zero(t, countRows(t, db, "SELECT COUNT(*) FROM message_inbox WHERE legacy_alert_id IS NOT NULL"), "every moved v1 row (00113 rows: none here)")
	require.NoError(t, db.QueryRowContext(ctx, `SELECT state, COALESCE(JSON_VALUE(evidence, '$.v1_superseded'), '') FROM message_inbox
		WHERE user_id = 9302 AND rule_key = 'preg.critical_symptom'`).Scan(&supersededState, &superseded))
	assert.Equal(t, "acknowledged", supersededState)
	assert.Empty(t, superseded)
	_, err = p.UpTo(ctx, pregnancyV1AlertsInboxVersion)
	require.NoError(t, err)
	again := read()
	require.Len(t, again, len(got))
	for id, r := range got {
		assert.Equal(t, r.key+"/"+r.state, again[id].key+"/"+again[id].state, id)
	}

	exec("DELETE FROM users WHERE id IN (9301, 9302, 9303)")
	// Restore the deleted rule row (00113 seeds it again).
	_, err = p.DownTo(ctx, pregnancyV1AlertsInboxVersion-2)
	require.NoError(t, err)
}
