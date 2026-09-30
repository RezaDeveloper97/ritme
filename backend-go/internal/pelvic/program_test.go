package pelvic

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func item(code, meta string) catalog.Item {
	it := catalog.Item{Code: code, Title: json.RawMessage(`{"fa":"سطح","en":"Level"}`)}
	if meta != "" {
		it.Meta = json.RawMessage(meta)
	}
	return it
}

func TestParseLevels_SkipsBadMetaAndSortsByWeek(t *testing.T) {
	levels := ParseLevels([]catalog.Item{
		item("late", `{"week_from":5,"hold_sec":8,"rest_sec":8,"reps":10,"sets":3}`),
		item("no_meta", ""),
		item("zero_reps", `{"week_from":2,"hold_sec":3,"rest_sec":3,"reps":0,"sets":3}`),
		item("string_meta", `{"week_from":"3","hold_sec":5,"rest_sec":5,"reps":10,"sets":3}`),
		item("early", `{"week_from":1,"hold_sec":3,"rest_sec":3,"reps":10,"sets":3}`),
	})
	require.Len(t, levels, 2)
	assert.Equal(t, "early", levels[0].Item.Code)
	assert.Equal(t, 1, levels[0].Number)
	assert.Equal(t, "late", levels[1].Item.Code)
	assert.Equal(t, 2, levels[1].Number)
	assert.Equal(t, 180, levels[0].SessionSec(), "3 sets × 10 reps × (3 + 3) s")
}

func TestLevelFor(t *testing.T) {
	levels := ParseLevels([]catalog.Item{
		item("l1", `{"week_from":1,"hold_sec":3,"rest_sec":3,"reps":10,"sets":3}`),
		item("l2", `{"week_from":3,"hold_sec":5,"rest_sec":5,"reps":10,"sets":3}`),
		item("l3", `{"week_from":5,"hold_sec":8,"rest_sec":8,"reps":10,"sets":3}`),
	})
	for week, want := range map[int]string{1: "l1", 2: "l1", 3: "l2", 4: "l2", 5: "l3", 8: "l3"} {
		assert.Equal(t, want, LevelFor(levels, week).Item.Code, "week %d", week)
	}
	// Board: week 3, level 2, 5-minute session.
	l := LevelFor(levels, 3)
	assert.Equal(t, 2, l.Number)
	assert.Equal(t, 300, l.SessionSec())

	assert.Nil(t, LevelFor(nil, 1))
	onlyLate := ParseLevels([]catalog.Item{item("x", `{"week_from":4,"hold_sec":3,"rest_sec":3,"reps":10,"sets":3}`)})
	assert.Equal(t, "x", LevelFor(onlyLate, 1).Item.Code, "no level starts yet → the first one")
}

func TestWeek(t *testing.T) {
	start := civildate.MustParse("2026-09-01")
	cases := []struct {
		today     string
		week      int
		completed bool
	}{
		{"2026-09-01", 1, false},
		{"2026-09-07", 1, false},
		{"2026-09-08", 2, false},
		{"2026-10-26", 8, false}, // day 55
		{"2026-10-27", 8, true},  // day 56: the 8 weeks are over
		{"2027-01-01", 8, true},
		{"2026-08-20", 1, false}, // start in the future of today (clock skew) → week 1
	}
	for _, c := range cases {
		w, done := Week(start, civildate.MustParse(c.today))
		assert.Equal(t, c.week, w, c.today)
		assert.Equal(t, c.completed, done, c.today)
	}
}

func dates(ds ...string) map[civildate.Date]bool {
	m := map[civildate.Date]bool{}
	for _, d := range ds {
		m[civildate.MustParse(d)] = true
	}
	return m
}

func TestStreak(t *testing.T) {
	today := civildate.MustParse("2026-09-23")
	assert.Equal(t, 0, Streak(dates(), today))
	assert.Equal(t, 3, Streak(dates("2026-09-23", "2026-09-22", "2026-09-21", "2026-09-19"), today))
	assert.Equal(t, 2, Streak(dates("2026-09-22", "2026-09-21"), today), "today not trained yet: counts to yesterday")
	assert.Equal(t, 0, Streak(dates("2026-09-21"), today), "a gap yesterday breaks it")
}

func TestWeekDays_SaturdayStart(t *testing.T) {
	today := civildate.MustParse("2026-09-23") // a Wednesday
	days := WeekDays(dates("2026-09-20", "2026-09-23", "2026-09-19"), today)
	require.Len(t, days, 7)
	assert.Equal(t, "2026-09-19", days[0].Date.String())
	assert.Equal(t, "2026-09-25", days[6].Date.String())
	got := []bool{}
	for _, d := range days {
		got = append(got, d.Done)
	}
	assert.Equal(t, []bool{true, true, false, false, true, false, false}, got)
}

