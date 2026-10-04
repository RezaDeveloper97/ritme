package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const aiPlatformVersion int64 = 32 // 00032_ai_platform.sql

// B-N6-05: Down drops ai_usage_logs and user_consents.version (keeping the consent rows); Up restores both; a
// deleted user leaves the usage row without its user (SET NULL).
func TestAIPlatformMigration_UpDown(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	state := func() (tables, cols int) {
		require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables
			WHERE table_schema = DATABASE() AND table_name = 'ai_usage_logs'`).Scan(&tables))
		require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns
			WHERE table_schema = DATABASE() AND table_name = 'user_consents' AND column_name = 'version'`).Scan(&cols))
		return tables, cols
	}
	tb, c := state()
	require.Equal(t, 1, tb)
	require.Equal(t, 1, c)

	res, err := db.ExecContext(ctx, `INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('A', '09129990033', NOW(), NOW())`)
	require.NoError(t, err)
	uid, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO user_consents (user_id, consent, granted, version, created_at, updated_at) VALUES (?, 'ai_assistant', 1, 1, NOW(), NOW())`, uid)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO ai_usage_logs (user_id, feature, op, provider, model, created_at) VALUES (?, 'assistant', 'chat', 'fake', 'm', NOW())`, uid)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, uid)
	require.NoError(t, err)
	var n int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ai_usage_logs WHERE user_id IS NULL`).Scan(&n))
	assert.Equal(t, 1, n)

	_, err = p.DownTo(ctx, aiPlatformVersion-1)
	require.NoError(t, err)
	tb, c = state()
	assert.Zero(t, tb)
	assert.Zero(t, c)
	_, err = p.Up(ctx)
	require.NoError(t, err)
	tb, c = state()
	assert.Equal(t, 1, tb)
	assert.Equal(t, 1, c)
}
