package migrations_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const onboardingAnchorVersion int64 = 103 // 00103_onboarding_declared_anchor.sql

// historyRows is a user's cycle_histories as "start end bleeding source confirmed" lines, oldest first.
func historyRows(t *testing.T, conn *sql.DB, uid uint64) []string {
	t.Helper()
	rows, err := conn.Query(`SELECT DATE_FORMAT(period_start_date, '%Y-%m-%d'), COALESCE(DATE_FORMAT(period_end_date, '%Y-%m-%d'), '-'),
		COALESCE(bleeding_length, -1), source, is_confirmed FROM cycle_histories WHERE user_id = ? ORDER BY period_start_date`, uid)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var start, end, source string
		var bleeding, confirmed int
		require.NoError(t, rows.Scan(&start, &end, &bleeding, &source, &confirmed))
		out = append(out, fmt.Sprintf("%s %s %d %s %d", start, end, bleeding, source, confirmed))
	}
	require.NoError(t, rows.Err())
	return out
}

func declaredAnchor(t *testing.T, conn *sql.DB, uid uint64) string {
	t.Helper()
	var d sql.NullString
	require.NoError(t, conn.QueryRow(`SELECT DATE_FORMAT(onboarding_last_period_start, '%Y-%m-%d') FROM user_profiles WHERE user_id = ?`, uid).Scan(&d))
	return d.String
}

func engineRev(t *testing.T, conn *sql.DB, uid uint64) [2]int {
	t.Helper()
	var r [2]int
	err := conn.QueryRow(`SELECT engine_rev, analysis_rev FROM user_revisions WHERE user_id = ?`, uid).Scan(&r[0], &r[1])
	if errors.Is(err, sql.ErrNoRows) {
		return r
	}
	require.NoError(t, err)
	return r
}

