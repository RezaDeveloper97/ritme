package home

// T-M9-34b: the v6 home's merged reads (challenge completions, doctor + medication reminders) return exactly what the
// v1 single-purpose queries return.

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/home/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMergedReads_MatchV1Queries(t *testing.T) {
	db := testdb.New(t)
	testdb.LoadSQL(t, filepath.Join("..", "..", "contract", "fixtures", "dump.sql"))
	ctx := context.Background()
	q := store.New(db)
	const uid = 1001
	day := civildate.MustParse("2026-09-23")

	exec := func(query string, args ...any) {
		t.Helper()
		_, err := db.ExecContext(ctx, query, args...)
		require.NoError(t, err)
	}
	exec("DELETE FROM reminders WHERE user_id = ?", uid)
	exec("DELETE FROM user_challenge_completions WHERE user_id = ?", uid)

	// Reminders that compete on each type's order: NULL scheduled_at sorts first for doctor; medication orders on
	// COALESCE(recurrence_time, TIME(scheduled_at)); inactive, expired and other-type rows never count.
	ins := func(typ, recurrence string, scheduled, recTime, starts, ends any, active int) {
		exec(`INSERT INTO reminders (user_id, type, title, scheduled_at, recurrence, recurrence_time, starts_on, ends_on, is_active)
			VALUES (?, ?, 't', ?, ?, ?, ?, ?, ?)`, uid, typ, scheduled, recurrence, recTime, starts, ends, active)
	}
	ins("doctor", "none", "2026-09-25 10:00:00", nil, nil, nil, 1)
	ins("doctor", "weekly", nil, "09:00:00", "2026-09-01", nil, 1)
	ins("doctor", "none", "2026-09-22 10:00:00", nil, nil, nil, 1) // past one-off
	ins("doctor", "none", "2026-09-23 08:00:00", nil, nil, nil, 0) // inactive
	ins("medication", "daily", "2026-09-01 22:00:00", "20:30:00", nil, nil, 1)
	ins("medication", "daily", "2026-09-01 07:15:00", nil, nil, nil, 1)
	ins("medication", "daily", nil, "06:00:00", nil, "2026-09-20", 1) // ended
	ins("custom", "daily", nil, "05:00:00", nil, nil, 1)

	hc := newContext(ctx, &Deps{Queries: q})
	hc.UserID, hc.Date = uid, day
	compare := func() {
		t.Helper()
		hc.v6, hc.remindersLoaded, hc.doctorRem, hc.medicationRem = nil, false, nil, nil
		d1, dErr1 := hc.doctorReminder()
		m1, mErr1 := hc.medicationReminder()
		hc.v6 = &engineView{}
		d6, dErr6 := hc.doctorReminder()
		m6, mErr6 := hc.medicationReminder()
		assert.Equal(t, errors.Is(dErr1, sql.ErrNoRows), errors.Is(dErr6, sql.ErrNoRows))
		assert.Equal(t, errors.Is(mErr1, sql.ErrNoRows), errors.Is(mErr6, sql.ErrNoRows))
		assert.Equal(t, d1, d6)
		assert.Equal(t, m1, m6)
	}
	compare()
	d, err := hc.doctorReminder()
	require.NoError(t, err)
	assert.False(t, d.ScheduledAt.Valid, "the recurring doctor reminder without scheduled_at sorts first")
	m, err := hc.medicationReminder()
	require.NoError(t, err)
	assert.Equal(t, "07:15", m.ScheduledAt.Time.In(civildate.Tehran).Format("15:04"))

	exec("DELETE FROM reminders WHERE user_id = ?", uid) // none of either type
	compare()

	// Challenge completions: window (day − 14 … day − 1) and the day itself from one read.
	var ids []uint64
	rows, err := db.QueryContext(ctx, "SELECT id FROM challenges ORDER BY id LIMIT 3")
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var id uint64
		require.NoError(t, rows.Scan(&id))
		ids = append(ids, id)
	}
	require.NoError(t, rows.Err())
	require.Len(t, ids, 3, "the fixture has challenges")
	insC := func(id uint64, d civildate.Date) {
		exec("INSERT INTO user_challenge_completions (user_id, challenge_id, completion_date) VALUES (?, ?, ?)", uid, id, d)
	}
	insC(ids[0], day.AddDays(-15)) // outside the window
	insC(ids[0], day.AddDays(-14))
	insC(ids[1], day.AddDays(-1))
	insC(ids[2], day)
	insC(ids[1], day.AddDays(1)) // future

	recent1, today1, err := challengeCompletions(ctx, q, uid, day, false)
	require.NoError(t, err)
	recent6, today6, err := challengeCompletions(ctx, q, uid, day, true)
	require.NoError(t, err)
	assert.ElementsMatch(t, recent1, recent6)
	assert.Nil(t, today1)
	assert.Equal(t, []uint64{ids[2]}, today6)
	for _, id := range ids {
		done, err := q.ChallengeCompletedOn(ctx, store.ChallengeCompletedOnParams{UserID: uid, ChallengeID: id, CompletionDate: day})
		require.NoError(t, err)
		assert.Equal(t, done, id == ids[2])
	}
}
