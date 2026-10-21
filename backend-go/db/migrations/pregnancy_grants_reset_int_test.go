package migrations_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const pregnancyGrantsResetVersion int64 = 95 // 00095_reset_stale_pregnancy_grants.sql

// T-M9-02 (FAM-H3): the data migration removes every pregnancy grant older than the owner's current pregnancy episode
// and every pregnancy grant of an owner who is not pregnant now, with one audit row each; a grant given during the
// current pregnancy (and every other section's grant) is kept. Down is a no-op.
func TestPregnancyGrantsResetMigration(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })
	_, err = p.DownTo(ctx, pregnancyGrantsResetVersion-1)
	require.NoError(t, err)

	exec := func(q string, args ...any) int64 {
		t.Helper()
		res, err := db.ExecContext(ctx, q, args...)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}
	user := func(mobile string) int64 {
		return exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('U', ?, '2026-01-01 09:00:00', '2026-01-01 09:00:00')`, mobile)
	}
	members := 0
	link := func(owner int64, grants map[string]string) int64 {
		members++
		member := user(fmt.Sprintf("091299960%02d", members))
		cid := exec(`INSERT INTO companions (owner_id, companion_user_id, type, status, created_at, updated_at)
			VALUES (?, ?, 'partner', 'active', '2026-01-01 09:00:00', '2026-01-01 09:00:00')`, owner, member)
		for sec, at := range grants {
			exec(`INSERT INTO companion_grants (companion_id, section, level, created_at, updated_at) VALUES (?, ?, 'view', ?, ?)`, cid, sec, at, at)
		}
		return cid
	}

	// Stale: granted 2026-03 for the earlier pregnancy, a loss in 2026-05, pregnant again (onboarded 2026-08).
	stale := user("09129995001")
	exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, onboarding_completed, onboarding_completed_at, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2026-07-20', 1, '2026-08-10 09:00:00', '2026-02-01 09:00:00', '2026-08-10 09:00:00')`, stale)
	exec(`INSERT INTO pregnancy_losses (user_id, loss_type, created_at, updated_at) VALUES (?, 'unspecified', '2026-05-01 09:00:00', '2026-05-01 09:00:00')`, stale)
	staleLink := link(stale, map[string]string{"pregnancy": "2026-03-01 09:00:00", "cycle": "2026-03-01 09:00:00"})

	// Kept: granted during the current pregnancy (after a loss before it).
	current := user("09129995002")
	exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, onboarding_completed, onboarding_completed_at, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2026-07-20', 1, '2026-08-10 09:00:00', '2026-02-01 09:00:00', '2026-08-10 09:00:00')`, current)
	exec(`INSERT INTO pregnancy_losses (user_id, loss_type, created_at, updated_at) VALUES (?, 'unspecified', '2026-05-01 09:00:00', '2026-05-01 09:00:00')`, current)
	keptLink := link(current, map[string]string{"pregnancy": "2026-08-20 09:00:00"})

	// Not pregnant now: postpartum (pregnancy closed by the birth) and a cycle-mode owner who granted in advance.
	mother := user("09129995003")
	exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 0, 'lmp', '2025-12-01', '2025-12-10 09:00:00', '2026-09-01 09:00:00')`, mother)
	motherLink := link(mother, map[string]string{"pregnancy": "2026-01-15 09:00:00"})
	cycler := user("09129995004")
	cyclerLink := link(cycler, map[string]string{"pregnancy": "2026-09-01 09:00:00", "meds": "2026-09-01 09:00:00"})

	_, err = p.UpTo(ctx, pregnancyGrantsResetVersion)
	require.NoError(t, err)

	has := func(cid int64, sec string) bool {
		t.Helper()
		var n int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM companion_grants WHERE companion_id = ? AND section = ?`, cid, sec).Scan(&n))
		return n > 0
	}
	audits := func(cid int64) int {
		t.Helper()
		var n int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM companion_audit_logs
			WHERE companion_id = ? AND section = 'pregnancy' AND action = 'grants_changed' AND actor_id IS NULL`, cid).Scan(&n))
		return n
	}
	assert.False(t, has(staleLink, "pregnancy"), "a grant of an earlier pregnancy is reset")
	assert.True(t, has(staleLink, "cycle"), "other sections are untouched")
	assert.Equal(t, 1, audits(staleLink))
	assert.True(t, has(keptLink, "pregnancy"), "a grant given during the current pregnancy is kept")
	assert.Zero(t, audits(keptLink))
	assert.False(t, has(motherLink, "pregnancy"), "postpartum is not pregnancy")
	assert.Equal(t, 1, audits(motherLink))
	assert.False(t, has(cyclerLink, "pregnancy"), "not pregnant now")
	assert.True(t, has(cyclerLink, "meds"))
	assert.Equal(t, 1, audits(cyclerLink))

	// Down is a no-op: nothing comes back.
	_, err = p.DownTo(ctx, pregnancyGrantsResetVersion-1)
	require.NoError(t, err)
	assert.False(t, has(staleLink, "pregnancy"))
	var n sql.NullInt64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM companion_audit_logs WHERE action = 'grants_changed'`).Scan(&n))
	assert.Equal(t, int64(3), n.Int64, "the audit rows stay")
}
