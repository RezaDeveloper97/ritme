package migrations_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/conditionnudges"
	"github.com/ritme/backend-go/internal/messages/menomessages"
	"github.com/ritme/backend-go/internal/messages/rules"
	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const menopauseConditionRulesVersion int64 = 115 // 00115_menopause_condition_rules.sql

type seededRule struct {
	key, name, level, kind, slot string
	prio                         int
	locked                       bool
	content                      func(base string) (string, error)
	conditions                   string
}

func menopauseConditionSeeds() []seededRule {
	var out []seededRule
	for _, d := range menomessages.Defs {
		out = append(out, seededRule{d.Key, d.Name, d.Level, d.Kind, d.Slot, d.Prio, d.Locked, d.ContentJSON, d.ConditionsJSON()})
	}
	for _, d := range conditionnudges.Defs {
		out = append(out, seededRule{d.Key, d.Name, d.Level, d.Kind, conditionnudges.Slot, d.Prio, false, d.ContentJSON,
			d.ConditionsJSON()})
	}
	return out
}

// T-M10-09: the seeded rows are exactly menomessages.Defs and conditionnudges.Defs with their seed copy (the default
// language as the base, every other language in content.translations), approved and served, the bleeding alert locked.
func TestMenopauseConditionRulesMigration_SeedMatchesDefs(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	var base string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT code FROM languages WHERE is_default = 1").Scan(&base))

	assert.Equal(t, len(menomessages.Defs), countRows(t, db, "SELECT COUNT(*) FROM message_rules WHERE rule_key LIKE 'meno.%'"))
	assert.Equal(t, len(conditionnudges.Defs), countRows(t, db, "SELECT COUNT(*) FROM message_rules WHERE rule_key LIKE 'cond.%'"))
	for _, d := range menopauseConditionSeeds() {
		var title, level, kind, placement, status, content, conditions string
		var prio, active, version, approved, locked int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT title, level, kind, priority, placement, status, is_active,
			version, approved_version, locked, content, conditions FROM message_rules WHERE rule_key = ?`, d.key).
			Scan(&title, &level, &kind, &prio, &placement, &status, &active, &version, &approved, &locked, &content, &conditions), d.key)
		assert.Equal(t, []any{d.name, d.level, d.kind, d.prio, d.slot, rules.StatusApproved, 1, 1, 1, d.locked},
			[]any{title, level, kind, prio, placement, status, active, version, approved, locked == 1}, d.key)
		want, err := d.content(base)
		require.NoError(t, err)
		assert.JSONEq(t, want, content, d.key)
		assert.JSONEq(t, d.conditions, conditions, d.key)
		assert.True(t, rules.LevelAllowed(d.level, d.kind), d.key)
		assert.True(t, rules.KindAllowedIn(d.slot, d.kind), d.key)
	}
	var locked string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT rule_key FROM message_rules WHERE locked = 1 AND (rule_key LIKE 'meno.%' OR rule_key LIKE 'cond.%')").Scan(&locked))
	assert.Equal(t, "meno.postmenopausal_bleeding", locked)
}

// T-M10-09: Down removes the inbox rows of these rules and the seeded rules (other rows stay); Up keeps an existing
// (edited) rule row as it is.
func TestMenopauseConditionRulesMigration_DownUp(t *testing.T) {
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
	exec(`INSERT INTO users (id, name, mobile, created_at, updated_at) VALUES (9301, 'Meno', '09120009301', NOW(), NOW())`)
	exec(`INSERT INTO message_inbox (user_id, rule_id, version, rule_key, kind, level, state, created_at, updated_at)
		SELECT 9301, id, 1, rule_key, kind, level, 'active', NOW(), NOW() FROM message_rules
		WHERE rule_key IN ('meno.postmenopausal_bleeding', 'cond.heavy_pain')`)
	exec(`INSERT INTO message_inbox (user_id, rule_id, version, rule_key, kind, level, state, created_at, updated_at)
		VALUES (9301, 2, 1, 'other', 'clinical', 'info', 'active', NOW(), NOW())`)

	_, err = p.DownTo(ctx, menopauseConditionRulesVersion-1)
	require.NoError(t, err)
	assert.Zero(t, countRows(t, db, "SELECT COUNT(*) FROM message_rules WHERE rule_key LIKE 'meno.%' OR rule_key LIKE 'cond.%'"))
	assert.Equal(t, 1, countRows(t, db, "SELECT COUNT(*) FROM message_inbox WHERE user_id = 9301"), "only the other row stays")

	exec(`INSERT INTO message_rules (rule_key, title, level, kind, priority, placement, content, conditions, exceptions,
		status, is_active, created_at, updated_at) VALUES ('cond.heavy_pain', 'edited', 'suggestion', 'content', 450,
		'alerts_inbox', '{"a":{"title":"t","body":"b"}}', '[]', '[]', 'draft', 1, NOW(), NOW())`)
	_, err = p.UpTo(ctx, menopauseConditionRulesVersion)
	require.NoError(t, err)
	var title sql.NullString
	require.NoError(t, db.QueryRowContext(ctx, "SELECT title FROM message_rules WHERE rule_key = 'cond.heavy_pain'").Scan(&title))
	assert.Equal(t, "edited", title.String)
	assert.Equal(t, len(menomessages.Defs), countRows(t, db, "SELECT COUNT(*) FROM message_rules WHERE rule_key LIKE 'meno.%'"))
	exec("DELETE FROM users WHERE id = 9301")
}
