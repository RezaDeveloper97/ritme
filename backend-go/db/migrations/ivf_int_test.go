package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const ivfVersion int64 = 27 // 00027_ivf.sql

// CB-IVF-01: Up creates the six ivf_* tables and the 36 catalog rows (all needs_review); one open cycle per user;
// Down removes exactly them and Up recreates them.
func TestIVFMigration_UpDown(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	count := func(query string, args ...any) int {
		var n int
		require.NoError(t, db.QueryRowContext(ctx, query, args...).Scan(&n))
		return n
	}
	tables := func() int {
		return count(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name IN
			('ivf_cycles', 'ivf_meds', 'ivf_dose_logs', 'ivf_scans', 'ivf_tww_logs', 'ivf_reminders')`)
	}
	seeds := func() int { return count("SELECT COUNT(*) FROM catalog_items WHERE `group` LIKE 'ivf\\_%'") }
	require.Equal(t, 6, tables())
	require.Equal(t, 36, seeds())
	assert.Zero(t, count("SELECT COUNT(*) FROM catalog_items WHERE `group` LIKE 'ivf\\_%' AND needs_review = 0"))
	assert.Equal(t, 8, count("SELECT COUNT(*) FROM catalog_items WHERE `group` = 'ivf_injection_sites'"))

	res, err := db.ExecContext(ctx, `INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('IVF', '09129990027', NOW(), NOW())`)
	require.NoError(t, err)
	uid, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO ivf_cycles (user_id, active_user_id, started_on) VALUES (?, ?, '2026-09-10')`, uid, uid)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO ivf_cycles (user_id, active_user_id, started_on) VALUES (?, ?, '2026-09-11')`, uid, uid)
	require.Error(t, err, "one open cycle per user")
	_, err = db.ExecContext(ctx, `INSERT INTO ivf_cycles (user_id, active_user_id, started_on) VALUES (?, NULL, '2026-08-01')`, uid)
	require.NoError(t, err, "closed cycles are unlimited")

	_, err = p.DownTo(ctx, ivfVersion-1)
	require.NoError(t, err)
	assert.Zero(t, tables())
	assert.Zero(t, seeds())

	_, err = p.Up(ctx)
	require.NoError(t, err)
	assert.Equal(t, 6, tables())
	assert.Equal(t, 36, seeds())
}
