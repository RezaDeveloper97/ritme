package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const bleedingLogSourceVersion int64 = 104 // 00104_bleeding_log_source.sql

// T-M9-28 (I-02, D-124 / D-138): the old bridge's unconfirmed, non-estimated user_logged rows become confirmed
// bleeding_log rows and their users' revisions are bumped; button rows, estimates and other users are untouched.
// Down reverts by source.
func TestBleedingLogSourceMigration(t *testing.T) {
	ctx := context.Background()
	conn := testdb.New(t)
	p, err := platformdb.NewMigrator(conn)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })
	_, err = p.DownTo(ctx, bleedingLogSourceVersion-1)
	require.NoError(t, err)

	exec := func(q string, args ...any) {
		t.Helper()
		_, err := conn.ExecContext(ctx, q, args...)
		require.NoError(t, err, q)
	}
	period := func(uid uint64, start string, end any, confirmed, estimated int, source string) {
		exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, is_confirmed, is_estimated, source, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid, start, end, confirmed, estimated, source)
	}

	bridged := newV6User(t, conn, "09129990281")
	period(bridged, "2026-07-01", "2026-07-05", 1, 0, "user_logged") // the period button
	period(bridged, "2026-08-01", "2026-08-01", 0, 0, "user_logged") // the old bridge, back-dated
	period(bridged, "2026-09-01", nil, 0, 0, "user_logged")          // the old bridge, newest
	exec(`INSERT INTO user_revisions (user_id, engine_rev, analysis_rev, updated_at) VALUES (?, 4, 2, NOW())`, bridged)

	noRevRow := newV6User(t, conn, "09129990282")
	period(noRevRow, "2026-09-10", nil, 0, 0, "user_logged")

	untouched := newV6User(t, conn, "09129990283")
	period(untouched, "2026-08-10", "2026-08-14", 1, 0, "user_logged")
	period(untouched, "2026-09-07", nil, 0, 1, "onboarding_estimate") // an estimate is no bridge row

	_, err = p.UpTo(ctx, bleedingLogSourceVersion)
	require.NoError(t, err)

	assert.Equal(t, []string{"2026-07-01 2026-07-05 -1 user_logged 1", "2026-08-01 2026-08-01 -1 bleeding_log 1",
		"2026-09-01 - -1 bleeding_log 1"}, historyRows(t, conn, bridged))
	assert.Equal(t, [2]int{5, 3}, engineRev(t, conn, bridged))
	assert.Equal(t, []string{"2026-09-10 - -1 bleeding_log 1"}, historyRows(t, conn, noRevRow))
	assert.Equal(t, [2]int{1, 1}, engineRev(t, conn, noRevRow), "a missing revision row is created")
	assert.Equal(t, []string{"2026-08-10 2026-08-14 -1 user_logged 1", "2026-09-07 - -1 onboarding_estimate 0"}, historyRows(t, conn, untouched))
	assert.Equal(t, [2]int{0, 0}, engineRev(t, conn, untouched), "not affected: no bump")
	var stamp string
	require.NoError(t, conn.QueryRow(`SELECT CAST(updated_at AS CHAR) FROM cycle_histories WHERE user_id = ? AND period_start_date = '2026-09-01'`, bridged).Scan(&stamp))
	assert.Equal(t, "2026-09-01 09:00:00", stamp, "a data conversion, not an edit")

	_, err = p.DownTo(ctx, bleedingLogSourceVersion-1)
	require.NoError(t, err)
	assert.Equal(t, []string{"2026-07-01 2026-07-05 -1 user_logged 1", "2026-08-01 2026-08-01 -1 user_logged 0",
		"2026-09-01 - -1 user_logged 0"}, historyRows(t, conn, bridged))
	assert.Equal(t, []string{"2026-09-10 - -1 user_logged 0"}, historyRows(t, conn, noRevRow))
	assert.Equal(t, [2]int{6, 4}, engineRev(t, conn, bridged), "down bumps the reverted users")
	assert.Equal(t, [2]int{0, 0}, engineRev(t, conn, untouched))

	_, err = p.UpTo(ctx, bleedingLogSourceVersion)
	require.NoError(t, err)
	assert.Equal(t, []string{"2026-09-10 - -1 bleeding_log 1"}, historyRows(t, conn, noRevRow), "up again")
}