var now = time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)

func body(t *testing.T, s string) phpval.Map {
	t.Helper()
	v, err := phpval.Decode([]byte(s))
	require.NoError(t, err)
	m, ok := v.(phpval.Map)
	require.True(t, ok)
	return m
}

func errorsOf(t *testing.T, err error) map[string]any {
	t.Helper()
	var fe *httpx.FailError
	require.True(t, errors.As(err, &fe), "want a controller 422, got %v", err)
	assert.Equal(t, 422, fe.Status)
	raw, jerr := json.Marshal(fe.Body())
	require.NoError(t, jerr)
	var out struct {
		Errors map[string]any `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	return out.Errors
}

func TestValidateDiary(t *testing.T) {
	in, err := validateDiary("2026-09-23", body(t, `{"leak":"cough","uti_symptoms":["cloudy_odor","burning","burning"],"extra":1}`), "en", now)
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"leak": true, "uti_symptoms": true}, in.Set)
	assert.Equal(t, "cough", *in.Leak)
	assert.Nil(t, in.NightVoids)
	assert.Equal(t, []string{"burning", "cloudy_odor"}, in.UTISymptoms, "known order, de-duplicated")

	in, err = validateDiary("2026-09-23", body(t, `{"night_voids":null,"uti_symptoms":null}`), "en", now)
	require.NoError(t, err)
	assert.True(t, in.Set["night_voids"])
	assert.Nil(t, in.NightVoids)
	assert.Nil(t, in.UTISymptoms)

	_, err = validateDiary("2026-09-24", body(t, `{"leak":"sometimes","night_voids":21,"uti_symptoms":["fever"]}`), "en", now)
	errs := errorsOf(t, err)
	assert.Equal(t, []any{"You cannot log a future day."}, errs["date"])
	assert.Contains(t, errs, "leak")
	assert.Contains(t, errs, "night_voids")
	assert.Contains(t, errs, "uti_symptoms.0")
}

func TestValidateSession(t *testing.T) {
	in, err := validateSession(body(t, `{"sets_completed":"3","duration_sec":300}`), "en", now)
	require.NoError(t, err)
	assert.Equal(t, SessionInput{Date: civildate.MustParse("2026-09-23"), SetsCompleted: 3, DurationSec: 300}, in)

	_, err = validateSession(body(t, `{"sets_completed":21,"date":"2026-09-30"}`), "fa", now)
	errs := errorsOf(t, err)
	assert.Contains(t, errs, "sets_completed")
	assert.Contains(t, errs, "duration_sec")
	assert.Equal(t, []any{"ثبت برای روزهای آینده ممکن نیست."}, errs["date"])
}

func TestValidateProgram_DefaultsToToday(t *testing.T) {
	d, err := validateProgram(body(t, `{}`), "en", now)
	require.NoError(t, err)
	assert.Equal(t, "2026-09-23", d.String())
	d, err = validateProgram(body(t, `{"started_on":"2026-09-02"}`), "en", now)
	require.NoError(t, err)
	assert.Equal(t, "2026-09-02", d.String())
	_, err = validateProgram(body(t, `{"started_on":"tomorrow"}`), "en", now)
	assert.Contains(t, errorsOf(t, err), "started_on")
}

func TestDiaryJSON_UTIAlert(t *testing.T) {
	d := Diary{Date: civildate.MustParse("2026-09-23"), UTISymptoms: []string{}}
	raw, err := jsonx.Marshal(DiaryJSON(d), 0)
	require.NoError(t, err)
	assert.JSONEq(t, `{"date":"2026-09-23","leak":null,"night_voids":null,"uti_symptoms":[],"uti_alert":false}`, string(raw))
	d.UTISymptoms = []string{"burning"}
	raw, err = jsonx.Marshal(DiaryJSON(d), 0)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"uti_alert":true`)
}

func TestLangFiles_SameKeys(t *testing.T) {
	for _, key := range []string{"messages.validation_failed", "messages.program_started", "messages.program_stopped",
		"messages.session_saved", "messages.diary_saved", "validation.date_future", "validation.program_required"} {
		for _, loc := range []string{"fa", "en"} {
			assert.NotEqual(t, "pelvic."+key, T(key, loc), "%s missing in %s", key, loc)
		}
	}
	assert.Len(t, attributes("fa"), len(attributes("en")))
}
