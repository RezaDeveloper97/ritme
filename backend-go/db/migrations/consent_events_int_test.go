package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const consentEventsVersion int64 = 101 // 00101_consent_events.sql

// T-M10-23: Up backfills the ledger from user_consents (a grant per granted or once-granted row at granted_at, a
// revoke at revoked_at, nothing for a never-granted «no»), oldest first; running Up again on a database that already
// has events does not duplicate them; the triggers refuse UPDATE / DELETE; deleting a user cascades; Down drops it all.
func TestConsentEventsMigration_BackfillTriggersDown(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	_, err = p.DownTo(ctx, consentEventsVersion-1)
	require.NoError(t, err)

	user := func(mobile string) int64 {
		res, err := db.ExecContext(ctx, `INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('U', ?, NOW(), NOW())`, mobile)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}
	a, b := user("09129990101"), user("09129990102")
	_, err = db.ExecContext(ctx, `INSERT INTO user_consents (user_id, consent, granted, version, granted_at, revoked_at, created_at, updated_at) VALUES
		(?, 'ai_assistant',      1, 1,    '2026-10-01 10:00:00', NULL,                  '2026-10-01 10:00:00', '2026-10-01 10:00:00'),
		(?, 'ai_lab_analysis',   0, 1,    '2026-10-02 10:00:00', '2026-10-03 10:00:00', '2026-10-02 10:00:00', '2026-10-03 10:00:00'),
		(?, 'anonymous_stats',   0, NULL, NULL,                  NULL,                  '2026-10-04 10:00:00', '2026-10-04 10:00:00'),
		(?, 'assistant_profile', 1, NULL, '2026-09-01 10:00:00', NULL,                  '2026-09-01 10:00:00', '2026-09-01 10:00:00')`, a, a, a, b)
	require.NoError(t, err)

	_, err = p.Up(ctx)
	require.NoError(t, err)

	events := func(uid int64) []string {
		rows, err := db.QueryContext(ctx, `SELECT CONCAT(consent, ':', version, ':', action, ':', source, ':', DATE_FORMAT(created_at, '%Y-%m-%d'), ':', IFNULL(scope, '-'))
			FROM consent_events WHERE user_id = ? ORDER BY id`, uid)
		require.NoError(t, err)
		defer func() { _ = rows.Close() }()
		var out []string
		for rows.Next() {
			var s string
			require.NoError(t, rows.Scan(&s))
			out = append(out, s)
		}
		require.NoError(t, rows.Err())
		return out
	}
	assert.Equal(t, []string{
		"ai_assistant:1:grant:backfill:2026-10-01:-",
		"ai_lab_analysis:1:grant:backfill:2026-10-02:-",
		"ai_lab_analysis:1:revoke:backfill:2026-10-03:-",
	}, events(a), "the never-granted «no» has no event")
	assert.Equal(t, []string{"assistant_profile:0:grant:backfill:2026-09-01:-"}, events(b), "an unversioned grant is version 0")

	// A second backfill (the Laravel twin ran first, then goose at cutover) skips pairs that already have events.
	_, err = db.ExecContext(ctx, `DROP TRIGGER consent_events_no_delete`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `DROP TRIGGER consent_events_no_update`)
	require.NoError(t, err)
	// Later migrations are forgotten too (goose refuses a missing version below the head); each is idempotent, as
	// every migration whose Laravel twin may have run first must be.
	_, err = db.ExecContext(ctx, `DELETE FROM goose_db_version WHERE version_id >= ?`, consentEventsVersion)
	require.NoError(t, err)
	_, err = p.Up(ctx) // CREATE TABLE / TRIGGER IF NOT EXISTS, guarded backfill
	require.NoError(t, err)
	assert.Len(t, events(a), 3)
	assert.Len(t, events(b), 1)

	var n int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.triggers WHERE trigger_schema = DATABASE() AND event_object_table = 'consent_events'`).Scan(&n))
	assert.Equal(t, 2, n)
	_, err = db.ExecContext(ctx, `UPDATE consent_events SET version = 9 WHERE user_id = ?`, a)
	require.ErrorContains(t, err, "append-only")
	_, err = db.ExecContext(ctx, `DELETE FROM consent_events WHERE user_id = ?`, a)
	require.ErrorContains(t, err, "append-only")
	_, err = db.ExecContext(ctx, `INSERT INTO consent_events (user_id, consent, version, action, source, created_at) VALUES (?, 'terms', 1, 'grant', 'signup', NOW())`, a)
	require.NoError(t, err, "appending works")

	_, err = db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, a)
	require.NoError(t, err, "a user delete cascades past the triggers")
	assert.Empty(t, events(a))
	assert.Len(t, events(b), 1)

	_, err = p.DownTo(ctx, consentEventsVersion-1)
	require.NoError(t, err)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'consent_events'`).Scan(&n))
	assert.Zero(t, n)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.triggers WHERE trigger_schema = DATABASE() AND event_object_table = 'consent_events'`).Scan(&n))
	assert.Zero(t, n)
	_, err = p.Up(ctx)
	require.NoError(t, err)
}
