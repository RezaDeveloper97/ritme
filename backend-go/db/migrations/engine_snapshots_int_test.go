package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const engineSnapshotsVersion int64 = 109 // 00109_engine_snapshots.sql

// T-M9-39 (D-137): Up creates both tables (PK / UNIQUE keys hold, the kind enum refuses other values), Down drops
// them, Up runs again — also when the tables already exist (the Laravel twin ran first).
func TestEngineSnapshotsMigration_UpDown(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	tables := func() int {
		var n int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE()
			AND table_name IN ('engine_daily_snapshots', 'engine_prediction_log')`).Scan(&n))
		return n
	}
	require.Equal(t, 2, tables())

	res, err := db.ExecContext(ctx, `INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('U', '09129991091', NOW(), NOW())`)
	require.NoError(t, err)
	uid, err := res.LastInsertId()
	require.NoError(t, err)

	snap := `INSERT INTO engine_daily_snapshots (user_id, day, main_phase, subphase, confidence, data_quality, mode) VALUES (?, '2026-10-27', 'luteal', 'mid_luteal', 'high', 'good', 'cycle')`
	_, err = db.ExecContext(ctx, snap, uid)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, snap, uid)
	require.Error(t, err, "PK (user_id, day)")

	pred := `INSERT INTO engine_prediction_log (user_id, cycle_start, kind, predicted_next, basis_cycles, ovulation_source, confidence) VALUES (?, '2026-10-01', ?, '2026-10-29', 2, 'calendar', 'high')`
	_, err = db.ExecContext(ctx, pred, uid, "first")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, pred, uid, "last")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, pred, uid, "first")
	require.Error(t, err, "UNIQUE (user_id, cycle_start, kind)")
	_, err = db.ExecContext(ctx, pred, uid, "other")
	require.Error(t, err, "kind is first | last")

	_, err = p.DownTo(ctx, engineSnapshotsVersion-1)
	require.NoError(t, err)
	assert.Zero(t, tables())

	_, err = p.Up(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, tables())

	// Idempotent Up over existing tables (the Laravel twin ran first): forget the version and apply again.
	_, err = db.ExecContext(ctx, `DELETE FROM goose_db_version WHERE version_id >= ?`, engineSnapshotsVersion)
	require.NoError(t, err)
	_, err = p.Up(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, tables())
}
