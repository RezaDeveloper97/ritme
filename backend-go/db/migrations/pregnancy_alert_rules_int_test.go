package migrations_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/pregnancyalerts/inboxalerts"
	"github.com/ritme/backend-go/internal/messages/rules"
	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const pregnancyAlertRulesVersion int64 = 113 // 00113_pregnancy_alert_rules.sql

// T-M10-08: the seeded rows are exactly inboxalerts.Defs with the embedded copy (the default language as the base,
// every other language in content.translations), approved and served in alerts_inbox, the safety rules locked.
func TestPregnancyAlertRulesMigration_SeedMatchesDefs(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	var base string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT code FROM languages WHERE is_default = 1").Scan(&base))

	assert.Equal(t, len(inboxalerts.Defs), countRows(t, db, "SELECT COUNT(*) FROM message_rules WHERE rule_key LIKE 'preg.%'"))
	for _, d := range inboxalerts.Defs {
		var title, level, kind, placement, status, content, conditions string
		var prio, active, version, approved, locked int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT title, level, kind, priority, placement, status, is_active,
			version, approved_version, locked, content, conditions FROM message_rules WHERE rule_key = ?`, d.Key).
			Scan(&title, &level, &kind, &prio, &placement, &status, &active, &version, &approved, &locked, &content, &conditions), d.Key)
		assert.Equal(t, []any{d.Name, d.Level, d.Kind, d.Prio, rules.SlotAlertsInbox, rules.StatusApproved, 1, 1, 1, d.Locked},
			[]any{title, level, kind, prio, placement, status, active, version, approved, locked == 1}, d.Key)
		want, err := d.ContentJSON(base)
		require.NoError(t, err)
		assert.JSONEq(t, want, content, d.Key)
		assert.JSONEq(t, d.ConditionsJSON(), conditions, d.Key)
		if d.Kind == rules.KindSafety {
			assert.True(t, d.Locked, "%s: a safety rule is locked", d.Key)
		}
	}
}

// T-M10-08: Down removes the seeded rules and the migrated inbox rows only; Up moves the v2 pregnancy_alerts rows
// into message_inbox with the state mapping (newest per user and rule open: acked → acknowledged, read →
// added_to_visit_note, else active; older ones resolved, > 30 days archived), the evidence variables renamed, and
// legacy_alert_id kept; a row already moved is skipped; v1 rows stay where they are.
func TestPregnancyAlertRulesMigration_MovesV2Alerts(t *testing.T) {
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

	_, err = p.DownTo(ctx, pregnancyAlertRulesVersion-1)
	require.NoError(t, err)
	assert.Zero(t, countRows(t, db, "SELECT COUNT(*) FROM message_rules WHERE rule_key LIKE 'preg.%'"))

	exec(`INSERT INTO users (id, name, mobile, created_at, updated_at) VALUES (9201, 'Preg', '09120009201', NOW(), NOW()),
		(9202, 'Preg2', '09120009202', NOW(), NOW())`)
	alert := `INSERT INTO pregnancy_alerts (id, user_id, alert_level, alert_type, title, message, pregnancy_week,
		trigger_symptoms, is_read, is_dismissed, read_at, dismissed_at, created_at, updated_at)
		VALUES (?, ?, 'warning', ?, 't', 'm', 20, ?, ?, ?, NULL, NULL, NOW() - INTERVAL ? DAY, NOW() - INTERVAL ? DAY)`
	add := func(id, user uint64, typ, trigger string, read, dismissed, days int) {
		t.Helper()
		exec(alert, id, user, typ, trigger, read, dismissed, days, days)
	}
	add(1, 9201, "v2:vomiting_streak", `{"rule":"vomiting_streak","vars":{"days":"4","severe_count":"1"}}`, 0, 0, 1)
	add(2, 9201, "v2:vomiting_streak", `{"rule":"vomiting_streak","vars":{"days":"3","severe_count":"0"}}`, 0, 0, 5)  // older → resolved
	add(3, 9201, "v2:vomiting_streak", `{"rule":"vomiting_streak","vars":{"days":"3","severe_count":"0"}}`, 0, 1, 40) // > 30 days → archived
	add(4, 9201, "v2:bp_high", `{"rule":"bp_high","variant":"severe","vars":{"systolic":"165","diastolic":"112"},"on":"2026-10-30"}`, 0, 1, 1)
	add(5, 9201, "v2:bp_high", `{"rule":"bp_high","level4":"urgent","vars":{"systolic":"145","diastolic":"\u0000none"}}`, 1, 0, 2)
	add(6, 9201, "v2:bp_high", `{"rule":"bp_high","level4":"follow_up","vars":{"systolic":"142","diastolic":"91"}}`, 1, 0, 2)
	add(7, 9201, "v2:fetal_movement", `{"rule":"fetal_movement","variant":"kicks","vars":{"count":"4","target":"10","minutes":"120"}}`, 0, 0, 0)
	add(8, 9201, "v2:fetal_movement", `{"rule":"fetal_movement","vars":{"status":"reduced"}}`, 0, 0, 0)
	add(9, 9201, "v2:week_entered", `{"rule":"week_entered","vars":{"week":"20","basis":"lmp"}}`, 0, 0, 45) // only row, > 30 days → archived
	add(10, 9202, "v2:critical_symptom", `{"rule":"critical_symptom","vars":{"symptom":"bleeding"},"on":"2026-10-29"}`, 0, 0, 1)
	add(11, 9202, "v2:contractions_511", `{"rule":"contractions_511","vars":{"count":"12","minutes":"60","interval":"05:00","duration":"01:05"}}`, 0, 0, 0)
	add(12, 9202, "bleeding", `{}`, 0, 0, 1)                                                                          // v1 row: not moved
	add(13, 9202, "v2:sugar_high", `{"rule":"sugar_high","vars":{"fasting":"98","post_meal":"\u0000none"}}`, 0, 0, 0) // already moved below
	// An open row of user 9202 for preg.gdm already exists (moved by an earlier run): alert 13 is skipped.
	exec(`INSERT INTO message_inbox (user_id, rule_id, version, rule_key, kind, level, state, legacy_alert_id, created_at, updated_at)
		VALUES (9202, 1, 1, 'preg.gdm', 'clinical', 'worth_followup', 'active', 13, NOW(), NOW())`)

	_, err = p.UpTo(ctx, pregnancyAlertRulesVersion)
	require.NoError(t, err)

	type row struct {
		key, state, level string
		acked, noted      bool
		vars              map[string]any
	}
	got := map[uint64]row{}
	func() {
		rows, err := db.QueryContext(ctx, `SELECT i.legacy_alert_id, i.rule_key, i.state, i.level, i.acked_at IS NOT NULL,
		i.noted_at IS NOT NULL, i.evidence, r.rule_key FROM message_inbox i JOIN message_rules r ON r.id = i.rule_id
		WHERE i.legacy_alert_id IS NOT NULL AND i.legacy_alert_id <> 13`)
		require.NoError(t, err)
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var id uint64
			var r row
			var ev, ruleKey string
			require.NoError(t, rows.Scan(&id, &r.key, &r.state, &r.level, &r.acked, &r.noted, &ev, &ruleKey))
			assert.Equal(t, r.key, ruleKey)
			var doc struct {
				Vars map[string]any `json:"vars"`
			}
			require.NoError(t, json.Unmarshal([]byte(ev), &doc))
			r.vars = doc.Vars
			got[id] = r
		}
		require.NoError(t, rows.Err())
	}()
	require.Len(t, got, 11)

	assert.Equal(t, row{"preg.vomiting", "active", "worth_followup", false, false,
		map[string]any{"preg_week": 20.0, "preg_vomit_days": "4", "preg_vomit_severe": "1"}}, got[1])
	assert.Equal(t, "resolved", got[2].state)
	assert.Equal(t, "archived", got[3].state)
	assert.Equal(t, row{"preg.bp_severe", "acknowledged", "follow_soon", true, false,
		map[string]any{"preg_week": 20.0, "preg_bp_sys": "165", "preg_bp_dia": "112", "preg_bp_on": "2026-10-30"}}, got[4])
	assert.Equal(t, "preg.bp_high_urgent", got[5].key)
	assert.Equal(t, "added_to_visit_note", got[5].state)
	assert.True(t, got[5].noted)
	assert.Equal(t, "—", got[5].vars["preg_bp_dia"])
	assert.Equal(t, "preg.bp_high", got[6].key)
	assert.Equal(t, "worth_followup", got[6].level)
	assert.Equal(t, "preg.fetal_movement_kicks", got[7].key)
	assert.Equal(t, "4", got[7].vars["preg_kick_count"])
	assert.NotEmpty(t, got[7].vars["preg_kick_at"])
	assert.Equal(t, "preg.fetal_movement", got[8].key)
	assert.Equal(t, "reduced", got[8].vars["preg_fetal_status"])
	assert.Equal(t, "archived", got[9].state)
	assert.Equal(t, "lmp", got[9].vars["preg_basis"])
	assert.NotEmpty(t, got[9].vars["preg_week_on"], "the week start (else the creation day)")
	assert.Equal(t, row{"preg.critical_symptom", "active", "follow_soon", false, false,
		map[string]any{"preg_week": 20.0, "preg_symptoms": []any{"bleeding"}, "preg_symptom_on": "2026-10-29"}}, got[10])
	assert.Equal(t, "preg.contractions_511", got[11].key)
	assert.Equal(t, "05:00", got[11].vars["preg_labor_interval"])
	assert.Equal(t, 1, countRows(t, db, "SELECT COUNT(*) FROM message_inbox WHERE legacy_alert_id = 13"), "already moved")
	assert.Equal(t, 13, countRows(t, db, "SELECT COUNT(*) FROM pregnancy_alerts"), "the old rows stay")
	assert.Equal(t, 1, countRows(t, db, "SELECT COUNT(*) FROM message_inbox WHERE user_id = 9201 AND rule_key = 'preg.vomiting' AND open_flag = 1"))

	// Down: the migrated rows and the seeded rules go; a live inbox row stays.
	exec(`INSERT INTO message_inbox (user_id, rule_id, version, rule_key, kind, level, state, created_at, updated_at)
		VALUES (9202, 2, 1, 'other', 'clinical', 'info', 'active', NOW(), NOW())`)
	_, err = p.DownTo(ctx, pregnancyAlertRulesVersion-1)
	require.NoError(t, err)
	assert.Equal(t, 1, countRows(t, db, "SELECT COUNT(*) FROM message_inbox"))
	assert.Zero(t, countRows(t, db, "SELECT COUNT(*) FROM message_rules WHERE rule_key LIKE 'preg.%'"))

	// Up keeps an existing (edited) rule row as it is.
	exec(`INSERT INTO message_rules (rule_key, title, level, kind, priority, placement, content, conditions, exceptions,
		status, is_active, created_at, updated_at) VALUES ('preg.gdm', 'edited', 'worth_followup', 'clinical', 650,
		'alerts_inbox', '{"a":{"title":"t","body":"b"}}', '[]', '[]', 'draft', 1, NOW(), NOW())`)
	_, err = p.UpTo(ctx, pregnancyAlertRulesVersion)
	require.NoError(t, err)
	var title sql.NullString
	require.NoError(t, db.QueryRowContext(ctx, "SELECT title FROM message_rules WHERE rule_key = 'preg.gdm'").Scan(&title))
	assert.Equal(t, "edited", title.String)
	exec("DELETE FROM users WHERE id IN (9201, 9202)")
}
