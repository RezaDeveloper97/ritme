package migrations_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const familyGrantsV6Version int64 = 110 // 00110_family_grants_v6.sql

// T-M11-05: legacy grants → v6 (family.md §3): symptoms deleted with one owner notice per owner (F-03), none deleted,
// cycle → cycle_phase view, pregnancy edit → view, meds / appointments edit → write; owner-data grants stamped with
// epoch 1, a pregnancy grant only when the owner is pregnant now; teen sections untouched; idempotent; Down maps back.
func TestFamilyGrantsV6Migration(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })
	_, err = p.DownTo(ctx, familyGrantsV6Version-1)
	require.NoError(t, err)

	exec := func(q string, args ...any) int64 {
		t.Helper()
		res, err := db.ExecContext(ctx, q, args...)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}
	n := 0
	user := func() int64 {
		n++
		return exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('U', ?, NOW(), NOW())`, fmt.Sprintf("0912777%04d", n))
	}
	link := func(owner int64, typ, status string, grants map[string]string) int64 {
		member := user()
		id := exec(`INSERT INTO companions (owner_id, companion_user_id, type, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, NOW(), NOW())`, owner, member, typ, status)
		for s, l := range grants {
			exec(`INSERT INTO companion_grants (companion_id, section, level, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())`, id, s, l)
		}
		return id
	}
	pregnant, notPregnant, teen := user(), user(), user()
	exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2026-08-01', NOW(), NOW())`, pregnant)
	p1 := link(pregnant, "spouse", "active", map[string]string{"cycle": "edit", "symptoms": "view", "meds": "edit",
		"appointments": "view", "pregnancy": "edit"})
	p2 := link(pregnant, "partner", "active", map[string]string{"symptoms": "edit", "meds": "none"})
	n1 := link(notPregnant, "partner", "active", map[string]string{"pregnancy": "view", "appointments": "edit", "cycle": "view"})
	r1 := link(notPregnant, "partner", "revoked", map[string]string{"symptoms": "view"}) // revoked: no notice for it alone
	t1 := link(teen, "parent", "active", map[string]string{"teen_kit": "view", "teen_notes": "view"})

	_, err = p.UpTo(ctx, familyGrantsV6Version)
	require.NoError(t, err)

	type g struct {
		level string
		epoch any
	}
	grants := func(id int64) map[string]g {
		t.Helper()
		rows, err := db.QueryContext(ctx, `SELECT section, level, mode_epoch FROM companion_grants WHERE companion_id = ?`, id)
		require.NoError(t, err)
		defer func() { _ = rows.Close() }()
		out := map[string]g{}
		for rows.Next() {
			var s, l string
			var e *int64
			require.NoError(t, rows.Scan(&s, &l, &e))
			var ev any
			if e != nil {
				ev = *e
			}
			out[s] = g{l, ev}
		}
		require.NoError(t, rows.Err())
		return out
	}
	assert.Equal(t, map[string]g{"cycle_phase": {"view", int64(1)}, "meds": {"write", int64(1)},
		"appointments": {"view", int64(1)}, "pregnancy": {"view", int64(1)}}, grants(p1))
	assert.Equal(t, map[string]g{}, grants(p2), "symptoms deleted (F-03), none deleted")
	assert.Equal(t, map[string]g{"pregnancy": {"view", nil}, "appointments": {"write", int64(1)},
		"cycle_phase": {"view", int64(1)}}, grants(n1), "a pregnancy grant of an owner not pregnant now stays inactive")
	assert.Equal(t, map[string]g{}, grants(r1))
	assert.Equal(t, map[string]g{"teen_kit": {"view", nil}, "teen_notes": {"view", nil}}, grants(t1), "teen card unchanged")

	notices := func(owner int64) int {
		return countRows(t, db, `SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND type = 'companion'
			AND JSON_VALUE(data, '$.event') = 'symptoms_sharing_ended'`, owner)
	}
	assert.Equal(t, 1, notices(pregnant), "one notice per owner, however many links held symptoms")
	assert.Equal(t, 0, notices(notPregnant), "a revoked link's symptoms grant ends silently")
	assert.Equal(t, 0, notices(teen))
	var title string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT title FROM user_notifications WHERE user_id = ?`, pregnant).Scan(&title))
	assert.NotEmpty(t, title)

	// Idempotent: a forgotten goose bookkeeping row re-runs Up without a second notice or a changed grant.
	_, err = db.ExecContext(ctx, `DELETE FROM goose_db_version WHERE version_id >= ?`, familyGrantsV6Version)
	require.NoError(t, err)
	_, err = p.UpTo(ctx, familyGrantsV6Version)
	require.NoError(t, err)
	assert.Equal(t, 1, notices(pregnant))
	assert.Equal(t, "write", grants(p1)["meds"].level)

	// Down maps back to the legacy codes.
	exec(`INSERT INTO companion_grants (companion_id, section, level, mode_epoch, created_at, updated_at) VALUES (?, 'vitals', 'view', 1, NOW(), NOW())`, p2)
	_, err = p.DownTo(ctx, familyGrantsV6Version-1)
	require.NoError(t, err)
	assert.Equal(t, map[string]g{"cycle": {"view", nil}, "meds": {"edit", nil}, "appointments": {"view", nil},
		"pregnancy": {"view", nil}}, grants(p1))
	assert.Equal(t, map[string]g{}, grants(p2), "v6-only domains are dropped")
	assert.Equal(t, 0, notices(pregnant))
}
