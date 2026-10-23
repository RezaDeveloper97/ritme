package healthlog_test

// Engine v6 bleeding-log bridge (T-M9-28; SPEC v6 §1.4, §2.39.9; DECISIONS.md I-02, I-05, H-14; D-124, D-138): flow
// days build confirmed bleeding_log periods, removals rebuild them, button rows are never touched, every write
// recomputes cycle lengths and re-anchors the LMP under the per-user lock; pregnancy flow only alerts; postpartum
// flow counts from birth + 42 days.

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// bridgeRow is one cycle_histories row: "start end bleeding cycle_length source confirmed" ("-" = NULL).
func (e *logsEnv) bridgeRows(t *testing.T, userID uint64) []string {
	t.Helper()
	rows, err := e.db.Query(`SELECT CAST(period_start_date AS CHAR), COALESCE(CAST(period_end_date AS CHAR), '-'),
		COALESCE(CAST(bleeding_length AS CHAR), '-'), COALESCE(CAST(cycle_length AS CHAR), '-'), source, is_confirmed
		FROM cycle_histories WHERE user_id = ? ORDER BY period_start_date`, userID)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	out := []string{}
	for rows.Next() {
		var start, end, bleeding, length, source string
		var confirmed int
		require.NoError(t, rows.Scan(&start, &end, &bleeding, &length, &source, &confirmed))
		out = append(out, fmt.Sprintf("%s %s %s %s %s %d", start, end, bleeding, length, source, confirmed))
	}
	require.NoError(t, rows.Err())
	return out
}

func (e *logsEnv) clearFlow(t *testing.T, tok string, dates ...string) {
	t.Helper()
	for _, d := range dates {
		r := e.do(t, "PUT", "/api/v1/logs/days/"+d, tok, "fa", `{"categories":{"bleeding":{"flow":null}}}`)
		require.Equal(t, 200, r.status, "%s: %s", d, r.raw)
	}
}

func (e *logsEnv) deleteDay(t *testing.T, tok, date string) {
	t.Helper()
	r := e.do(t, "DELETE", "/api/v1/logs/days/"+date, tok, "fa", "")
	require.Equal(t, 200, r.status, r.raw)
}

func days(from string, n int) []string {
	d := civildate.MustParse(from)
	out := make([]string, n)
	for i := range out {
		out[i] = d.AddDays(i).String()
	}
	return out
}

