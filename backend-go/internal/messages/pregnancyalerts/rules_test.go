package pregnancyalerts

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

var today = civildate.MustParse("2026-09-23")

func cfg(window int, params map[string]any) Config {
	return Config{Enabled: true, Level: "follow_up", WindowDays: window, Params: params}
}

func facts(days map[int]Day) Facts {
	f := Facts{Today: today, Week: 13, WeekDay: 0, Source: "lmp", Days: map[civildate.Date]Day{}}
	for back, d := range days {
		f.Days[today.AddDays(-back)] = d
	}
	return f
}

func TestVomitingStreak(t *testing.T) {
	c := cfg(7, map[string]any{"min_streak_days": 3.0, "severe_min_count": 2.0})
	// 2 mild days: below both thresholds.
	assert.Empty(t, Detect("vomiting_streak", c, facts(map[int]Day{0: {"vomiting": "mild"}, 1: {"vomiting": "mild"}})))
	// 3 days in a row (today not logged yet: counted from yesterday).
	hits := Detect("vomiting_streak", c, facts(map[int]Day{1: {"vomiting": "mild"}, 2: {"vomiting": "severe"}, 3: {"vomiting": "mild"}}))
	require.Len(t, hits, 1)
	assert.Equal(t, [][2]string{{"days", "3"}, {"severe_count", "1"}}, hits[0].Vars)
	// A gap breaks the streak.
	assert.Empty(t, Detect("vomiting_streak", c, facts(map[int]Day{0: {"vomiting": "mild"}, 1: {"vomiting": "mild"}, 3: {"vomiting": "mild"}})))
	// 2 severe days reach severe_min_count.
	assert.Len(t, Detect("vomiting_streak", c, facts(map[int]Day{0: {"vomiting": "severe"}, 1: {"vomiting": "severe"}})), 1)
	// Disabled.
	c.Enabled = false
	assert.Empty(t, Detect("vomiting_streak", c, facts(map[int]Day{0: {"vomiting": "severe"}, 1: {"vomiting": "severe"}, 2: {"vomiting": "severe"}})))
}

func TestSevereSymptomCount(t *testing.T) {
	c := cfg(7, map[string]any{"min_count": 3.0, "symptoms": []any{"nausea", "headache"}})
	f := facts(map[int]Day{0: {"nausea": "severe", "headache": "severe"}, 3: {"nausea": "severe", "fatigue": "severe"}})
	hits := Detect("severe_symptom_count", c, f)
	require.Len(t, hits, 1)
	assert.Equal(t, [][2]string{{"count", "3"}}, hits[0].Vars)
	// Outside the window / unlisted symptoms do not count.
	f = facts(map[int]Day{0: {"nausea": "severe", "fatigue": "severe"}, 8: {"headache": "severe"}})
	assert.Empty(t, Detect("severe_symptom_count", c, f))
}

func TestCriticalSymptom(t *testing.T) {
	c := cfg(1, map[string]any{"symptoms": []any{"bleeding", "spotting"}, "spotting_until_week": 12.0})
	f := facts(map[int]Day{0: {"bleeding": "mild", "spotting": "mild"}, 1: {"bleeding": "severe"}})
	hits := Detect("critical_symptom", c, f) // week 13: mild spotting is past the window; yesterday is outside
	require.Len(t, hits, 1)
	assert.Equal(t, "bleeding@2026-09-23", hits[0].Dedupe)
	f.Week = 12
	assert.Len(t, Detect("critical_symptom", c, f), 2)
}

func TestCalendarRules(t *testing.T) {
	f := facts(nil)
	// Day 0 of week 13: too early for "weight missing" (default from_weekday 5).
	assert.Empty(t, Detect("weight_missing_week", cfg(7, map[string]any{"from_week": 13.0}), f))
	assert.Len(t, Detect("weight_missing_week", cfg(7, map[string]any{"from_week": 13.0, "from_weekday": 0.0}), f), 1)
	f.WeekDay = 5
	assert.Len(t, Detect("weight_missing_week", cfg(7, map[string]any{"from_week": 13.0}), f), 1)
	assert.Empty(t, Detect("weight_missing_week", cfg(7, map[string]any{"from_week": 14.0}), f))
	f.HasWeight = true
	assert.Empty(t, Detect("weight_missing_week", cfg(7, map[string]any{"from_week": 1.0}), f))
	f.HasWeight, f.WeekDay = false, 0

	hits := Detect("week_entered", cfg(1, nil), f)
	require.Len(t, hits, 1)
	assert.Equal(t, "w13", hits[0].Dedupe)
	f.WeekDay = 1
	assert.Empty(t, Detect("week_entered", cfg(1, nil), f))
}

