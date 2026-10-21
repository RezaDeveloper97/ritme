package migrations_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/db"
	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const userProfilesUniqueVersion int64 = 96 // 00096_user_profiles_user_id_unique.sql

type profileRow struct {
	ID                          int64
	UserID                      int64
	Birthday, Weight, Intention sql.NullString
	Height, Cycle               sql.NullInt64
	Conditions                  sql.NullString
	Goal, Subscription          string
	Version                     int
	CreatedAt, UpdatedAt        sql.NullString
}

const profileCols = `id, user_id, DATE_FORMAT(birthday, '%Y-%m-%d'), weight, pregnancy_intention, height, cycle_duration,
	chronic_conditions, user_goal, subscription_type, calculation_version,
	DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s'), DATE_FORMAT(updated_at, '%Y-%m-%d %H:%i:%s')`

func scanProfiles(t *testing.T, conn *sql.DB, table string, userID int64) []profileRow {
	t.Helper()
	rows, err := conn.Query(`SELECT `+profileCols+` FROM `+table+` WHERE user_id = ? ORDER BY id`, userID)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var out []profileRow
	for rows.Next() {
		var r profileRow
		require.NoError(t, rows.Scan(&r.ID, &r.UserID, &r.Birthday, &r.Weight, &r.Intention, &r.Height, &r.Cycle,
			&r.Conditions, &r.Goal, &r.Subscription, &r.Version, &r.CreatedAt, &r.UpdatedAt))
		out = append(out, r)
	}
	require.NoError(t, rows.Err())
	return out
}

func indexNames(t *testing.T, conn *sql.DB) []string {
	t.Helper()
	rows, err := conn.Query(`SELECT DISTINCT index_name FROM information_schema.statistics
		WHERE table_schema = DATABASE() AND table_name = 'user_profiles' ORDER BY index_name`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var n string
		require.NoError(t, rows.Scan(&n))
		out = append(out, n)
	}
	require.NoError(t, rows.Err())
	return out
}

