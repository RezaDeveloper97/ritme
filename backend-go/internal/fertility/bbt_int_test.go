package fertility_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/fertility"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// period logs a confirmed period start (5-day bleed) in the cycle history.
func (e *env) period(t *testing.T, userID uint64, start string) {
	t.Helper()
	end := civildate.MustParse(start).AddDays(4).String()
	_, err := e.db.Exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, bleeding_length, is_confirmed, is_estimated, source, created_at, updated_at)
		VALUES (?, ?, ?, 5, 1, 0, 'user_logged', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, userID, start, end)
	require.NoError(t, err)
}

// temps logs BBT readings on consecutive days from `from` (0 = no reading that day).
func (e *env) temps(t *testing.T, userID uint64, from string, hundredths ...int) {
	t.Helper()
	d := civildate.MustParse(from)
	for i, v := range hundredths {
		if v == 0 {
			continue
		}
		_, err := e.db.Exec(`INSERT INTO daily_health_logs (user_id, log_date, basal_body_temperature, created_at, updated_at)
			VALUES (?, ?, ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`,
			userID, d.AddDays(i).String(), fmt.Sprintf("%d.%02d", v/100, v%100))
		require.NoError(t, err)
	}
}

// biphasic is a 28-day cycle whose readings rise on shiftDay.
func biphasic(shiftDay int) []int {
	out := make([]int, 28)
	for i := range out {
		out[i] = 3620 + (i%3)*5
		if i+1 >= shiftDay {
			out[i] = 3680
		}
	}
	return out
}

// A TTC user with two finished biphasic cycles (shifts on day 15 and 16) and a current cycle
// (started 2026-09-10, today = day 14) logged on days 1–13.
func bbtUser(t *testing.T, e *env, mobile string) (uint64, string) {
	t.Helper()
	id, tok := e.user(t, mobile, "2026-09-10")
	for _, s := range []string{"2026-07-16", "2026-08-13", "2026-09-10"} {
		e.period(t, id, s)
	}
	e.temps(t, id, "2026-07-16", biphasic(15)...)
	e.temps(t, id, "2026-08-13", biphasic(16)...)
	e.temps(t, id, "2026-09-10", 3620, 3630, 3610, 3640, 3625, 3635, 3630, 3620, 3645, 3630, 3625, 3635, 3630)
	return id, tok
}

