package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const childrenVersion int64 = 31 // 00031_children.sql

// B-N5-02: Up creates the four children tables, the family_children → children FK and the child_* catalog seed;
// deleting a child cascades to its rows and family shares; Down removes exactly that and Up restores it.
func TestChildrenMigration_UpDown(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	count := func(q string, args ...any) int {
		var n int
		require.NoError(t, db.QueryRowContext(ctx, q, args...).Scan(&n))
		return n
	}
	tables := func() int {
		return count(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE()
			AND table_name IN ('children', 'child_measurements', 'child_vaccine_doses', 'child_milestone_checks')`)
	}
	fk := func() int {
		return count(`SELECT COUNT(*) FROM information_schema.referential_constraints WHERE constraint_schema = DATABASE()
			AND constraint_name = 'family_children_child_id_foreign'`)
	}
	seed := func() int { return count("SELECT COUNT(*) FROM catalog_items WHERE `group` LIKE 'child\\_%'") }
	require.Equal(t, 4, tables())
	require.Equal(t, 1, fk())
	require.Equal(t, 132, seed())
	assert.Equal(t, 21, count("SELECT COUNT(*) FROM catalog_items WHERE `group` = 'child_vaccines'"))

	res, err := db.ExecContext(ctx, `INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Mother', '09129990031', NOW(), NOW())`)
	require.NoError(t, err)
	owner, err := res.LastInsertId()
	require.NoError(t, err)
	res, err = db.ExecContext(ctx, `INSERT INTO children (owner_id, name, birth_date) VALUES (?, 'Ava', '2026-06-20')`, owner)
	require.NoError(t, err)
	child, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO child_measurements (child_id, measured_on, weight_kg) VALUES (?, '2026-09-01', 5.1)`, child)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO child_measurements (child_id, measured_on, weight_kg) VALUES (?, '2026-09-01', 5.2)`, child)
	require.Error(t, err, "one measurement per day")
	_, err = db.ExecContext(ctx, `INSERT INTO child_vaccine_doses (child_id, dose_code, given_on) VALUES (?, 'bcg', '2026-06-20')`, child)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO child_milestone_checks (child_id, milestone_code, checked_on) VALUES (?, 'm2_coos', '2026-08-25')`, child)
	require.NoError(t, err)
	res, err = db.ExecContext(ctx, `INSERT INTO companions (owner_id, type, status, created_at, updated_at) VALUES (?, 'spouse', 'active', NOW(), NOW())`, owner)
	require.NoError(t, err)
	cid, err := res.LastInsertId()
	require.NoError(t, err)
	res, err = db.ExecContext(ctx, `INSERT INTO families (owner_id, companion_id) VALUES (?, ?)`, owner, cid)
	require.NoError(t, err)
	fid, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO family_children (family_id, child_id) VALUES (?, 999999)`, fid)
	require.Error(t, err, "a share must reference a child")
	_, err = db.ExecContext(ctx, `INSERT INTO family_children (family_id, child_id) VALUES (?, ?)`, fid, child)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `DELETE FROM children WHERE id = ?`, child)
	require.NoError(t, err)
	for _, tbl := range []string{"child_measurements", "child_vaccine_doses", "child_milestone_checks", "family_children"} {
		assert.Zero(t, count("SELECT COUNT(*) FROM "+tbl), tbl+" cascades")
	}

	_, err = p.DownTo(ctx, childrenVersion-1)
	require.NoError(t, err)
	assert.Zero(t, tables())
	assert.Zero(t, fk())
	assert.Zero(t, seed())
	_, err = p.Up(ctx)
	require.NoError(t, err)
	assert.Equal(t, 4, tables())
	assert.Equal(t, 1, fk())
	assert.Equal(t, 132, seed())
}