func TestBridge_Table(t *testing.T) {
	e := setupLogs(t)
	cases := []struct {
		name string
		run  func(t *testing.T, uid uint64, tok string)
		want []string
		lmp  string
	}{
		{
			name: "forward start: an open, confirmed bleeding_log period; the LMP follows",
			run:  func(t *testing.T, _ uint64, tok string) { e.bleed(t, tok, "2026-09-20") },
			want: []string{"2026-09-20 - - - bleeding_log 1"},
			lmp:  "2026-09-20",
		},
		{
			name: "extending an open period: later days and a skipped day stay inside it",
			run:  func(t *testing.T, _ uint64, tok string) { e.bleed(t, tok, "2026-09-18", "2026-09-19", "2026-09-21") },
			want: []string{"2026-09-18 - - - bleeding_log 1"},
			lmp:  "2026-09-18",
		},
		{
			name: "extending a closed period: a day two days after its end moves the end",
			run: func(t *testing.T, _ uint64, tok string) {
				e.bleed(t, tok, days("2026-08-01", 3)...)
				e.bleed(t, tok, "2026-09-01") // closes 08-01 at its last bleeding day (08-03)
				e.bleed(t, tok, "2026-08-05")
			},
			want: []string{"2026-08-01 2026-08-05 5 - bleeding_log 1", "2026-09-01 - - 31 bleeding_log 1"},
			lmp:  "2026-09-01",
		},
		{
			name: "a back-dated day between two periods: closed one-day period, the next cycle_length recomputed",
			run: func(t *testing.T, _ uint64, tok string) {
				e.bleed(t, tok, days("2026-08-01", 3)...)
				e.bleed(t, tok, "2026-09-01")
				e.bleed(t, tok, "2026-08-15")
			},
			want: []string{"2026-08-01 2026-08-03 3 - bleeding_log 1", "2026-08-15 2026-08-15 1 14 bleeding_log 1",
				"2026-09-01 - - 17 bleeding_log 1"},
			lmp: "2026-09-01",
		},
		{
			name: "deleting the only bleeding day deletes the period (and the LMP with it)",
			run: func(t *testing.T, _ uint64, tok string) {
				e.bleed(t, tok, "2026-09-20")
				e.deleteDay(t, tok, "2026-09-20")
			},
			want: []string{},
			lmp:  "",
		},
		{
			name: "clearing the middle day of a 5-day period: a one-day gap does not split it",
			run: func(t *testing.T, _ uint64, tok string) {
				e.bleed(t, tok, days("2026-08-01", 5)...)
				e.bleed(t, tok, "2026-09-01")
				e.clearFlow(t, tok, "2026-08-03")
			},
			want: []string{"2026-08-01 2026-08-05 5 - bleeding_log 1", "2026-09-01 - - 31 bleeding_log 1"},
			lmp:  "2026-09-01",
		},
		{
			name: "a gap wider than the adjacency splits the period; cycle lengths follow",
			run: func(t *testing.T, _ uint64, tok string) {
				e.bleed(t, tok, days("2026-08-01", 5)...)
				e.bleed(t, tok, "2026-09-01")
				e.clearFlow(t, tok, "2026-08-02", "2026-08-03")
			},
			want: []string{"2026-08-01 2026-08-01 1 - bleeding_log 1", "2026-08-04 2026-08-05 2 3 bleeding_log 1",
				"2026-09-01 - - 28 bleeding_log 1"},
			lmp: "2026-09-01",
		},
		{
			name: "clearing the last day shrinks the end",
			run: func(t *testing.T, _ uint64, tok string) {
				e.bleed(t, tok, days("2026-08-01", 5)...)
				e.bleed(t, tok, "2026-09-01")
				e.clearFlow(t, tok, "2026-08-05")
			},
			want: []string{"2026-08-01 2026-08-04 4 - bleeding_log 1", "2026-09-01 - - 31 bleeding_log 1"},
			lmp:  "2026-09-01",
		},
		{
			name: "clearing the first day of the open period moves its start",
			run: func(t *testing.T, _ uint64, tok string) {
				e.bleed(t, tok, "2026-09-18", "2026-09-19")
				e.clearFlow(t, tok, "2026-09-18")
			},
			want: []string{"2026-09-19 - - - bleeding_log 1"},
			lmp:  "2026-09-19",
		},
		{
			name: "the old DELETE /health-logs/{date} rebuilds too",
			run: func(t *testing.T, _ uint64, tok string) {
				e.bleed(t, tok, "2026-09-20", "2026-09-21")
				r := e.do(t, "DELETE", "/api/v1/health-logs/2026-09-20", tok, "fa", "")
				require.Equal(t, 200, r.status, r.raw)
			},
			want: []string{"2026-09-21 - - - bleeding_log 1"},
			lmp:  "2026-09-21",
		},
		{
			name: "spotting creates nothing",
			run: func(t *testing.T, _ uint64, tok string) {
				r := e.do(t, "PUT", "/api/v1/logs/days/2026-09-20", tok, "fa", `{"categories":{"bleeding":{"spotting":true}}}`)
				require.Equal(t, 200, r.status, r.raw)
			},
			want: []string{},
			lmp:  "",
		},
		{
			name: "merging: re-adding the gap day joins the two halves again",
			run: func(t *testing.T, _ uint64, tok string) {
				e.bleed(t, tok, days("2026-08-01", 5)...)
				e.bleed(t, tok, "2026-09-01")
				e.clearFlow(t, tok, "2026-08-02", "2026-08-03")
				e.bleed(t, tok, "2026-08-02")
			},
			want: []string{"2026-08-01 2026-08-05 5 - bleeding_log 1", "2026-09-01 - - 31 bleeding_log 1"},
			lmp:  "2026-09-01",
		},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			uid, tok := e.user(t, "0912100"+strconv.Itoa(1000+i), "")
			e.withProfile(t, uid)
			c.run(t, uid, tok)
			assert.Equal(t, c.want, e.bridgeRows(t, uid))
			assert.Equal(t, c.lmp, e.lmp(t, uid))
		})
	}
}

