package healthlog_test

// Prediction log from the bleeding-log bridge (T-M9-39b, D-137): a flow day that starts a new real period queues the
// same prediction-log job as the period button, after the commit, with ids and revisions only; a flow day inside an
// existing period, a log without flow and a removal write nothing.

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/periods"
	"github.com/ritme/backend-go/internal/cycle/snapshots"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/queue"
)

type startEvents struct {
	mu  sync.Mutex
	evs []periods.StartEvent
}

func (s *startEvents) list() []periods.StartEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]periods.StartEvent(nil), s.evs...)
}

// predictionLogs is the logs env with the snapshots prediction job on a sync queue (Dispatch runs it inline) behind
// the healthlog start hook; every event is recorded too.
func predictionLogs(t *testing.T) (*logsEnv, *startEvents) {
	t.Helper()
	rec := &startEvents{}
	e := setupLogsWith(t, func(db *sql.DB) periods.StartHook {
		q := queue.New(nil, queue.Options{Mode: queue.ModeSync})
		now := func() time.Time { return time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran) }
		require.NoError(t, snapshots.New(db, nil).RegisterJobs(q, now))
		inner := snapshots.Hook(q, nil)
		return func(ctx context.Context, ev periods.StartEvent) {
			// After commit: the period row and the bumped revision are visible on another connection.
			var rev uint64
			require.NoError(t, db.QueryRowContext(ctx, `SELECT engine_rev FROM user_revisions WHERE user_id = ?`, ev.UserID).Scan(&rev))
			assert.Equal(t, rev, ev.RevAfter, "rev_after is the committed engine_rev")
			var n int
			require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cycle_histories WHERE id = ? AND user_id = ?`, ev.PeriodID, ev.UserID).Scan(&n))
			assert.Equal(t, 1, n, "the new period is committed")
			rec.mu.Lock()
			rec.evs = append(rec.evs, ev)
			rec.mu.Unlock()
			inner(ctx, ev)
		}
	})
	return e, rec
}

func (e *logsEnv) predictionLog(t *testing.T, uid uint64) []string {
	t.Helper()
	rows, err := e.db.Query(`SELECT CONCAT(kind, ':', DATE_FORMAT(cycle_start, '%Y-%m-%d'), ':', DATE_FORMAT(predicted_next, '%Y-%m-%d'), ':',
		IFNULL(DATE_FORMAT(actual_start, '%Y-%m-%d'), '-'), ':', IFNULL(error_days, '-'))
		FROM engine_prediction_log WHERE user_id = ? ORDER BY cycle_start, kind`, uid)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	out := []string{}
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		out = append(out, s)
	}
	require.NoError(t, rows.Err())
	return out
}

func (e *logsEnv) postLog(t *testing.T, tok, body string) {
	t.Helper()
	r := e.do(t, "POST", "/api/v1/health-logs", tok, "fa", body)
	require.Contains(t, []int{200, 201}, r.status, r.raw)
}

func flow(date string) string { return `{"log_date":"` + date + `","bleeding_intensity":"medium"}` }

func TestBridgePredictionLog_NewStartsOnly(t *testing.T) {
	e, rec := predictionLogs(t)
	uid, tok := e.user(t, "09121100001", "")

	// First bleeding day: a new real period → `first` of its cycle (28-day default → 08-29).
	e.postLog(t, tok, flow("2026-08-01"))
	assert.Equal(t, []string{"first:2026-08-01:2026-08-29:-:-"}, e.predictionLog(t, uid))
	require.Len(t, rec.list(), 1)

	// Continuation days, a day inside the open period after a gap and a log without flow: no new start, nothing.
	e.postLog(t, tok, flow("2026-08-02"))
	e.postLog(t, tok, flow("2026-08-03"))
	e.postLog(t, tok, flow("2026-08-05"))
	e.postLog(t, tok, `{"log_date":"2026-08-10","moods":["sad"]}`)
	assert.Equal(t, []string{"first:2026-08-01:2026-08-29:-:-"}, e.predictionLog(t, uid))
	require.Len(t, rec.list(), 1)

	// The next bleeding start 09-02: `last` of the 08-01 cycle, actual 09-02 and error 4 on both its rows, then `first`
	// of the new cycle.
	e.postLog(t, tok, flow("2026-09-02"))
	got := e.predictionLog(t, uid)
	require.Len(t, got, 3, got)
	assert.Equal(t, []string{
		"first:2026-08-01:2026-08-29:2026-09-02:4",
		"last:2026-08-01:2026-08-29:2026-09-02:4",
	}, got[:2])
	assert.Regexp(t, `^first:2026-09-02:\d{4}-\d{2}-\d{2}:-:-$`, got[2])

	evs := rec.list()
	require.Len(t, evs, 2)
	for _, ev := range evs {
		assert.Equal(t, uid, ev.UserID)
		assert.Equal(t, ev.RevAfter-1, ev.RevBefore)
	}
	assert.Less(t, evs[0].RevAfter, evs[1].RevAfter)
	assert.NotEqual(t, evs[0].PeriodID, evs[1].PeriodID)

	// A flow day inside the new period and a removal (rebuild) are no new start.
	e.postLog(t, tok, flow("2026-09-03"))
	e.deleteDay(t, tok, "2026-09-03")
	assert.Equal(t, got, e.predictionLog(t, uid))
	assert.Len(t, rec.list(), 2)
}
