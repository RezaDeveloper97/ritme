package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const companionsVersion int64 = 26 // 00026_companions.sql (00025 reserved by a parallel task)

var companionTables = []string{"companions", "companion_invites", "companion_grants", "families", "family_children", "companion_audit_logs"}

// B-N4-01: Down drops exactly the six companion tables (with their FKs) and Up recreates them.
func TestCompanionsMigration_UpDown(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	tables := func() int {
		var n int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables
			WHERE table_schema = DATABASE() AND table_name IN (?, ?, ?, ?, ?, ?)`,
			companionTables[0], companionTables[1], companionTables[2], companionTables[3], companionTables[4], companionTables[5]).Scan(&n))
		return n
	}
	require.Equal(t, len(companionTables), tables())

	// A link with every child row, so Down has foreign keys to drop through.
	res, err := db.ExecContext(ctx, `INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Owner', '09129990001', NOW(), NOW())`)
	require.NoError(t, err)
	owner, err := res.LastInsertId()
	require.NoError(t, err)
	res, err = db.ExecContext(ctx, `INSERT INTO companions (owner_id, type, status, created_at, updated_at) VALUES (?, 'spouse', 'invited', NOW(), NOW())`, owner)
	require.NoError(t, err)
	cid, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO companion_invites (companion_id, owner_id, code_hash, expires_at, created_at, updated_at) VALUES (?, ?, REPEAT('a', 64), NOW(), NOW(), NOW())`, cid, owner)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO companion_grants (companion_id, section, level) VALUES (?, 'cycle', 'view')`, cid)
	require.NoError(t, err)
	res, err = db.ExecContext(ctx, `INSERT INTO families (owner_id, companion_id) VALUES (?, ?)`, owner, cid)
	require.NoError(t, err)
	fid, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO family_children (family_id, child_id) VALUES (?, 42)`, fid) // no children FK until B-N5-02
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO companion_audit_logs (owner_id, actor_id, companion_id, action, created_at) VALUES (?, ?, ?, 'invited', NOW())`, owner, owner, cid)
	require.NoError(t, err)

	// The code hash is unique; a second (owner, section) grant is refused.
	_, err = db.ExecContext(ctx, `INSERT INTO companion_invites (companion_id, owner_id, code_hash, created_at) VALUES (?, ?, REPEAT('a', 64), NOW())`, cid, owner)
	require.Error(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO companion_grants (companion_id, section, level) VALUES (?, 'cycle', 'edit')`, cid)
	require.Error(t, err)

	_, err = p.DownTo(ctx, companionsVersion-1)
	require.NoError(t, err)
	assert.Zero(t, tables())
	var users int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE id = ?`, owner).Scan(&users))
	assert.Equal(t, 1, users, "Down leaves users alone")

	_, err = p.Up(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(companionTables), tables())
}