// A period-button row is never modified by a bridge rebuild (I-02), not even its updated_at.
func TestBridge_ButtonRowNeverRebuilt(t *testing.T) {
	e := setupLogs(t)
	uid, tok := e.user(t, "09121002001", "")
	e.withProfile(t, uid)
	e.logPeriod(t, tok, "2026-08-01", "2026-08-05")
	var before string
	require.NoError(t, e.db.QueryRow(`SELECT CAST(updated_at AS CHAR) FROM cycle_histories WHERE user_id = ?`, uid).Scan(&before))

	e.bleed(t, tok, "2026-08-02", "2026-08-03")
	e.clearFlow(t, tok, "2026-08-02")
	e.deleteDay(t, tok, "2026-08-03")

	assert.Equal(t, []string{"2026-08-01 2026-08-05 5 - user_logged 1"}, e.bridgeRows(t, uid))
	var after string
	require.NoError(t, e.db.QueryRow(`SELECT CAST(updated_at AS CHAR) FROM cycle_histories WHERE user_id = ?`, uid).Scan(&after))
	assert.Equal(t, before, after)
}

// PUT /cycle/period/{id} on a bridge row makes it a confirmed user_logged row (gap 02 S5); the bridge then leaves it.
func TestBridge_EditedRowBecomesUserLogged(t *testing.T) {
	e := setupLogs(t)
	uid, tok := e.user(t, "09121002002", "")
	e.withProfile(t, uid)
	e.bleed(t, tok, days("2026-08-01", 3)...)
	e.bleed(t, tok, "2026-09-01")
	var id uint64
	require.NoError(t, e.db.QueryRow(`SELECT id FROM cycle_histories WHERE user_id = ? AND period_start_date = '2026-08-01'`, uid).Scan(&id))

	r := e.do(t, "PUT", fmt.Sprintf("/api/v1/cycle/period/%d", id), tok, "fa", `{"start_date":"2026-08-01","end_date":"2026-08-04"}`)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, []string{"2026-08-01 2026-08-04 4 - user_logged 1", "2026-09-01 - - 31 bleeding_log 1"}, e.bridgeRows(t, uid))

	e.clearFlow(t, tok, "2026-08-03")
	assert.Equal(t, []string{"2026-08-01 2026-08-04 4 - user_logged 1", "2026-09-01 - - 31 bleeding_log 1"}, e.bridgeRows(t, uid),
		"the edited row is the user's, never rebuilt")
}

