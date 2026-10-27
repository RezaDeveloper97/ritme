package snapshots

import (
	"context"
	"database/sql"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/periods"
	"github.com/ritme/backend-go/internal/cycle/skips"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/queue"
)

func TestMain(m *testing.M) { testdb.Main(m) }

type env struct {
	t   *testing.T
	ctx context.Context
	db  *sql.DB
}

func newEnv(t *testing.T) *env {
	return &env{t: t, ctx: context.Background(), db: testdb.New(t)}
}

func (e *env) exec(q string, args ...any) {
	e.t.Helper()
	_, err := e.db.ExecContext(e.ctx, q, args...)
	require.NoError(e.t, err)
}

func (e *env) user(mobile string) uint64 {
	e.t.Helper()
	res, err := e.db.ExecContext(e.ctx, `INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('U', ?, NOW(), NOW())`, mobile)
	require.NoError(e.t, err)
	id, err := res.LastInsertId()
	require.NoError(e.t, err)
	return uint64(id)
}

func (e *env) count(q string, args ...any) int {
	e.t.Helper()
	var n int
	require.NoError(e.t, e.db.QueryRowContext(e.ctx, q, args...).Scan(&n))
	return n
}

func (e *env) strings(q string, args ...any) []string {
	e.t.Helper()
	rows, err := e.db.QueryContext(e.ctx, q, args...)
	require.NoError(e.t, err)
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var s string
		require.NoError(e.t, rows.Scan(&s))
		out = append(out, s)
	}
	require.NoError(e.t, rows.Err())
	return out
}

func tehran(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, civildate.Tehran)
	if err != nil {
		panic(err)
	}
	return t
}

