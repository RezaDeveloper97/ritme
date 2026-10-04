package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const babyLogsVersion int64 = 33 // 00033_baby_logs.sql

// B-N5-03: Up creates the six tables and the contractions_511 rule rows; one running session per child / user is a
// unique index; Down drops the tables and the untouched rule rows but keeps an admin-edited one; Up again re-seeds.
func TestBabyLogsMigration_UpDown(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	tables := func() int {
		var n int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE()
			AND table_name IN ('baby_feeds', 'baby_sleeps', 'baby_diapers', 'pregnancy_kick_sessions', 'pregnancy_contraction_sessions', 'pregnancy_contractions')`).Scan(&n))
		return n
	}
	rules := func() int {
		var n int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM message_contents WHERE `+"`group`"+` = 'pregnancy_alert' AND item_key = 'contractions_511'`).Scan(&n))
		return n
	}
	require.Equal(t, 6, tables())
	require.Equal(t, 2, rules())

	res, err := db.ExecContext(ctx, `INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('A', '09129990034', NOW(), NOW())`)
	require.NoError(t, err)
	uid, _ := res.LastInsertId()
	res, err = db.ExecContext(ctx, `INSERT INTO children (owner_id, name, birth_date, created_at, updated_at) VALUES (?, 'Ava', '2026-06-20', NOW(), NOW())`, uid)
	require.NoError(t, err)
	cid, _ := res.LastInsertId()
	feed := `INSERT INTO baby_feeds (child_id, type, started_at, active_lock) VALUES (?, 'breast', NOW(), ?)`
	_, err = db.ExecContext(ctx, feed, cid, 1)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, feed, cid, 1)
	require.Error(t, err, "a second running feed for the child")
	_, err = db.ExecContext(ctx, feed, cid, nil)
	require.NoError(t, err, "ended feeds are unlimited")
	_, err = db.ExecContext(ctx, feed, cid, nil)
	require.NoError(t, err)

	// An admin edit survives a rollback.
	_, err = db.ExecContext(ctx, `UPDATE message_contents SET payload = JSON_SET(payload, '$.title', 'Edited'), updated_at = DATE_ADD(updated_at, INTERVAL 1 MINUTE)
		WHERE `+"`group`"+` = 'pregnancy_alert' AND item_key = 'contractions_511' AND locale = 'en'`)
	require.NoError(t, err)
	_, err = p.DownTo(ctx, babyLogsVersion-1)
	require.NoError(t, err)
	assert.Zero(t, tables())
	assert.Equal(t, 1, rules(), "only the edited row is kept")
	_, err = p.Up(ctx)
	require.NoError(t, err)
	assert.Equal(t, 6, tables())
	assert.Equal(t, 2, rules())
}