// Pregnancy mode: flow creates no period and keeps the LMP; it raises the bleeding alert once (a re-save or a
// lighter value does not alert again; heavier flow does).
func TestBridge_PregnancyFlowAlertsOnly(t *testing.T) {
	e := setupLogs(t)
	uid, tok := e.user(t, "09121002003", "")
	e.withProfile(t, uid)
	_, err := e.db.Exec(`UPDATE user_profiles SET last_period_start = '2026-07-01' WHERE user_id = ?`, uid)
	require.NoError(t, err)
	_, err = e.db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2026-07-01', NOW(), NOW())`, uid)
	require.NoError(t, err)
	alerts := func() int {
		return e.count(t, `SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ? AND alert_type = 'symptom_based'`, uid)
	}

	e.bleed(t, tok, "2026-09-20")
	assert.Empty(t, e.bridgeRows(t, uid), "no period in pregnancy")
	assert.Equal(t, "2026-07-01", e.lmp(t, uid), "the LMP does not move")
	assert.Equal(t, 1, alerts())
	var level, trigger string
	require.NoError(t, e.db.QueryRow(`SELECT alert_level, CAST(trigger_symptoms AS CHAR) FROM pregnancy_alerts WHERE user_id = ?`, uid).Scan(&level, &trigger))
	assert.Equal(t, "warning", level)
	assert.JSONEq(t, `{"bleeding":"moderate","flow":"medium","source":"day_log"}`, trigger)

	e.bleed(t, tok, "2026-09-20") // the same value again
	assert.Equal(t, 1, alerts(), "raised once")

	r := e.do(t, "PUT", "/api/v1/logs/days/2026-09-20", tok, "fa", `{"categories":{"bleeding":{"flow":"very_heavy"}}}`)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, 2, alerts(), "heavier flow raises the emergency")
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ? AND alert_level = 'emergency'`, uid))

	e.clearFlow(t, tok, "2026-09-20")
	assert.Empty(t, e.bridgeRows(t, uid))
	assert.Equal(t, 2, alerts())
}

// Postpartum: flow is lochia before birth + 42 days (not offered, a 422 on the v2 sheet, ignored by the bridge on the
// old endpoint); from day 42 it is offered and builds a period as in cycle mode (I-05, H-14).
func TestBridge_PostpartumFlowFromDay42(t *testing.T) {
	e := setupLogs(t)
	uid, tok := e.user(t, "09121002004", "postpartum")
	e.withProfile(t, uid)
	// logsNow is 2026-09-23: birth 2026-08-12 → 09-22 is day 41, 09-23 day 42.
	_, err := e.db.Exec(`INSERT INTO postpartum_profiles (user_id, birth_date, baby_count, source, created_at, updated_at)
		VALUES (?, '2026-08-12', 1, 'direct', NOW(), NOW())`, uid)
	require.NoError(t, err)

	tax := e.do(t, "GET", "/api/v1/logs/taxonomy", tok, "fa", "")
	require.Equal(t, 200, tax.status, tax.raw)
	assert.Contains(t, tax.raw, `"code":"flow"`, "flow is offered on day 42")

	r := e.do(t, "PUT", "/api/v1/logs/days/2026-09-22", tok, "fa", bleedingBody)
	assert.Equal(t, 422, r.status, "day 41: flow is not accepted: %s", r.raw)
	assert.Contains(t, r.errorKeys(), "categories.bleeding.flow")

	r = e.do(t, "POST", "/api/v1/health-logs", tok, "fa", `{"log_date":"2026-09-22","bleeding_intensity":"medium"}`)
	require.Equal(t, 201, r.status, r.raw)
	assert.Empty(t, e.bridgeRows(t, uid), "day 41 on the old endpoint: lochia, no period")

	e.bleed(t, tok, "2026-09-23")
	assert.Equal(t, []string{"2026-09-23 - - - bleeding_log 1"}, e.bridgeRows(t, uid),
		"day 42: a period, although day 41 bled (lochia does not continue a period)")
	assert.Equal(t, "2026-09-23", e.lmp(t, uid))

	// Birth later (day 41 today): flow is not offered yet.
	other, otherTok := e.user(t, "09121002005", "postpartum")
	_, err = e.db.Exec(`INSERT INTO postpartum_profiles (user_id, birth_date, baby_count, source, created_at, updated_at)
		VALUES (?, '2026-08-13', 1, 'direct', NOW(), NOW())`, other)
	require.NoError(t, err)
	tax = e.do(t, "GET", "/api/v1/logs/taxonomy", otherTok, "fa", "")
	require.Equal(t, 200, tax.status, tax.raw)
	assert.NotContains(t, tax.raw, `"code":"flow"`)
}

// 20 concurrent saves of overlapping flow days: the per-user lock serialises the bridge, so the periods never
// overlap and end up as the one period the days make.
func TestBridge_ConcurrentSavesNeverOverlap(t *testing.T) {
	e := setupLogs(t)
	uid, tok := e.user(t, "09121002006", "")
	e.withProfile(t, uid)
	order := []string{"2026-09-05", "2026-09-01", "2026-09-03", "2026-09-08", "2026-09-02", "2026-09-06", "2026-09-04",
		"2026-09-10", "2026-09-07", "2026-09-09"}
	var wg sync.WaitGroup
	statuses := make([]int, 2*len(order))
	for i := range statuses {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			statuses[i] = e.do(t, "PUT", "/api/v1/logs/days/"+order[i%len(order)], tok, "fa", bleedingBody).status
		}(i)
	}
	wg.Wait()
	for i, s := range statuses {
		assert.Equal(t, 200, s, "save %d", i)
	}
	rows, err := e.db.Query(`SELECT period_start_date, period_end_date FROM cycle_histories WHERE user_id = ? ORDER BY period_start_date`, uid)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	type span struct {
		start civildate.Date
		end   civildate.NullDate
	}
	var spans []span
	for rows.Next() {
		var s span
		require.NoError(t, rows.Scan(&s.start, &s.end))
		spans = append(spans, s)
	}
	require.NoError(t, rows.Err())
	for i := 1; i < len(spans); i++ {
		prev := spans[i-1]
		require.True(t, prev.end.Valid, "only the newest period may be open: %+v", spans)
		assert.True(t, prev.end.Date.Before(spans[i].start), "periods overlap: %+v", spans)
	}
	assert.Equal(t, []string{"2026-09-01 - - - bleeding_log 1"}, e.bridgeRows(t, uid), "ten adjacent days are one period")
}
