package events_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/events"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

func seed(t *testing.T, db *sql.DB, goal string) (uid, rule uint64) {
	t.Helper()
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', '09120006201', NOW(), NOW())`)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	res, err = db.Exec(`INSERT INTO message_rules (title, level, kind, priority, placement, content, conditions, exceptions,
		status, is_active, version, goal_event, created_at, updated_at)
		VALUES ('t', 'info', 'clinical', 500, 'smart_tip', '{"a":{"title":"x","body":"y"}}', '[]', '[]', 'approved', 1, 3, ?, NOW(), NOW())`, goal)
	require.NoError(t, err)
	rid, _ := res.LastInsertId()
	return uint64(id), uint64(rid) //nolint:gosec // test ids
}

func count(t *testing.T, db *sql.DB, q string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow(q, args...).Scan(&n))
	return n
}

// The multi-row insert stores every column (NULL version / slot for 0 / "") as the Tehran wall clock.
func TestInsertBatch_Int(t *testing.T) {
	db := testdb.New(t)
	uid, rule := seed(t, db, "period_logged")
	q := store.New(db)
	at := time.Date(2026, 10, 21, 10, 0, 0, 0, civildate.Tehran)
	var evs []events.Event
	for i := range 250 {
		e := events.Event{RuleID: rule, UserID: uid, Version: 3, Variant: "b", Slot: "smart_tip", Event: events.Shown, At: at.Add(time.Duration(i) * time.Second)}
		if i == 0 {
			e.Version, e.Slot = 0, ""
		}
		evs = append(evs, e)
	}
	w := events.NewWriter(q, nil)
	w.Write(context.Background(), evs...)
	w.Close(context.Background())
	assert.Equal(t, 250, count(t, db, `SELECT COUNT(*) FROM message_events WHERE user_id = ?`, uid))
	assert.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM message_events WHERE user_id = ? AND version IS NULL AND slot IS NULL AND created_at = '2026-10-21 10:00:00'`, uid))
	assert.Equal(t, 249, count(t, db, `SELECT COUNT(*) FROM message_events WHERE user_id = ? AND version = 3 AND slot = 'smart_tip' AND variant = 'b'`, uid))
}

// M-17: a goal is matched 6 days 23 h after the last shown, not at 7 days + 1 s, and once per shown window.
func TestGoals_Int(t *testing.T) {
	db := testdb.New(t)
	uid, rule := seed(t, db, "period_logged")
	q := store.New(db)
	shownAt := time.Date(2026, 10, 1, 9, 0, 0, 0, civildate.Tehran)
	_, err := db.Exec(`INSERT INTO message_events (rule_id, user_id, version, variant, slot, event, created_at) VALUES (?, ?, 3, 'a', 'smart_tip', 'impression', ?)`,
		rule, uid, shownAt)
	require.NoError(t, err)
	w := events.NewWriter(q, nil)
	defer w.Close(context.Background())
	g := events.NewGoals(q, w, nil, nil)
	goals := func() int {
		w.Flush(context.Background())
		return count(t, db, `SELECT COUNT(*) FROM message_events WHERE user_id = ? AND rule_id = ? AND event = 'goal' AND version = 3 AND slot = 'smart_tip'`, uid, rule)
	}

	n, err := g.Match(context.Background(), uid, "period_logged", shownAt.Add(7*24*time.Hour+time.Second))
	require.NoError(t, err)
	assert.Zero(t, n, "7 days + 1 s: outside the window")
	n, err = g.Match(context.Background(), uid, "log_saved", shownAt.Add(time.Hour))
	require.NoError(t, err)
	assert.Zero(t, n, "another goal event")
	n, err = g.Match(context.Background(), uid, "period_logged", shownAt.Add(6*24*time.Hour+23*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 1, n, "6 d 23 h: matched")
	assert.Equal(t, 1, goals())
	n, err = g.Match(context.Background(), uid, "period_logged", shownAt.Add(6*24*time.Hour+23*time.Hour+time.Minute))
	require.NoError(t, err)
	assert.Zero(t, n, "once per shown window")

	// A new shown opens a new window.
	_, err = db.Exec(`INSERT INTO message_events (rule_id, user_id, version, variant, slot, event, created_at) VALUES (?, ?, 3, 'a', 'smart_tip', 'impression', ?)`,
		rule, uid, shownAt.Add(7*24*time.Hour))
	require.NoError(t, err)
	n, err = g.Match(context.Background(), uid, "period_logged", shownAt.Add(8*24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, 2, goals())

	// The process hook: installed matcher, request clock; unknown event and user 0 are no-ops.
	events.Install(g)
	events.Goal(context.Background(), 0, "period_logged")
	events.Goal(context.Background(), uid, "bogus")
	assert.Equal(t, 2, goals())
}

// M-13: the rollup is idempotent (run twice → same counts); prune deletes raw rows older than 180 days in batches and
// leaves the rollup — including days it never rolled up, which it backfills first.
func TestRollupAndPrune_Int(t *testing.T) {
	db := testdb.New(t)
	uid, rule := seed(t, db, "period_logged")
	q := store.New(db)
	j := events.NewJobs(q)
	now := time.Date(2026, 10, 21, 3, 0, 0, 0, civildate.Tehran)
	ins := func(at time.Time, event, slot string, n int) {
		for range n {
			_, err := db.Exec(`INSERT INTO message_events (rule_id, user_id, version, variant, slot, event, created_at) VALUES (?, ?, 3, 'a', ?, ?, ?)`,
				rule, uid, slot, event, at)
			require.NoError(t, err)
		}
	}
	yesterday := time.Date(2026, 10, 20, 12, 0, 0, 0, civildate.Tehran)
	ins(yesterday, "impression", "smart_tip", 3)
	ins(yesterday, "click", "smart_tip", 1)
	ins(now, "impression", "smart_tip", 2) // today: not rolled up
	old := now.AddDate(0, 0, -200)
	ins(old, "impression", "smart_tip", 4)

	daily := func(day string) map[string]int {
		rows, err := db.Query(`SELECT event, `+"`count`"+` FROM message_event_daily WHERE rule_id = ? AND `+"`day`"+` = ? AND version = 3 AND slot = 'smart_tip'`, rule, day)
		require.NoError(t, err)
		defer func() { _ = rows.Close() }()
		out := map[string]int{}
		for rows.Next() {
			var e string
			var n int
			require.NoError(t, rows.Scan(&e, &n))
			out[e] = n
		}
		require.NoError(t, rows.Err())
		return out
	}
	require.NoError(t, j.Rollup(context.Background(), now))
	require.NoError(t, j.Rollup(context.Background(), now))
	assert.Equal(t, map[string]int{"impression": 3, "click": 1}, daily("2026-10-20"), "run twice → same counts")
	assert.Empty(t, daily("2026-10-21"))

	n, err := j.Prune(context.Background(), now)
	require.NoError(t, err)
	assert.EqualValues(t, 4, n)
	assert.Equal(t, map[string]int{"impression": 4}, daily(civildate.InTehran(old).String()), "the pruned day was backfilled")
	assert.Equal(t, map[string]int{"impression": 3, "click": 1}, daily("2026-10-20"), "prune leaves the rollup intact")
	assert.Equal(t, 6, count(t, db, `SELECT COUNT(*) FROM message_events WHERE rule_id = ?`, rule))
	n, err = j.Prune(context.Background(), now)
	require.NoError(t, err)
	assert.Zero(t, n, "idempotent")
	assert.Equal(t, map[string]int{"impression": 4}, daily(civildate.InTehran(old).String()))
}
