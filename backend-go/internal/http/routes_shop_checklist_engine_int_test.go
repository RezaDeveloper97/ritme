package http

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/revision"
	"github.com/ritme/backend-go/internal/cycle/state"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/shop/checklist"
	shopstore "github.com/ritme/backend-go/internal/shop/store"
)

// T-M9-37 (D-136, I-10): the shop supply reminder on the real v6 engine — a period edit (an engine revision bump)
// moves the stored reminder on the next read; a pregnant user without a stored life mode gets no reminder.
func TestShopSupply_EnginePredictor(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	q := cyclestore.New(db)
	p, err := state.New(q, nil, clock.Real{}, state.DefaultOptions())
	require.NoError(t, err)
	pred := shopEnginePredictor{states: state.Source{P: p, Revs: q}, revs: q}
	svc := checklist.NewService(shopstore.New(db), nil, pred)
	today := civildate.MustParse("2026-10-06")
	now := time.Date(2026, 10, 6, 10, 0, 0, 0, civildate.Tehran)

	user := func(mobile string) uint64 {
		res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', ?, NOW(), NOW())`, mobile)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return uint64(id) //nolint:gosec // test id
	}
	period := func(uid uint64, start string) {
		s := civildate.MustParse(start)
		_, err := db.Exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, bleeding_length, is_confirmed,
			is_estimated, source, created_at, updated_at) VALUES (?, ?, ?, 5, 1, 0, 'user_logged', NOW(), NOW())`, uid, s, s.AddDays(4))
		require.NoError(t, err)
		require.NoError(t, revision.Bump(ctx, db, uid))
	}
	reminder := func(uid uint64) (string, string) {
		var at time.Time
		var meta string
		require.NoError(t, db.QueryRow(`SELECT scheduled_at, meta FROM reminders WHERE user_id = ?`, uid).Scan(&at, &meta))
		return at.Format("2006-01-02 15:04"), meta
	}

	u := user("09120000981")
	period(u, "2026-08-14")
	period(u, "2026-09-11") // 28 days → next 2026-10-09
	st, err := svc.SaveSupply(ctx, u, checklist.Supply{Enabled: true, LeadDays: 3}, today, now, "en")
	require.NoError(t, err)
	assert.Equal(t, "2026-10-06", st.RemindOn.String())
	at, meta := reminder(u)
	assert.Equal(t, "2026-10-06 09:00", at)
	assert.JSONEq(t, `{"source":"shop_supply","engine_rev":2}`, meta)

	// A period edit (the 09-11 period was really on 09-15) bumps the revision: the next read moves the stored reminder
	// past today and records the new revision.
	_, err = db.Exec(`UPDATE cycle_histories SET period_start_date = '2026-09-15', period_end_date = '2026-09-19' WHERE user_id = ? AND period_start_date = '2026-09-11'`, u)
	require.NoError(t, err)
	require.NoError(t, revision.Bump(ctx, db, u))
	st, err = svc.SyncSupply(ctx, u, today, now, "en")
	require.NoError(t, err)
	assert.True(t, st.RemindOn.After(today), st.RemindOn.String())
	at2, meta := reminder(u)
	assert.NotEqual(t, at, at2)
	assert.JSONEq(t, `{"source":"shop_supply","engine_rev":3}`, meta)

	// Pregnant (active pregnancy profile) without a stored life mode: the engine's resolved mode is pregnancy.
	preg := user("09120000982")
	period(preg, "2026-08-14")
	period(preg, "2026-09-11")
	_, err = db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, created_at, updated_at) VALUES (?, 1, NOW(), NOW())`, preg)
	require.NoError(t, err)
	require.NoError(t, revision.Bump(ctx, db, preg))
	pr, err := pred.NextPeriodStart(ctx, preg, today)
	require.NoError(t, err)
	assert.False(t, pr.Known)
	st, err = svc.SaveSupply(ctx, preg, checklist.Supply{Enabled: true, LeadDays: 3}, today, now, "en")
	require.NoError(t, err)
	assert.True(t, st.RemindOn.IsZero())
	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM reminders WHERE user_id = ?`, preg).Scan(&n))
	assert.Zero(t, n)
}
