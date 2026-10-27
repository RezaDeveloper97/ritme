package messages_test

// T-M9-38 (D-130): /messages/daily and the v2 facts (selected, admin preview) read the v6 engine state when an engine
// source is attached (CYCLE_ENGINE=shadow|v6), over a real database.

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/state"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/messages"
	"github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/messages/manager"
	mstore "github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

func engineSource(t *testing.T, db *sql.DB) state.UserStates {
	t.Helper()
	q := cyclestore.New(db)
	p, err := state.New(q, nil, clock.Real{}, state.DefaultOptions())
	require.NoError(t, err)
	return state.Source{P: p, Revs: q}
}

// seedCycleUser inserts a user with goal, an optional life mode and six closed 5-day periods every 28 days, the last
// starting on last.
func seedCycleUser(t *testing.T, db *sql.DB, mobile, goal, lifeMode string, last civildate.Date) uint64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara Test', ?, NOW(), NOW())`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_profiles (user_id, birthday, last_period_start, cycle_duration, period_duration, user_goal, created_at, updated_at)
		VALUES (?, '1998-01-01', ?, 28, 5, ?, NOW(), NOW())`, id, last.AddDays(-140).String(), goal)
	require.NoError(t, err)
	if lifeMode != "" {
		_, err = db.Exec(`INSERT INTO user_life_profiles (user_id, life_mode, created_at, updated_at) VALUES (?, ?, NOW(), NOW())`, id, lifeMode)
		require.NoError(t, err)
	}
	for i := 5; i >= 0; i-- {
		s := last.AddDays(-28 * i)
		_, err = db.Exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, bleeding_length, is_confirmed,
			is_estimated, source, created_at, updated_at) VALUES (?, ?, ?, 5, 1, 0, 'user_logged', NOW(), NOW())`,
			id, s.String(), s.AddDays(4).String())
		require.NoError(t, err)
	}
	return uint64(id) //nolint:gosec // test id
}

// Late 8 days: the facts and the daily message count the cycle on (day 37), never "day N of a new cycle".
func TestV6Int_LateUser(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	today := civildate.MustParse("2026-10-21")
	uid := seedCycleUser(t, db, "09120006381", "non_ttc", "", today.AddDays(-36))
	states := engineSource(t, db)

	f, err := messages.BuildFacts(ctx, messages.NewHandlers(db, clock.Real{}).WithEngine(states), uid, today)
	require.NoError(t, err)
	assert.Equal(t, 37.0, f["cycle_day"])
	assert.Equal(t, 8.0, f["days_late"])
	assert.Equal(t, 0.0, f["days_to_period"])
	assert.Equal(t, "period_expected", f["main_phase"])
	assert.Equal(t, "luteal", f["phase"])
	assert.Equal(t, false, f["is_fertile_window"])
	assert.Contains(t, f, "data_quality")
	assert.Contains(t, f, "confidence")
	assert.Contains(t, f, "suggest_pregnancy_test")

	// v1 (no engine) still wraps — the legacy path is unchanged.
	v1, err := messages.BuildFacts(ctx, messages.NewHandlers(db, clock.Real{}), uid, today)
	require.NoError(t, err)
	assert.NotContains(t, v1, "main_phase")

	src := messages.NewStoreSource(mstore.New(db), pstore.New(db), uid, today).WithEngine(states)
	res, err := manager.New(src, content.NewRepository(mstore.New(db)), "fa", today).Generate(ctx, today, "")
	require.NoError(t, err)
	ci, _ := res.ContextInfo.Get("cycle_day")
	assert.Equal(t, 37, ci)
	phase, _ := res.ContextInfo.Get("phase")
	assert.Equal(t, "luteal", phase)
}

// A teen with goal TTC on her fertile days gets no fertile facts and no TTC targeting.
func TestV6Int_TeenTTCHasNoFertileFacts(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	today := civildate.MustParse("2026-10-21")
	uid := seedCycleUser(t, db, "09120006382", "ttc", "teen", today.AddDays(-12)) // O−1 of a 28-day cycle
	h := messages.NewHandlers(db, clock.Real{}).WithEngine(engineSource(t, db))
	for i := -3; i <= 3; i++ {
		f, err := messages.BuildFacts(ctx, h, uid, today.AddDays(i))
		require.NoError(t, err)
		msg := today.AddDays(i).String()
		assert.NotContains(t, f, "is_fertile_window", msg)
		assert.Equal(t, true, f["fertility_suppressed"], msg)
		assert.Equal(t, "non_ttc", f["user_goal"], msg)
		assert.NotEqual(t, "ovulation", f["phase"], msg)
		assert.NotEqual(t, "fertile", f["main_phase"], msg)
		assert.Equal(t, false, f["suggest_pregnancy_test"], msg)
	}
}
