package healthlog_test

// B-N3-14b (N3 stage smoke B-1, deviations.md D-55): a bleeding day logged for a past date joins the period
// it falls in (or sits right next to) instead of starting a new open period with a negative cycle_length, and the
// profile LMP never moves back to a back-dated day. Engine v6 (T-M9-28, I-02, D-124): bridge rows are confirmed
// bleeding_log rows, and a period-button row is never edited by the bridge.

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type periodRow struct {
	start, end  string
	cycleLength sql.NullInt64
	confirmed   bool
}

func (e *logsEnv) periods(t *testing.T, userID uint64) []periodRow {
	t.Helper()
	rows, err := e.db.Query(`SELECT CAST(period_start_date AS CHAR), COALESCE(CAST(period_end_date AS CHAR), ''), cycle_length, is_confirmed
		FROM cycle_histories WHERE user_id = ? ORDER BY period_start_date`, userID)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var out []periodRow
	for rows.Next() {
		var r periodRow
		require.NoError(t, rows.Scan(&r.start, &r.end, &r.cycleLength, &r.confirmed))
		out = append(out, r)
	}
	require.NoError(t, rows.Err())
	return out
}

func (e *logsEnv) lmp(t *testing.T, userID uint64) string {
	t.Helper()
	var v sql.NullString
	require.NoError(t, e.db.QueryRow(`SELECT CAST(last_period_start AS CHAR) FROM user_profiles WHERE user_id = ?`, userID).Scan(&v))
	return v.String
}

func (e *logsEnv) withProfile(t *testing.T, userID uint64) {
	t.Helper()
	_, err := e.db.Exec(`INSERT INTO user_profiles (user_id, period_duration, cycle_duration, created_at, updated_at)
		VALUES (?, 5, 28, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, userID)
	require.NoError(t, err)
}

const bleedingBody = `{"categories":{"bleeding":{"flow":"medium"}}}`

func (e *logsEnv) bleed(t *testing.T, tok string, dates ...string) {
	t.Helper()
	for _, d := range dates {
		r := e.do(t, "PUT", "/api/v1/logs/days/"+d, tok, "fa", bleedingBody)
		require.Equal(t, 200, r.status, "%s: %s", d, r.raw)
	}
}

func (e *logsEnv) logPeriod(t *testing.T, tok, start, end string) {
	t.Helper()
	r := e.do(t, "POST", "/api/v1/cycle/period", tok, "fa", `{"start_date":"`+start+`","end_date":"`+end+`"}`)
	require.Equal(t, 200, r.status, r.raw)
}

func assertNoNegativeLengths(t *testing.T, rows []periodRow) {
	t.Helper()
	for _, r := range rows {
		if r.cycleLength.Valid {
			assert.Positive(t, r.cycleLength.Int64, "period %s", r.start)
		}
	}
}

func TestPastBleedingInsideConfirmedPeriod_Repro(t *testing.T) {
	e := setupLogs(t)
	uid, tok := e.user(t, "09120000901", "")
	e.withProfile(t, uid)

	e.logPeriod(t, tok, "2026-06-01", "2026-06-05")
	require.Equal(t, "2026-06-01", e.lmp(t, uid))

	e.bleed(t, tok, "2026-06-01", "2026-06-02", "2026-06-04") // 06-03 skipped
	rows := e.periods(t, uid)
	require.Len(t, rows, 1, "still one period: %+v", rows)
	assert.Equal(t, "2026-06-01", rows[0].start)
	assert.Equal(t, "2026-06-05", rows[0].end, "the logged end is kept")
	assert.True(t, rows[0].confirmed)
	assert.Equal(t, "2026-06-01", e.lmp(t, uid), "the LMP does not move")
}

// T-M9-28 (I-02): a button row is never edited — flow right after its end or right before its start is only a log
// (D-55 used to move the end / start).
func TestFlowNextToAButtonPeriodLeavesIt(t *testing.T) {
	e := setupLogs(t)
	uid, tok := e.user(t, "09120000902", "")
	e.withProfile(t, uid)
	e.logPeriod(t, tok, "2026-06-01", "2026-06-05")

	e.bleed(t, tok, "2026-06-07") // one day without a log after the end: covered by the button row, no new period
	rows := e.periods(t, uid)
	require.Len(t, rows, 1, "%+v", rows)
	assert.Equal(t, "2026-06-05", rows[0].end, "the button row's end is kept")

	e.bleed(t, tok, "2026-05-30") // two days before the start: not moved, no new period
	rows = e.periods(t, uid)
	require.Len(t, rows, 1, "%+v", rows)
	assert.Equal(t, "2026-06-01", rows[0].start)
	assert.Equal(t, "2026-06-05", rows[0].end)
	assert.Equal(t, "2026-06-01", e.lmp(t, uid))
}

func TestBackDatedBleedingBetweenPeriods(t *testing.T) {
	e := setupLogs(t)
	uid, tok := e.user(t, "09120000903", "")
	e.withProfile(t, uid)
	e.logPeriod(t, tok, "2026-06-01", "2026-06-05")
	e.logPeriod(t, tok, "2026-07-01", "2026-07-05")
	require.Equal(t, "2026-07-01", e.lmp(t, uid))

	e.bleed(t, tok, "2026-06-15")
	rows := e.periods(t, uid)
	require.Len(t, rows, 3, "%+v", rows)
	assert.Equal(t, periodRow{start: "2026-06-15", end: "2026-06-15", cycleLength: sql.NullInt64{Int64: 14, Valid: true}, confirmed: true}, rows[1],
		"a closed, confirmed one-day period with a positive cycle length")
	assert.Equal(t, sql.NullInt64{Int64: 16, Valid: true}, rows[2].cycleLength, "the next row's cycle_length is recomputed (B5)")
	assert.Equal(t, "2026-07-01", e.lmp(t, uid), "the LMP stays on the latest period")
	assertNoNegativeLengths(t, rows)

	e.bleed(t, tok, "2026-06-16") // the next day extends the back-dated period
	rows = e.periods(t, uid)
	require.Len(t, rows, 3, "%+v", rows)
	assert.Equal(t, "2026-06-16", rows[1].end)
}

func TestNewBleedingAfterTheLatestPeriodStartsOne(t *testing.T) {
	e := setupLogs(t)
	uid, tok := e.user(t, "09120000904", "")
	e.withProfile(t, uid)
	e.logPeriod(t, tok, "2026-08-20", "2026-08-24")

	e.bleed(t, tok, "2026-09-20") // a new cycle (Laravel behaviour kept)
	rows := e.periods(t, uid)
	require.Len(t, rows, 2, "%+v", rows)
	assert.Equal(t, periodRow{start: "2026-09-20", cycleLength: sql.NullInt64{Int64: 31, Valid: true}, confirmed: true}, rows[1], "open and confirmed (I-02)")
	assert.Equal(t, "2026-08-24", rows[0].end, "a logged end is never cut")
	assert.Equal(t, "2026-09-20", e.lmp(t, uid))

	e.bleed(t, tok, "2026-09-22") // a skipped day inside the open period → no new period
	assert.Len(t, e.periods(t, uid), 2)
	assertNoNegativeLengths(t, e.periods(t, uid))
}