func tableExists(t *testing.T, conn *sql.DB, name string) bool {
	t.Helper()
	var n int
	require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`, name).Scan(&n))
	return n > 0
}

// upStatements is the goose Up section of 00096 split into statements, to prove a second run is harmless.
func upStatements(t *testing.T) []string {
	t.Helper()
	raw, err := db.Migrations.ReadFile("migrations/00096_user_profiles_user_id_unique.sql")
	require.NoError(t, err)
	up := strings.SplitN(strings.SplitN(string(raw), "-- +goose Up", 2)[1], "-- +goose Down", 2)[0]
	var lines []string
	for _, l := range strings.Split(up, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(l), "--") {
			lines = append(lines, l)
		}
	}
	var out []string
	for _, st := range strings.Split(strings.Join(lines, "\n"), ";\n") {
		if st = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(st), ";")); st != "" {
			out = append(out, st)
		}
	}
	return out
}

// T-M9-23 (D-127): duplicates are merged into the lowest id (NULL columns filled from the newest duplicate that has a
// value), every removed row is backed up, UNIQUE(user_id) replaces the FK index; a second run changes nothing; down
// restores the removed rows and the old index.
func TestUserProfilesUniqueMigration(t *testing.T) {
	ctx := context.Background()
	conn := testdb.New(t)
	p, err := platformdb.NewMigrator(conn)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })
	_, err = p.DownTo(ctx, userProfilesUniqueVersion-1)
	require.NoError(t, err)
	assert.Equal(t, []string{"PRIMARY", "user_profiles_user_id_foreign"}, indexNames(t, conn))

	exec := func(q string, args ...any) int64 {
		t.Helper()
		res, err := conn.ExecContext(ctx, q, args...)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}
	user := func(mobile string) int64 {
		return exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('U', ?, '2026-01-01 09:00:00', '2026-01-01 09:00:00')`, mobile)
	}

	// dup: three rows. Kept (lowest id) has a birthday and a goal but no weight / height / intention / conditions.
	// The middle row has a weight and a height, the newest has a weight, an intention and conditions, no height.
	dup := user("09129996001")
	kept := exec(`INSERT INTO user_profiles (user_id, birthday, cycle_duration, user_goal, subscription_type, calculation_version, created_at, updated_at)
		VALUES (?, '1995-04-01', 28, 'ttc', 'free', 2, '2026-02-01 09:00:00', '2026-02-01 09:00:00')`, dup)
	middle := exec(`INSERT INTO user_profiles (user_id, weight, height, cycle_duration, user_goal, subscription_type, created_at, updated_at)
		VALUES (?, 58.50, 165, 30, 'non_ttc', 'plus', '2026-02-01 09:00:01', '2026-02-02 09:00:00')`, dup)
	newest := exec(`INSERT INTO user_profiles (user_id, weight, pregnancy_intention, chronic_conditions, user_goal, created_at, updated_at)
		VALUES (?, 61.00, 'trying', '["pcos"]', 'non_ttc', '2026-02-01 09:00:02', '2026-02-03 09:00:00')`, dup)

	// clean: one row, untouched.
	clean := user("09129996002")
	exec(`INSERT INTO user_profiles (user_id, weight, created_at, updated_at) VALUES (?, 70.00, '2026-02-01 09:00:00', '2026-02-01 09:00:00')`, clean)
	cleanBefore := scanProfiles(t, conn, "user_profiles", clean)
	dupBefore := scanProfiles(t, conn, "user_profiles", dup)
	require.Len(t, dupBefore, 3)

	_, err = p.UpTo(ctx, userProfilesUniqueVersion)
	require.NoError(t, err)

	got := scanProfiles(t, conn, "user_profiles", dup)
	require.Len(t, got, 1, "one profile per user")
	k := got[0]
	assert.Equal(t, kept, k.ID, "the lowest id is kept")
	assert.Equal(t, "1995-04-01", k.Birthday.String, "a kept value is never overwritten")
	assert.Equal(t, int64(28), k.Cycle.Int64, "a kept value is never overwritten")
	assert.Equal(t, "61.00", k.Weight.String, "NULL filled from the newest duplicate")
	assert.Equal(t, int64(165), k.Height.Int64, "NULL filled from the newest duplicate that has a value")
	assert.Equal(t, "trying", k.Intention.String)
	assert.JSONEq(t, `["pcos"]`, k.Conditions.String)
	assert.Equal(t, "ttc", k.Goal, "NOT NULL columns keep the kept row's value")
	assert.Equal(t, "free", k.Subscription)
	assert.Equal(t, 2, k.Version)
	assert.Equal(t, "2026-02-01 09:00:00", k.UpdatedAt.String, "updated_at is not touched")

	backup := scanProfiles(t, conn, "user_profiles_dedupe_backup", dup)
	require.Len(t, backup, 2, "every removed row is backed up")
	assert.Equal(t, dupBefore[1:], backup, "the backup is the removed rows as they were")
	assert.Equal(t, []int64{middle, newest}, []int64{backup[0].ID, backup[1].ID})
	var missingRemovedAt int
	require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM user_profiles_dedupe_backup WHERE removed_at IS NULL`).Scan(&missingRemovedAt))
	assert.Zero(t, missingRemovedAt)

	assert.Equal(t, cleanBefore, scanProfiles(t, conn, "user_profiles", clean), "a single profile is untouched")
	assert.Empty(t, scanProfiles(t, conn, "user_profiles_dedupe_backup", clean), "a clean user has no backup row")
	assert.Equal(t, []string{"PRIMARY", "user_profiles_user_id_unique"}, indexNames(t, conn))
	_, err = conn.Exec(`INSERT INTO user_profiles (user_id, created_at, updated_at) VALUES (?, NOW(), NOW())`, clean)
	require.Error(t, err, "a second profile for a user is refused")

	// A second run of the Up statements (a re-run after a partial failure) changes nothing.
	for _, st := range upStatements(t) {
		_, err := conn.ExecContext(ctx, st)
		require.NoError(t, err, st)
	}
	assert.Equal(t, got, scanProfiles(t, conn, "user_profiles", dup))
	assert.Equal(t, backup, scanProfiles(t, conn, "user_profiles_dedupe_backup", dup))
	assert.Equal(t, cleanBefore, scanProfiles(t, conn, "user_profiles", clean))
	assert.Equal(t, []string{"PRIMARY", "user_profiles_user_id_unique"}, indexNames(t, conn))

	// Down: the removed rows come back with their ids and values, the FK index returns, the backup table goes.
	_, err = p.DownTo(ctx, userProfilesUniqueVersion-1)
	require.NoError(t, err)
	restored := scanProfiles(t, conn, "user_profiles", dup)
	require.Len(t, restored, 3)
	assert.Equal(t, dupBefore[1:], restored[1:], "both removed rows are restored as they were")
	assert.Equal(t, k, restored[0], "the kept row keeps its merged values")
	assert.Equal(t, cleanBefore, scanProfiles(t, conn, "user_profiles", clean))
	assert.Equal(t, []string{"PRIMARY", "user_profiles_user_id_foreign"}, indexNames(t, conn))
	assert.False(t, tableExists(t, conn, "user_profiles_dedupe_backup"))

	// Up again is deterministic: the same row is kept with the same values.
	_, err = p.UpTo(ctx, userProfilesUniqueVersion)
	require.NoError(t, err)
	assert.Equal(t, got, scanProfiles(t, conn, "user_profiles", dup))
}

// A clean table: the keys change, nothing else; the backup table exists and is empty; down is clean too.
func TestUserProfilesUniqueMigration_CleanTable(t *testing.T) {
	ctx := context.Background()
	conn := testdb.New(t)
	p, err := platformdb.NewMigrator(conn)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })
	_, err = p.DownTo(ctx, userProfilesUniqueVersion-1)
	require.NoError(t, err)

	res, err := conn.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('U', '09129996101', '2026-01-01 09:00:00', '2026-01-01 09:00:00')`)
	require.NoError(t, err)
	uid, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = conn.Exec(`INSERT INTO user_profiles (user_id, weight, created_at, updated_at) VALUES (?, 55.00, '2026-02-01 09:00:00', '2026-02-01 09:00:00')`, uid)
	require.NoError(t, err)
	before := scanProfiles(t, conn, "user_profiles", uid)

	_, err = p.UpTo(ctx, userProfilesUniqueVersion)
	require.NoError(t, err)
	var n int
	require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM user_profiles_dedupe_backup`).Scan(&n))
	assert.Zero(t, n, "no backup rows")
	assert.Equal(t, before, scanProfiles(t, conn, "user_profiles", uid))

	_, err = p.DownTo(ctx, userProfilesUniqueVersion-1)
	require.NoError(t, err)
	assert.Equal(t, before, scanProfiles(t, conn, "user_profiles", uid))
	assert.False(t, tableExists(t, conn, "user_profiles_dedupe_backup"))
}
