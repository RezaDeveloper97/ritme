package migrations_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const messageEngineV6Version int64 = 111 // 00111_message_engine_v6.sql

// messageEngineV6Objects counts the 00111 columns, tables and indexes, and the width of message_events.variant.
func messageEngineV6Objects(t *testing.T, db *sql.DB) (columns, tables, indexes, variantLen int) {
	t.Helper()
	columns = countRows(t, db, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE()
		AND ((table_name = 'message_rules' AND column_name IN ('rule_key', 'approved_version', 'topics', 'caps', 'variants',
		      'starts_at', 'ends_at', 'hour_windows', 'goal_event', 'locked'))
		  OR (table_name = 'message_events' AND column_name IN ('version', 'slot'))
		  OR (table_name = 'challenge_programs' AND column_name IN ('contraindications', 'start_phase')))`)
	tables = countRows(t, db, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE()
		AND table_name IN ('message_event_daily', 'message_inbox')`)
	indexes = countRows(t, db, `SELECT COUNT(DISTINCT index_name) FROM information_schema.statistics WHERE table_schema = DATABASE()
		AND index_name IN ('message_rules_rule_key_unique', 'message_events_user_id_created_at_index',
		  'message_events_created_at_rule_id_index')`)
	variantLen = countRows(t, db, `SELECT character_maximum_length FROM information_schema.columns WHERE table_schema = DATABASE()
		AND table_name = 'message_events' AND column_name = 'variant'`)
	return
}

// T-M10-02: Down removes every 00111 object (variant back to one character); Up re-maps the unambiguous slot/kind
// mismatches (M-18) and leaves the rest; open_flag allows one open inbox row per user and message and any number of
// history rows; a second Up on a database whose Laravel twin already ran is a no-op.
func TestMessageEngineV6Migration_UpDownRemapInbox(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	columns, tables, indexes, variantLen := messageEngineV6Objects(t, db)
	assert.Equal(t, []int{14, 2, 3, 16}, []int{columns, tables, indexes, variantLen})

	_, err = p.DownTo(ctx, messageEngineV6Version-1)
	require.NoError(t, err)
	columns, tables, indexes, variantLen = messageEngineV6Objects(t, db)
	assert.Equal(t, []int{0, 0, 0, 1}, []int{columns, tables, indexes, variantLen})

	exec := func(q string, args ...any) {
		t.Helper()
		_, err := db.ExecContext(ctx, q, args...)
		require.NoError(t, err)
	}
	exec("DELETE FROM message_rules")
	ins := `INSERT INTO message_rules (id, title, level, kind, priority, placement, content, conditions, exceptions,
		status, is_active, created_at, updated_at) VALUES (?, 'r', 'info', ?, ?, ?, '{"a":{"title":"t","body":"b"}}', '[]', '[]',
		'draft', 1, NOW(), NOW())`
	exec(ins, 1, "product", 30, "daily_advice")   // → content
	exec(ins, 2, "product", 30, "modal")          // → marketing
	exec(ins, 3, "product", 30, "smart_tip")      // mismatch, kept
	exec(ins, 4, "clinical", 200, "daily_advice") // mismatch, kept
	exec(ins, 5, "product", 30, "push")           // allowed, kept

	_, err = p.UpTo(ctx, messageEngineV6Version)
	require.NoError(t, err)
	kinds := func() []string {
		rows, err := db.QueryContext(ctx, "SELECT kind FROM message_rules ORDER BY id")
		require.NoError(t, err)
		defer rows.Close()
		var out []string
		for rows.Next() {
			var k string
			require.NoError(t, rows.Scan(&k))
			out = append(out, k)
		}
		require.NoError(t, rows.Err())
		return out
	}
	assert.Equal(t, []string{"content", "marketing", "product", "clinical", "product"}, kinds())
	var locked, approved sql.NullInt64
	require.NoError(t, db.QueryRowContext(ctx, "SELECT locked, approved_version FROM message_rules WHERE id = 1").Scan(&locked, &approved))
	assert.Equal(t, int64(0), locked.Int64)
	assert.False(t, approved.Valid)

	// open_flag: one open row per (user, rule); history rows are unlimited; resolving the open row frees the slot.
	exec(`INSERT INTO users (id, name, mobile, created_at, updated_at) VALUES (9101, 'Inbox', '09120009101', NOW(), NOW())`)
	inbox := `INSERT INTO message_inbox (user_id, rule_id, version, kind, level, state, created_at, updated_at)
		VALUES (9101, 7, 1, 'clinical', 'worth_followup', ?, NOW(), NOW())`
	exec(inbox, "active")
	_, err = db.ExecContext(ctx, inbox, "acknowledged")
	require.Error(t, err, "a second open row for the same user and rule")
	exec(inbox, "resolved")
	exec(inbox, "archived")
	exec("UPDATE message_inbox SET state = 'resolved' WHERE state = 'active'")
	exec(inbox, "added_to_visit_note")
	assert.Equal(t, 1, countRows(t, db, "SELECT COUNT(*) FROM message_inbox WHERE open_flag = 1"))
	assert.Equal(t, 3, countRows(t, db, "SELECT COUNT(*) FROM message_inbox WHERE open_flag IS NULL"))
	exec("DELETE FROM users WHERE id = 9101")
	assert.Equal(t, 0, countRows(t, db, "SELECT COUNT(*) FROM message_inbox"), "cascade with the user")

	// Down maps the v6 kinds back (content ≥ 100 → clinical, else product; marketing → product).
	exec("UPDATE message_rules SET priority = 300 WHERE id = 4")
	exec("UPDATE message_rules SET kind = 'content', priority = 300 WHERE id = 5")
	_, err = p.DownTo(ctx, messageEngineV6Version-1)
	require.NoError(t, err)
	assert.Equal(t, []string{"product", "product", "product", "clinical", "clinical"}, kinds())

	// Idempotent Up (IF NOT EXISTS everywhere).
	_, err = p.UpTo(ctx, messageEngineV6Version)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "ALTER TABLE `message_rules` ADD COLUMN IF NOT EXISTS `rule_key` varchar(64) DEFAULT NULL")
	require.NoError(t, err)
	exec("DELETE FROM message_rules")
}
