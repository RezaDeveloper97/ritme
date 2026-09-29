package migrations_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const pregnancyCopyVersion int64 = 8 // 00008_pregnancy_v2_copy.sql

// msgField is JSON_VALUE(payload, path) of one message_contents row ("" when the row is missing,
// "<null>" when the path is).
func msgField(t *testing.T, db *sql.DB, group, item, locale, path string) string {
	t.Helper()
	var v sql.NullString
	err := db.QueryRowContext(context.Background(),
		"SELECT JSON_VALUE(payload, ?) FROM message_contents WHERE `group` = ? AND item_key = ? AND locale = ?",
		path, group, item, locale).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return ""
	}
	require.NoError(t, err)
	if !v.Valid {
		return "<null>"
	}
	return v.String
}

func exec(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), q, args...)
	require.NoError(t, err)
}

// 00008 (T-M7-20): week_entered loses «از امروز», weight_missing_week gains from_weekday = 5 and
// pregnancy_setup/calendar_note is seeded — each only where the row is still the seed; Down undoes
// exactly that and keeps admin edits.
func TestPregnancyV2CopyMigration(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	_, err = p.DownTo(ctx, pregnancyCopyVersion-1)
	require.NoError(t, err)
	t.Cleanup(func() {
		c := context.Background()
		_, _ = db.ExecContext(c, "DELETE FROM message_contents WHERE `group` = 'pregnancy_setup' AND item_key = 'calendar_note'")
		_, _ = db.ExecContext(c, "UPDATE message_contents SET payload = JSON_SET(payload, '$.what_we_saw', ?) "+
			"WHERE `group` = 'pregnancy_alert' AND item_key = 'week_entered' AND locale = 'en'", "Pregnancy week {week} started today")
		_, _ = db.ExecContext(c, "UPDATE message_contents SET payload = JSON_SET(payload, '$.params', JSON_OBJECT('from_week', 1)) "+
			"WHERE `group` = 'pregnancy_alert' AND item_key = 'weight_missing_week' AND locale = 'fa'")
		_, _ = p.Up(c)
	})

	const (
		oldFa = "هفتهٔ {week} بارداری از امروز شروع شده"
		newFa = "هفتهٔ {week} بارداری شروع شده"
		newEn = "Pregnancy week {week} has started"
	)
	require.Equal(t, oldFa, msgField(t, db, "pregnancy_alert", "week_entered", "fa", "$.what_we_saw"), "00005 seed")
	require.Equal(t, "<null>", msgField(t, db, "pregnancy_alert", "weight_missing_week", "en", "$.params.from_weekday"))
	require.Equal(t, "", msgField(t, db, "pregnancy_setup", "calendar_note", "fa", "$.plan_note"))

	// Admin edits that must survive Up and Down: the en week_entered text, the fa rule's params, and an
	// en calendar_note row created before the seed.
	exec(t, db, "UPDATE message_contents SET payload = JSON_SET(payload, '$.what_we_saw', 'Custom') "+
		"WHERE `group` = 'pregnancy_alert' AND item_key = 'week_entered' AND locale = 'en'")
	exec(t, db, "UPDATE message_contents SET payload = JSON_SET(payload, '$.params.from_week', 3) "+
		"WHERE `group` = 'pregnancy_alert' AND item_key = 'weight_missing_week' AND locale = 'fa'")
	exec(t, db, "INSERT INTO message_contents (`group`, item_key, locale, label, payload, is_active, is_approved, created_at, updated_at) "+
		"VALUES ('pregnancy_setup', 'calendar_note', 'en', 'pregnancy_setup / calendar_note', ?, 1, 1, '2026-09-29 09:00:00', '2026-09-29 09:05:00')",
		`{"plan_note":"Admin note.","basis_lmp":"A","basis_ultrasound":"B","basis_manual":"C"}`)

	_, err = p.UpByOne(ctx)
	require.NoError(t, err)
	assert.Equal(t, newFa, msgField(t, db, "pregnancy_alert", "week_entered", "fa", "$.what_we_saw"))
	assert.Equal(t, "Custom", msgField(t, db, "pregnancy_alert", "week_entered", "en", "$.what_we_saw"))
	assert.Equal(t, "5", msgField(t, db, "pregnancy_alert", "weight_missing_week", "en", "$.params.from_weekday"))
	assert.Equal(t, "1", msgField(t, db, "pregnancy_alert", "weight_missing_week", "en", "$.params.from_week"))
	assert.Equal(t, "<null>", msgField(t, db, "pregnancy_alert", "weight_missing_week", "fa", "$.params.from_weekday"),
		"edited params are left alone")
	assert.Contains(t, msgField(t, db, "pregnancy_setup", "calendar_note", "fa", "$.plan_note"), "ممکنه پزشکت برنامهٔ متفاوتی بده")
	assert.Equal(t, "Admin note.", msgField(t, db, "pregnancy_setup", "calendar_note", "en", "$.plan_note"), "INSERT IGNORE")

	_, err = p.DownTo(ctx, pregnancyCopyVersion-1)
	require.NoError(t, err)
	assert.Equal(t, oldFa, msgField(t, db, "pregnancy_alert", "week_entered", "fa", "$.what_we_saw"))
	assert.Equal(t, "Custom", msgField(t, db, "pregnancy_alert", "week_entered", "en", "$.what_we_saw"))
	assert.Equal(t, "<null>", msgField(t, db, "pregnancy_alert", "weight_missing_week", "en", "$.params.from_weekday"))
	assert.Equal(t, "3", msgField(t, db, "pregnancy_alert", "weight_missing_week", "fa", "$.params.from_week"))
	assert.Equal(t, "", msgField(t, db, "pregnancy_setup", "calendar_note", "fa", "$.plan_note"), "seeded row removed")
	assert.Equal(t, "Admin note.", msgField(t, db, "pregnancy_setup", "calendar_note", "en", "$.plan_note"), "admin row kept")

	// A seeded row an admin edited after Up is kept by Down.
	exec(t, db, "DELETE FROM message_contents WHERE `group` = 'pregnancy_setup' AND item_key = 'calendar_note' AND locale = 'en'")
	_, err = p.UpByOne(ctx)
	require.NoError(t, err)
	exec(t, db, "UPDATE message_contents SET payload = JSON_SET(payload, '$.plan_note', 'Edited'), updated_at = '2030-01-01 00:00:00' "+
		"WHERE `group` = 'pregnancy_setup' AND item_key = 'calendar_note' AND locale = 'fa'")
	_, err = p.DownTo(ctx, pregnancyCopyVersion-1)
	require.NoError(t, err)
	assert.Equal(t, "Edited", msgField(t, db, "pregnancy_setup", "calendar_note", "fa", "$.plan_note"))
	assert.Equal(t, "", msgField(t, db, "pregnancy_setup", "calendar_note", "en", "$.plan_note"), "untouched seed removed")
}