// 3 active users (a session, a log in Berlin, an analytics event) and 1 inactive → 3 rows; a rerun upserts the same
// 3 rows; the Berlin user gets her own (earlier) date; engine_rev and codes are stored.
func TestNightly_ActiveUsersOnTheirLocalDay(t *testing.T) {
	e := newEnv(t)
	now := tehran("2026-10-28 01:00") // Berlin (CET): 2026-10-27 22:30
	a, b, c, d := e.user("09120000001"), e.user("09120000002"), e.user("09120000003"), e.user("09120000004")

	e.exec(`INSERT INTO oauth_access_tokens (id, user_id, client_id, revoked, created_at, updated_at) VALUES
		('tok-a', ?, 'c', 0, '2026-10-20 10:00:00', '2026-10-20 10:00:00'),
		('tok-d', ?, 'c', 0, '2026-08-01 10:00:00', '2026-08-01 10:00:00')`, a, d)
	e.exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, is_confirmed, is_estimated, source, created_at, updated_at)
		VALUES (?, '2026-10-01', '2026-10-05', 1, 0, 'user_logged', '2026-08-01 10:00:00', '2026-08-01 10:00:00')`, a)
	e.exec(`INSERT INTO user_revisions (user_id, engine_rev, analysis_rev, updated_at) VALUES (?, 7, 7, NOW())`, a)
	e.exec(`INSERT INTO user_profiles (user_id, timezone, created_at, updated_at) VALUES (?, 'Europe/Berlin', NOW(), NOW())`, b)
	e.exec(`INSERT INTO daily_health_logs (user_id, log_date, created_at, updated_at) VALUES (?, '2026-10-25', NOW(), NOW())`, b)
	e.exec(`INSERT INTO analytics_events (user_id, name, occurred_on, occurred_at, created_at) VALUES (?, 'app_open', '2026-10-27', '2026-10-27 09:00:00', NOW())`, c)

	svc := New(e.db, nil)
	q := queue.New(nil, queue.Options{Mode: queue.ModeSync})
	require.NoError(t, svc.RegisterJobs(q, func() time.Time { return now }))

	for range 2 {
		require.NoError(t, q.Dispatch(e.ctx, JobNightly, struct{}{}))
		assert.Equal(t, 3, e.count(`SELECT COUNT(*) FROM engine_daily_snapshots`))
	}
	got := e.strings(`SELECT CONCAT(user_id, ':', DATE_FORMAT(day, '%Y-%m-%d'), ':', engine_rev) FROM engine_daily_snapshots ORDER BY user_id`)
	assert.Equal(t, []string{
		itoa(a) + ":2026-10-28:7",
		itoa(b) + ":2026-10-27:0",
		itoa(c) + ":2026-10-28:0",
	}, got)
	assert.Zero(t, e.count(`SELECT COUNT(*) FROM engine_daily_snapshots WHERE user_id = ?`, d), "inactive for > 30 days")
	// User a: one real period 10-01 → day 28 of a 28-day default cycle, resolved with codes only.
	assert.Equal(t, []string{"cycle:good"}, e.strings(
		`SELECT CONCAT(mode, ':', IF(main_phase <> '' AND subphase <> '' AND confidence <> '', 'good', 'empty')) FROM engine_daily_snapshots WHERE user_id = ?`, a))
}

func itoa(v uint64) string { return strconv.FormatUint(v, 10) }

func TestPurge_KeepsFourHundredDays(t *testing.T) {
	e := newEnv(t)
	u := e.user("09120000011")
	e.exec(`INSERT INTO engine_daily_snapshots (user_id, day, main_phase, subphase, confidence, data_quality, mode) VALUES
		(?, '2025-09-01', 'unknown', 'unknown', 'unknown', 'insufficient', 'cycle'),
		(?, '2025-10-01', 'unknown', 'unknown', 'unknown', 'insufficient', 'cycle')`, u, u)
	svc := New(e.db, nil)
	for range 2 {
		require.NoError(t, svc.Purge(e.ctx, tehran("2026-10-28 03:50")))
		assert.Equal(t, []string{"2025-10-01"}, e.strings(`SELECT DATE_FORMAT(day, '%Y-%m-%d') FROM engine_daily_snapshots`))
	}
}

type predEnv struct {
	*env
	periods *periods.Service
}

func newPredEnv(t *testing.T) *predEnv {
	e := newEnv(t)
	svc := New(e.db, nil)
	q := queue.New(nil, queue.Options{Mode: queue.ModeSync}) // Dispatch runs the job inline
	require.NoError(t, svc.RegisterJobs(q, func() time.Time { return tehran("2026-08-02 12:00") }))
	return &predEnv{env: e, periods: periods.NewService(e.db).WithStartHook(Hook(q, nil))}
}

func (e *predEnv) log(uid uint64) []string {
	return e.strings(`SELECT CONCAT(kind, ':', DATE_FORMAT(cycle_start, '%Y-%m-%d'), ':', DATE_FORMAT(predicted_next, '%Y-%m-%d'), ':',
		IFNULL(DATE_FORMAT(actual_start, '%Y-%m-%d'), '-'), ':', IFNULL(error_days, '-'))
		FROM engine_prediction_log WHERE user_id = ? ORDER BY cycle_start, kind`, uid)
}

func ptr(s string) *civildate.Date {
	d := civildate.MustParse(s)
	return &d
}

// Start 07-01 → first 07-29; real start 08-02 → last of 07-01 (07-29), actual 08-02 and error 4 on both rows of the
// cycle, first of 08-02. Editing the old period and backfilling an older one write nothing.
func TestPredictionLog_StartsEditsAndBackfill(t *testing.T) {
	e := newPredEnv(t)
	u := e.user("09120000021")

	_, err := e.periods.Store(e.ctx, u, civildate.MustParse("2026-07-01"), ptr("2026-07-05"), tehran("2026-07-05 10:00"))
	require.NoError(t, err)
	assert.Equal(t, []string{"first:2026-07-01:2026-07-29:-:-"}, e.log(u))

	_, err = e.periods.Start(e.ctx, u, civildate.MustParse("2026-08-02"), tehran("2026-08-02 10:00"))
	require.NoError(t, err)
	want := []string{
		"first:2026-07-01:2026-07-29:2026-08-02:4",
		"last:2026-07-01:2026-07-29:2026-08-02:4",
	}
	got := e.log(u)
	require.Len(t, got, 3)
	assert.Equal(t, want, got[:2])
	assert.Regexp(t, `^first:2026-08-02:\d{4}-\d{2}-\d{2}:-:-$`, got[2])

	// An edit of the old period (Update is not a new start) and a backfilled older period (not the newest) add nothing.
	var oldID uint64
	require.NoError(t, e.db.QueryRowContext(e.ctx, `SELECT id FROM cycle_histories WHERE user_id = ? AND period_start_date = '2026-07-01'`, u).Scan(&oldID))
	_, err = e.periods.Update(e.ctx, u, oldID, civildate.MustParse("2026-06-30"), ptr("2026-07-04"), tehran("2026-08-03 10:00"))
	require.NoError(t, err)
	_, err = e.periods.Store(e.ctx, u, civildate.MustParse("2026-05-01"), ptr("2026-05-05"), tehran("2026-08-03 10:00"))
	require.NoError(t, err)
	assert.Equal(t, got, e.log(u))
	assert.Zero(t, e.count(`SELECT COUNT(*) FROM engine_prediction_log WHERE cycle_start IN ('2026-06-30', '2026-05-01')`))
}

// A skipped predicted start is not an actual start: the cycle keeps actual_start NULL; rerunning the job for the
// same write changes nothing.
func TestPredictionLog_SkipIsNoActual(t *testing.T) {
	e := newPredEnv(t)
	u := e.user("09120000031")
	_, err := e.periods.Store(e.ctx, u, civildate.MustParse("2026-07-01"), ptr("2026-07-05"), tehran("2026-07-05 10:00"))
	require.NoError(t, err)

	created, err := skips.NewService(e.db).Record(e.ctx, u, civildate.MustParse("2026-07-29"), civildate.MustParse("2026-07-31"))
	require.NoError(t, err)
	require.True(t, created)
	assert.Equal(t, []string{"first:2026-07-01:2026-07-29:-:-"}, e.log(u))

	var id uint64
	require.NoError(t, e.db.QueryRowContext(e.ctx, `SELECT id FROM cycle_histories WHERE user_id = ?`, u).Scan(&id))
	svc := New(e.db, nil)
	require.NoError(t, svc.RecordPrediction(e.ctx, periods.StartEvent{UserID: u, PeriodID: id, RevBefore: 1, RevAfter: 2}, tehran("2026-08-01 10:00")))
	require.NoError(t, svc.RecordPrediction(e.ctx, periods.StartEvent{UserID: u, PeriodID: 999999}, tehran("2026-08-01 10:00")), "a deleted period is a no-op")
	assert.Equal(t, []string{"first:2026-07-01:2026-07-29:-:-"}, e.log(u))
}

// Deleting the user removes her rows (account deletion).
func TestTables_CascadeWithUser(t *testing.T) {
	e := newPredEnv(t)
	u := e.user("09120000041")
	_, err := e.periods.Store(e.ctx, u, civildate.MustParse("2026-07-01"), ptr("2026-07-05"), tehran("2026-07-05 10:00"))
	require.NoError(t, err)
	e.exec(`INSERT INTO engine_daily_snapshots (user_id, day, main_phase, subphase, confidence, data_quality, mode) VALUES (?, '2026-07-05', 'menstrual', 'menstrual', 'low', 'insufficient', 'cycle')`, u)
	e.exec(`DELETE FROM users WHERE id = ?`, u)
	assert.Zero(t, e.count(`SELECT COUNT(*) FROM engine_prediction_log`))
	assert.Zero(t, e.count(`SELECT COUNT(*) FROM engine_daily_snapshots`))
}