// T-M9-27 (I-01, D-123): the onboarding seed rows leave cycle_histories. Personas: an untouched seed (past and still
// running), a seed the user edited, a seed followed by later logged periods, a user without a seed. Up → the untouched
// seeds move to onboarding_last_period_start, the edited one becomes user_logged, revisions are bumped; down → the
// seeds are back (the converted row stays user_logged).
func TestOnboardingDeclaredAnchorMigration(t *testing.T) {
	ctx := context.Background()
	conn := testdb.New(t)
	p, err := platformdb.NewMigrator(conn)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })
	_, err = p.DownTo(ctx, onboardingAnchorVersion-1)
	require.NoError(t, err)

	exec := func(q string, args ...any) {
		t.Helper()
		_, err := conn.ExecContext(ctx, q, args...)
		require.NoError(t, err, q)
	}
	profile := func(uid uint64, lmp string) {
		exec(`INSERT INTO user_profiles (user_id, period_duration, cycle_duration, last_period_start, created_at, updated_at)
			VALUES (?, 5, 28, ?, NOW(), NOW())`, uid, lmp)
	}
	period := func(uid uint64, start string, end, bleeding any, source string) {
		exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, bleeding_length, is_confirmed, is_estimated, source, created_at, updated_at)
			VALUES (?, ?, ?, ?, 1, 0, ?, NOW(), NOW())`, uid, start, end, bleeding, source)
	}
	today := civildate.InTehran(time.Now())
	running := today.AddDays(-1).String() // start + 4 > today: the old re-sync left end and length NULL

	untouched := newV6User(t, conn, "09129990271")
	profile(untouched, "2026-09-10")
	period(untouched, "2026-09-10", "2026-09-14", 5, "user_profile_confirmed")
	exec(`INSERT INTO user_revisions (user_id, engine_rev, analysis_rev, updated_at) VALUES (?, 3, 2, NOW())`, untouched)

	ongoing := newV6User(t, conn, "09129990272")
	profile(ongoing, running)
	period(ongoing, running, nil, nil, "user_profile_confirmed")

	edited := newV6User(t, conn, "09129990273")
	profile(edited, "2026-09-10")
	period(edited, "2026-09-10", "2026-09-16", 7, "user_profile_confirmed") // the user moved the end

	later := newV6User(t, conn, "09129990274")
	profile(later, "2026-10-05") // logging a period moved last_period_start to the latest actual start
	period(later, "2026-08-01", "2026-08-05", 5, "user_profile_confirmed")
	period(later, "2026-09-02", "2026-09-06", 5, "user_logged")
	period(later, "2026-10-05", nil, nil, "user_logged")

	noSeed := newV6User(t, conn, "09129990275")
	profile(noSeed, "2026-09-01")
	period(noSeed, "2026-09-01", "2026-09-05", 5, "user_logged")

	_, err = p.UpTo(ctx, onboardingAnchorVersion)
	require.NoError(t, err)

	assert.Empty(t, historyRows(t, conn, untouched))
	assert.Equal(t, "2026-09-10", declaredAnchor(t, conn, untouched))
	assert.Equal(t, [2]int{4, 3}, engineRev(t, conn, untouched))

	assert.Empty(t, historyRows(t, conn, ongoing))
	assert.Equal(t, running, declaredAnchor(t, conn, ongoing))
	assert.Equal(t, [2]int{1, 1}, engineRev(t, conn, ongoing))

	assert.Equal(t, []string{"2026-09-10 2026-09-16 7 user_logged 1"}, historyRows(t, conn, edited))
	assert.Empty(t, declaredAnchor(t, conn, edited))
	assert.Equal(t, [2]int{1, 1}, engineRev(t, conn, edited))

	assert.Equal(t, []string{"2026-09-02 2026-09-06 5 user_logged 1", "2026-10-05 - -1 user_logged 1"}, historyRows(t, conn, later))
	assert.Equal(t, "2026-08-01", declaredAnchor(t, conn, later))
	assert.Equal(t, [2]int{1, 1}, engineRev(t, conn, later))

	assert.Equal(t, []string{"2026-09-01 2026-09-05 5 user_logged 1"}, historyRows(t, conn, noSeed))
	assert.Empty(t, declaredAnchor(t, conn, noSeed))
	assert.Equal(t, [2]int{0, 0}, engineRev(t, conn, noSeed), "not affected: no bump")

	var seeds int
	require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM cycle_histories WHERE source = 'user_profile_confirmed'`).Scan(&seeds))
	assert.Zero(t, seeds)

	// Down: the seeds come back from the column with the old re-sync rule; the converted row stays user_logged.
	_, err = p.DownTo(ctx, onboardingAnchorVersion-1)
	require.NoError(t, err)

	assert.Equal(t, []string{"2026-09-10 2026-09-14 5 user_profile_confirmed 1"}, historyRows(t, conn, untouched))
	assert.Equal(t, []string{running + " - -1 user_profile_confirmed 1"}, historyRows(t, conn, ongoing))
	assert.Equal(t, []string{"2026-09-10 2026-09-16 7 user_logged 1"}, historyRows(t, conn, edited))
	assert.Equal(t, []string{"2026-08-01 2026-08-05 5 user_profile_confirmed 1", "2026-09-02 2026-09-06 5 user_logged 1",
		"2026-10-05 - -1 user_logged 1"}, historyRows(t, conn, later))
	assert.Equal(t, []string{"2026-09-01 2026-09-05 5 user_logged 1"}, historyRows(t, conn, noSeed))
	for _, uid := range []uint64{untouched, ongoing, edited, later, noSeed} {
		assert.Empty(t, declaredAnchor(t, conn, uid), "down: the column is NULL again (nothing wrote it before 00103)")
	}
	assert.Equal(t, [2]int{5, 4}, engineRev(t, conn, untouched), "down bumps the re-seeded users")
	assert.Equal(t, [2]int{1, 1}, engineRev(t, conn, edited))

	// Up again is the same result (the re-created seeds are untouched by definition).
	_, err = p.UpTo(ctx, onboardingAnchorVersion)
	require.NoError(t, err)
	assert.Empty(t, historyRows(t, conn, untouched))
	assert.Equal(t, "2026-09-10", declaredAnchor(t, conn, untouched))
	assert.Equal(t, "2026-08-01", declaredAnchor(t, conn, later))
	assert.Equal(t, []string{"2026-09-10 2026-09-16 7 user_logged 1"}, historyRows(t, conn, edited))
}
