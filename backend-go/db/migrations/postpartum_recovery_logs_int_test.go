package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const postpartumRecoveryLogsVersion int64 = 97 // 00097_postpartum_recovery_logs.sql

// T-M9-06: Down drops postpartum_recovery_logs and Up recreates it; one row per (user, day), NULL = not answered, a
// temperature keeps one decimal, and deleting the user deletes her rows.
func TestPostpartumRecoveryLogsMigration_UpDown(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	exists := func() int {
		var n int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables
			WHERE table_schema = DATABASE() AND table_name = 'postpartum_recovery_logs'`).Scan(&n))
		return n
	}
	require.Equal(t, 1, exists())

	res, err := db.ExecContext(ctx, `INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Mother', '09129990097', NOW(), NOW())`)
	require.NoError(t, err)
	uid, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO postpartum_recovery_logs (user_id, log_date, temperature_c, breathless) VALUES (?, '2026-10-21', 38.04, 1)`, uid)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO postpartum_recovery_logs (user_id, log_date) VALUES (?, '2026-10-21')`, uid)
	require.Error(t, err, "one row per (user, day)")
	var temp string
	var pad *int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT temperature_c, pad_soaked_within_hour FROM postpartum_recovery_logs WHERE user_id = ?`, uid).Scan(&temp, &pad))
	assert.Equal(t, "38.0", temp)
	assert.Nil(t, pad)

	_, err = db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, uid)
	require.NoError(t, err)
	var n int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM postpartum_recovery_logs WHERE user_id = ?`, uid).Scan(&n))
	assert.Zero(t, n, "cascade on user delete")

	_, err = p.DownTo(ctx, postpartumRecoveryLogsVersion-1)
	require.NoError(t, err)
	assert.Zero(t, exists())
	_, err = p.Up(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, exists())
}