func TestBBT_CurrentCycle(t *testing.T) {
	e := setup(t)
	_, tok := bbtUser(t, e, "09120000601")

	e.queries.n.Store(0)
	r := e.do(t, http.MethodGet, "/api/v1/fertility/bbt", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.EqualValues(t, 3, e.queries.n.Load(), "profile, cycle history, readings")
	assert.Regexp(t, `^\{"success":true,"data":\{"range":1,"cycles":\[\{"start_date":"2026-09-10","points":\[\{"cycle_day":1,"date":"2026-09-10","value":"36.20"\}`, r.raw)
	d := r.data()
	assert.EqualValues(t, 1, d["range"])
	cycles, _ := d["cycles"].([]any)
	require.Len(t, cycles, 1)
	cur := cycles[0].(map[string]any)
	assert.Len(t, cur["points"], 13)
	assert.Equal(t, "36.45", cur["coverline"], "provisional: max of the latest 6 readings")
	assert.Nil(t, cur["shift_day"])
	assert.Equal(t, "pre_shift", cur["phase"])
	assert.Equal(t, map[string]any{"from_day": float64(10), "to_day": float64(15)}, cur["fertile_window"],
		"the cycle engine's fertile zone: ovulation 2026-09-24 (day 15) and the 5 days before")

	assert.Equal(t, map[string]any{
		"pre_ovulation_avg": "36.27", "logged_days": float64(13), "cycle_days_so_far": float64(14), "gaps": float64(1),
	}, d["stats"])
	assert.Equal(t, []any{float64(16), float64(15)}, d["past_shift_days"], "newest previous cycle first")
	tip, _ := d["tip"].(map[string]any)
	require.NotNil(t, tip)
	assert.Equal(t, "دنبال جهش ۰٫۲ تا ۰٫۵ درجه باش", tip["title"])
	assert.Contains(t, tip["body"], "در ۲ سیکل قبل این جهش روز ۱۶ و ۱۵ بود.")

	en := e.do(t, http.MethodGet, "/api/v1/fertility/bbt", tok, "en", "")
	require.Equal(t, http.StatusOK, en.status, en.raw)
	enTip := en.data()["tip"].(map[string]any)
	assert.Equal(t, "Look for a 0.2–0.5 °C rise", enTip["title"])
	assert.Contains(t, enTip["body"], "In the last 2 cycles it came on days 16 and 15.")
}

func TestBBT_Ranges(t *testing.T) {
	e := setup(t)
	_, tok := bbtUser(t, e, "09120000602")

	for _, c := range []struct {
		query  string
		cycles int
	}{{"?range=1", 1}, {"?range=3", 3}, {"?range=6", 3}} {
		r := e.do(t, http.MethodGet, "/api/v1/fertility/bbt"+c.query, tok, "fa", "")
		require.Equal(t, http.StatusOK, r.status, r.raw)
		cycles, _ := r.data()["cycles"].([]any)
		require.Len(t, cycles, c.cycles, c.query)
		assert.Equal(t, []any{float64(16), float64(15)}, r.data()["past_shift_days"], c.query)
		if c.cycles < 3 {
			continue
		}
		prev := cycles[1].(map[string]any)
		assert.Equal(t, "2026-08-13", prev["start_date"])
		assert.EqualValues(t, 16, prev["shift_day"])
		assert.Equal(t, "post_shift", prev["phase"])
		assert.Equal(t, "36.30", prev["coverline"])
		assert.Len(t, prev["points"], 28, "a finished cycle keeps its own readings only")
		first := prev["points"].([]any)[0].(map[string]any)
		assert.EqualValues(t, 1, first["cycle_day"], "aligned by cycle day")
		oldest := cycles[2].(map[string]any)
		assert.Equal(t, "2026-07-16", oldest["start_date"])
		assert.EqualValues(t, 15, oldest["shift_day"])
		assert.NotNil(t, oldest["fertile_window"])
	}
}

func TestBBT_Validation(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000603", "2026-09-10")
	for _, q := range []string{"?range=2", "?range=abc", "?range=12"} {
		r := e.do(t, http.MethodGet, "/api/v1/fertility/bbt"+q, tok, "fa", "")
		require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
		assert.Equal(t, false, r.body["success"])
		assert.Equal(t, "اطلاعات واردشده نامعتبر است", r.body["message"])
		assert.Contains(t, r.errors(), "range", q)
	}
}

func TestBBT_NoCycleAndNoReadings(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000604", "")
	r := e.do(t, http.MethodGet, "/api/v1/fertility/bbt?range=3", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, map[string]any{
		"title": fertility.T("bbt.tip.title", "fa"), "body": fertility.T("bbt.tip.body", "fa"),
	}, d["tip"], "no past shift days, no second sentence")
	delete(d, "tip")
	assert.JSONEq(t, `{"range":3,"cycles":[],
		"stats":{"pre_ovulation_avg":null,"logged_days":0,"cycle_days_so_far":0,"gaps":0},
		"past_shift_days":[]}`, mustJSON(t, d))

	// A profile but nothing logged: one empty cycle, every day so far a gap.
	_, tok2 := e.user(t, "09120000605", "2026-09-10")
	r = e.do(t, http.MethodGet, "/api/v1/fertility/bbt", tok2, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	cur := r.data()["cycles"].([]any)[0].(map[string]any)
	assert.Equal(t, []any{}, cur["points"])
	assert.Nil(t, cur["coverline"])
	assert.Equal(t, map[string]any{
		"pre_ovulation_avg": nil, "logged_days": float64(0), "cycle_days_so_far": float64(14), "gaps": float64(14),
	}, r.data()["stats"])
}

func TestBBT_OtherUsersReadingsAreInvisible(t *testing.T) {
	e := setup(t)
	bbtUser(t, e, "09120000606")
	_, tokB := e.user(t, "09120000607", "2026-09-10")
	r := e.do(t, http.MethodGet, "/api/v1/fertility/bbt?range=6", tokB, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	cycles := r.data()["cycles"].([]any)
	require.Len(t, cycles, 1)
	assert.Equal(t, []any{}, cycles[0].(map[string]any)["points"])
	assert.Equal(t, []any{}, r.data()["past_shift_days"])
}

func TestBBT_Unauthenticated(t *testing.T) {
	e := setup(t)
	r := e.do(t, http.MethodGet, "/api/v1/fertility/bbt", "", "fa", "")
	assert.Equal(t, http.StatusUnauthorized, r.status)
	assert.JSONEq(t, `{"message":"Unauthenticated.","error_code":"unauthenticated"}`, r.raw)
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}