func TestBPAndSugar(t *testing.T) {
	f := facts(nil)
	f.Weekly = []store.PregnancyWeeklyLog{
		{LogDate: today, SystolicPressure: sql.NullInt32{Int32: 139, Valid: true}, DiastolicPressure: sql.NullInt32{Int32: 89, Valid: true},
			FastingBloodSugar: sql.NullString{String: "95.00", Valid: true}},
		{LogDate: today.AddDays(-2), SystolicPressure: sql.NullInt32{Int32: 140, Valid: true},
			PostMealBloodSugar: sql.NullString{String: "140.50", Valid: true}},
	}
	bp := cfg(7, map[string]any{"systolic_min": 140.0, "diastolic_min": 90.0})
	hits := Detect("bp_high", bp, f)
	require.Len(t, hits, 1)
	assert.Equal(t, [][2]string{{"systolic", "140"}, {"diastolic", none}}, hits[0].Vars)

	sugar := cfg(7, map[string]any{"fasting_max": 95.0, "post_meal_max": 140.0})
	hits = Detect("sugar_high", sugar, f)
	require.Len(t, hits, 1)
	assert.Equal(t, "140.50", hits[0].Vars[1][1])
}

func TestFetalMovement(t *testing.T) {
	c := cfg(1, map[string]any{"from_week": 24.0, "statuses": []any{"reduced", "none"}})
	f := facts(nil)
	f.Fetal = []store.PregnancyFetalMovement{
		{LogDate: today, PregnancyWeek: 23, MovementStatus: "none"},
		{LogDate: today, PregnancyWeek: 24, MovementStatus: "normal"},
	}
	assert.Empty(t, Detect("fetal_movement", c, f))
	f.Fetal[0].PregnancyWeek = 24
	assert.Len(t, Detect("fetal_movement", c, f), 1)
}

func TestRenderAndLevels(t *testing.T) {
	assert.Equal(t, "emergency", V1Level("urgent"))
	assert.Equal(t, "warning", V1Level("follow_up"))
	assert.Equal(t, "info", V1Level("suggestion"))
	txt := render(map[string]any{"title": "{symptom} logged", "what_we_saw": "{systolic}/{diastolic}"},
		meta{Vars: map[string]string{"symptom": "spotting", "systolic": "150", "diastolic": none}}, "en")
	assert.Equal(t, "Spotting logged", txt.title)
	assert.Equal(t, "150/—", txt.whatWeSaw)
}

// Numeric vars (stored as ASCII) are rendered in the locale's digits: «وارد هفتهٔ ۱۰ شدی».
func TestRenderLocalizesDigits(t *testing.T) {
	week := Detect("week_entered", Config{Enabled: true, Level: "info", WindowDays: 7}, Facts{Week: 10, WeekDay: 0, Source: "lmp"})
	require.Len(t, week, 1)
	vars := map[string]string{}
	for _, kv := range week[0].Vars {
		vars[kv[0]] = kv[1]
	}
	assert.Equal(t, "10", vars["week"], "stored vars stay ASCII")
	p := map[string]any{"title": "وارد هفتهٔ {week} شدی", "what_we_saw": "{systolic}/{diastolic} · {fasting}"}
	m := meta{Vars: map[string]string{"week": vars["week"], "systolic": "150", "diastolic": none, "fasting": "95.5"}}
	fa := render(p, m, "fa")
	assert.Equal(t, "وارد هفتهٔ ۱۰ شدی", fa.title)
	assert.Equal(t, "۱۵۰/"+T("none", "fa")+" · ۹۵.۵", fa.whatWeSaw)
	en := render(p, m, "en")
	assert.Equal(t, "وارد هفتهٔ 10 شدی", en.title)
}
