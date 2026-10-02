package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const postpartumVersion int64 = 30 // 00030_postpartum.sql (00027–00029 belong to parallel tasks)

// B-N5-01: Down drops exactly the two postpartum tables and Up recreates them; one profile per user, one check per
// (user, kind, day); a deleted pregnancy profile leaves the postpartum profile (SET NULL).
func TestPostpartumMigration_UpDown(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	tables := func() int {
		var n int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables
			WHERE table_schema = DATABASE() AND table_name IN ('postpartum_profiles', 'epds_checks')`).Scan(&n))
		return n
	}
	require.Equal(t, 2, tables())

	res, err := db.ExecContext(ctx, `INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Mother', '09129990030', NOW(), NOW())`)
	require.NoError(t, err)
	uid, err := res.LastInsertId()
	require.NoError(t, err)
	res, err = db.ExecContext(ctx, `INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, created_at, updated_at) VALUES (?, 0, NOW(), NOW())`, uid)
	require.NoError(t, err)
	pid, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO postpartum_profiles (user_id, birth_date, source, pregnancy_profile_id) VALUES (?, '2026-09-14', 'pregnancy', ?)`, uid, pid)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO postpartum_profiles (user_id, birth_date) VALUES (?, '2026-09-15')`, uid)
	require.Error(t, err, "one profile per user")
	_, err = db.ExecContext(ctx, `INSERT INTO epds_checks (user_id, kind, taken_on, answers, total) VALUES (?, 'short', '2026-09-20', '{}', 0)`, uid)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO epds_checks (user_id, kind, taken_on, answers, total) VALUES (?, 'short', '2026-09-20', '{}', 1)`, uid)
	require.Error(t, err, "one check per (user, kind, day)")

	_, err = db.ExecContext(ctx, `DELETE FROM pregnancy_profiles WHERE id = ?`, pid)
	require.NoError(t, err)
	var linked *int64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT pregnancy_profile_id FROM postpartum_profiles WHERE user_id = ?`, uid).Scan(&linked))
	assert.Nil(t, linked)

	_, err = p.DownTo(ctx, postpartumVersion-1)
	require.NoError(t, err)
	assert.Zero(t, tables())
	_, err = p.Up(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, tables())
}
