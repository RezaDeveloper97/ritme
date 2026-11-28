package migrations_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/cycletips"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

// D-216: the seeded rows are exactly cycletips.Defs with the legacy copy (the default language as the base, every
// other language in content.translations), approved and served in the smart_tip slot.
func TestCycleSmartTipRulesMigration_SeedMatchesDefs(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	var base string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT code FROM languages WHERE is_default = 1").Scan(&base))

	assert.Equal(t, len(cycletips.Defs), countRows(t, db, "SELECT COUNT(*) FROM message_rules WHERE rule_key LIKE 'tip.%'"))
	for _, d := range cycletips.Defs {
		var title, level, kind, placement, status, content, conditions string
		var topics sql.NullString
		var prio, active, version, approved, locked int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT title, level, kind, priority, placement, status, is_active,
			version, approved_version, locked, content, conditions, topics FROM message_rules WHERE rule_key = ?`, d.Key).
			Scan(&title, &level, &kind, &prio, &placement, &status, &active, &version, &approved, &locked, &content, &conditions, &topics), d.Key)
		assert.Equal(t, []any{d.Name, cycletips.Level, cycletips.Kind, d.Prio, cycletips.Slot, rules.StatusApproved, 1, 1, 1, 0},
			[]any{title, level, kind, prio, placement, status, active, version, approved, locked}, d.Key)
		want, err := d.ContentJSON(base)
		require.NoError(t, err)
		assert.JSONEq(t, want, content, d.Key)
		assert.JSONEq(t, d.ConditionsJSON(), conditions, d.Key)
		if js := d.TopicsJSON(); js == "" {
			assert.False(t, topics.Valid, d.Key)
		} else {
			assert.JSONEq(t, js, topics.String, d.Key)
		}
	}
}
