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

const dropVisitNoteActionVersion int64 = 128 // 00128_drop_alert_visit_note_action.sql

// alertActions is JSON_EXTRACT(payload, '$.actions') of one pregnancy_alert row.
func alertActions(t *testing.T, db *sql.DB, item, locale string) string {
	t.Helper()
	var v sql.NullString
	require.NoError(t, db.QueryRowContext(context.Background(),
		"SELECT JSON_COMPACT(JSON_EXTRACT(payload, '$.actions')) FROM message_contents WHERE `group` = 'pregnancy_alert' AND item_key = ? AND locale = ?",
		item, locale).Scan(&v))
	return v.String
}

// 00128: add_to_visit_note leaves every pregnancy_alert copy; the other actions keep their order and labels (an
// admin-edited label included) and rows without it are untouched. Its Down is a no-op, so the test plants the
// action the way 00005 seeded it.
func TestDropAlertVisitNoteActionMigration(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	_, err = p.DownTo(ctx, dropVisitNoteActionVersion-1)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	exec(t, db, "UPDATE message_contents SET payload = JSON_SET(payload, '$.actions', JSON_ARRAY("+
		"JSON_OBJECT('key', 'call', 'label', 'Call'), JSON_OBJECT('key', 'add_to_visit_note', 'label', 'Note'), "+
		"JSON_OBJECT('key', 'ack', 'label', 'Seen'))) WHERE `group` = 'pregnancy_alert' AND item_key = 'bp_high' AND locale = 'en'")
	exec(t, db, "UPDATE message_contents SET payload = JSON_SET(payload, '$.actions', JSON_ARRAY("+
		"JSON_OBJECT('key', 'add_to_visit_note', 'label', 'افزودن به یادداشت ویزیت'), JSON_OBJECT('key', 'ack', 'label', 'دیدم، ممنون'))) "+
		"WHERE `group` = 'pregnancy_alert' AND item_key = 'vomiting_streak' AND locale = 'fa'")
	weekBefore := alertActions(t, db, "week_entered", "en")

	_, err = p.UpTo(ctx, dropVisitNoteActionVersion)
	require.NoError(t, err)

	assert.Equal(t, `[{"key":"ack","label":"دیدم، ممنون"}]`, alertActions(t, db, "vomiting_streak", "fa"))
	assert.Equal(t, `[{"key":"call","label":"Call"},{"key":"ack","label":"Seen"}]`, alertActions(t, db, "bp_high", "en"))
	assert.Equal(t, weekBefore, alertActions(t, db, "week_entered", "en"), "a row without the action is untouched")
	var left int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM message_contents WHERE `group` = 'pregnancy_alert' "+
		"AND JSON_SEARCH(payload, 'one', 'add_to_visit_note', NULL, '$.actions[*].key') IS NOT NULL").Scan(&left))
	assert.Zero(t, left)
}
