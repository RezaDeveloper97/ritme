package menopause

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/care"
	"github.com/ritme/backend-go/internal/menopause/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

var testNow = time.Date(2026, 10, 1, 10, 0, 0, 0, civildate.Tehran)

func body(kv ...any) phpval.Map {
	m := phpval.NewMap()
	for i := 0; i+1 < len(kv); i += 2 {
		m.Set(kv[i].(string), kv[i+1])
	}
	return m
}

func TestValidateItem(t *testing.T) {
	in, err := validateItem(body("kind", "hrt", "name", "  Estrogen gel ", "dose", "1 pump", "schedule", "morning",
		"started_on", "2026-08-01", "remind", false), "", "en", testNow)
	require.NoError(t, err)
	assert.Equal(t, ItemInput{Kind: KindHRT, Name: "Estrogen gel", Dose: "1 pump", Schedule: ScheduleMorning,
		StartedOn: day("2026-08-01")}, in)

	in, err = validateItem(body("name", "Walk", "weekly_goal", "150", "goal_unit", "minutes", "schedule", "night"),
		KindLifestyle, "en", testNow)
	require.NoError(t, err)
	assert.Equal(t, 150, in.WeeklyGoal)
	assert.Empty(t, in.Schedule, "lifestyle goals have no schedule")

	for name, c := range map[string]struct {
		b     phpval.Map
		kind  string
		field string
	}{
		"no schedule":       {body("kind", "supplement", "name", "x"), "", "schedule"},
		"unknown kind":      {body("kind", "pill", "name", "x"), "", "kind"},
		"lifestyle no goal": {body("name", "x"), KindLifestyle, "weekly_goal"},
		"too many sessions": {body("name", "x", "weekly_goal", 30, "goal_unit", "sessions"), KindLifestyle, "weekly_goal"},
		"stopped in future": {body("name", "x", "schedule", "night", "stopped_on", "2026-10-02"), KindHRT, "stopped_on"},
		"stop before start": {body("name", "x", "schedule", "night", "started_on", "2026-09-10", "stopped_on", "2026-09-01"), KindHRT, "stopped_on"},
		"review before":     {body("name", "x", "schedule", "night", "started_on", "2026-09-10", "review_on", "2026-09-01"), KindHRT, "review_on"},
		"blank name":        {body("name", "   ", "schedule", "night"), KindHRT, "name"},
		"bad form":          {body("name", "x", "schedule", "night", "form", "gel"), KindHRT, "form"},
	} {
		_, err := validateItem(c.b, c.kind, "en", testNow)
		require.Error(t, err, name)
		raw, _ := jsonx.Marshal(errorBody(err), 0)
		assert.Contains(t, string(raw), `"`+c.field+`"`, name)
	}
}

// errorBody is the httpx error's extra fields (errors …).
func errorBody(err error) any {
	var fe *httpx.FailError
	if errors.As(err, &fe) {
		return fe.Extras
	}
	return err.Error()
}

func TestParseLogDay(t *testing.T) {
	d, err := parseLogDay("2026-09-01", "en", testNow)
	require.NoError(t, err)
	assert.Equal(t, day("2026-09-01"), d)
	for _, raw := range []string{"2026-10-02", "2026-08-31", "1-1-2026", ""} {
		_, err := parseLogDay(raw, "en", testNow)
		assert.Error(t, err, raw)
	}
}

func TestValidateIntakeAndReportMonths(t *testing.T) {
	n, err := validateIntake(body("amount", 30), ItemKind{Kind: KindHRT}, "en", testNow)
	require.NoError(t, err)
	assert.Zero(t, n, "only lifestyle goals carry an amount")
	_, err = validateIntake(body(), ItemKind{Kind: KindLifestyle, Unit: UnitMinutes}, "en", testNow)
	assert.Error(t, err)
	n, err = validateIntake(body(), ItemKind{Kind: KindLifestyle, Unit: UnitSessions}, "en", testNow)
	require.NoError(t, err)
	assert.Zero(t, n)
	_, err = validateIntake(body("amount", 11), ItemKind{Kind: KindLifestyle, Unit: UnitSessions}, "en", testNow)
	assert.Error(t, err)

	m, err := validateReportMonths(body(), "en", testNow)
	require.NoError(t, err)
	assert.Equal(t, DefaultReportMonths, m)
	m, err = validateReportMonths(body("months", "6"), "en", testNow)
	require.NoError(t, err)
	assert.Equal(t, 6, m)
	_, err = validateReportMonths(body("months", "2"), "en", testNow)
	assert.Error(t, err)

	in, err := validateSideEffects(body("codes", []any{"spotting", "headache", "spotting"}), "en", testNow)
	require.NoError(t, err)
	assert.Equal(t, []string{"spotting", "headache"}, in.Codes, "SideEffectCodes order, de-duplicated")
	_, err = validateSideEffects(body(), "en", testNow)
	assert.Error(t, err, "codes must be present (an empty list clears)")
}

func TestMedicationMeta(t *testing.T) {
	in := ItemInput{Kind: KindHRT, Name: "Progesterone", Dose: "100 mg", Schedule: ScheduleWeekly,
		StartedOn: day("2026-09-28"), Remind: true} // a Monday
	m := in.medicationMeta(day("2026-10-01"))
	assert.Equal(t, []string{"09:00"}, m.Times)
	assert.Equal(t, []int{care.SaturdayWeekday(day("2026-09-28"))}, m.Weekdays)
	assert.Equal(t, care.Forms[0], m.Form)
	require.NotNil(t, m.Dose)
	assert.Equal(t, "100 mg", *m.Dose)
	assert.True(t, m.Notify)

	cols, err := ItemInput{Kind: KindSupplement, Name: "x", Schedule: ScheduleNight, StoppedOn: day("2026-10-01")}.
		careColumns(day("2026-10-01"), "en")
	require.NoError(t, err)
	assert.False(t, cols.IsActive, "stopped today → reminder off")
	assert.Equal(t, "daily", cols.Recurrence)
	assert.Equal(t, day("2026-10-01"), cols.StartsOn.Date, "no start → today")
}

func item(kind string, start, stop string) store.TreatmentItem {
	it := store.TreatmentItem{ID: 1, Kind: kind, Name: "x"}
	if start != "" {
		it.StartedOn = civildate.NullDate{Date: day(start), Valid: true}
	}
	if stop != "" {
		it.StoppedOn = civildate.NullDate{Date: day(stop), Valid: true}
	}
	return it
}

func TestAdherenceOf(t *testing.T) {
	taken := map[civildate.Date]sql.NullInt16{day("2026-09-28"): {}, day("2026-09-29"): {}, day("2026-09-01"): {}}
	a := adherenceOf(item(KindHRT, "2026-09-27", ""), nil, taken, day("2026-09-01"), day("2026-10-01"))
	assert.Equal(t, 5, a.Days, "27 Sep … 1 Oct")
	assert.Equal(t, 2, a.DaysTaken, "an intake before the start does not count")
	assert.Equal(t, 40, *a.AdherencePct())

	weekly := &care.Medication{Meta: care.MedicationMeta{Weekdays: []int{care.SaturdayWeekday(day("2026-09-28"))}}}
	a = adherenceOf(item(KindHRT, "", ""), weekly, taken, day("2026-09-21"), day("2026-10-01"))
	assert.Equal(t, 2, a.Days, "two Mondays")
	assert.Equal(t, 1, a.DaysTaken)

	walk := item(KindLifestyle, "2026-09-25", "")
	walk.GoalUnit = sql.NullString{String: UnitMinutes, Valid: true}
	a = adherenceOf(walk, nil, map[civildate.Date]sql.NullInt16{
		day("2026-09-26"): {Int16: 40, Valid: true}, day("2026-09-30"): {Int16: 30, Valid: true},
	}, day("2026-09-01"), day("2026-10-01"))
	assert.Nil(t, a.AdherencePct())
	assert.Equal(t, 7, a.ActiveDays)
	assert.InDelta(t, 70.0, *a.PerWeek(), 0.001)

	assert.True(t, overlaps(item(KindHRT, "", "2026-09-02"), day("2026-09-01"), day("2026-10-01")))
	assert.False(t, overlaps(item(KindHRT, "", "2026-09-01"), day("2026-09-01"), day("2026-10-01")))
	assert.False(t, overlaps(item(KindHRT, "2026-10-02", ""), day("2026-09-01"), day("2026-10-01")))
}

func TestReportJSON(t *testing.T) {
	assert.Equal(t, day("2026-07-01"), func() civildate.Date { f, _ := ReportWindow(day("2026-10-01"), 3); return f }())
	r := Report{From: day("2026-09-01"), To: day("2026-10-01"), ScoreMax: 44}
	assert.True(t, r.Empty())
	raw, err := jsonx.Marshal(ReportJSON(r, nil), 0)
	require.NoError(t, err)
	s := string(raw)
	for _, frag := range []string{`"score":null`, `"per_day":null`, `"avg_hours":null`, `"blood_pressure":null`,
		`"events":0`, `"dates":[]`, `"top":[]`, `"hrt_adherence_pct":null`, `"supplements":[]`, `"days":31`} {
		assert.Contains(t, s, frag)
	}

	r.BleedingDays = []civildate.Date{day("2026-09-03"), day("2026-09-04"), day("2026-09-20")}
	assert.Equal(t, []civildate.Date{day("2026-09-03"), day("2026-09-20")}, r.BleedingEvents())
	r.Items = []ItemAdherence{
		{Item: item(KindHRT, "", ""), DaysTaken: 9, Days: 10},
		{Item: item(KindHRT, "", ""), DaysTaken: 3, Days: 10},
		{Item: item(KindSupplement, "", ""), DaysTaken: 1, Days: 10},
	}
	assert.Equal(t, 60, *r.HRTAdherencePct(), "pooled over HRT items only")
	r.Symptoms = []SymptomShare{{Key: SlotHotFlashes, Days: 3}}
	r.TrackedDays = 4
	raw, err = jsonx.Marshal(ReportJSON(r, strings.ToUpper), 0)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"label":"SYMPTOMS.GENERAL.HOT_FLASHES","days":3,"percent":75`)
	assert.NotContains(t, string(raw), `"id"`)
}

func TestTreatmentReview(t *testing.T) {
	a := item(KindHRT, "2026-08-01", "")
	b := item(KindHRT, "2026-09-01", "")
	b.ID = 2
	sc := TreatmentScreen{Day: day("2026-10-01"), Items: []ItemView{{Item: a, Day: day("2026-10-01")}, {Item: b, Day: day("2026-10-01")}}}
	sc.review(3)
	require.NotNil(t, sc.Review)
	assert.Equal(t, day("2026-11-01"), *sc.Review)
	assert.True(t, sc.Suggested)

	b.ReviewOn = civildate.NullDate{Date: day("2026-12-10"), Valid: true}
	sc = TreatmentScreen{Day: day("2026-10-01"), Items: []ItemView{{Item: a, Day: day("2026-10-01")}, {Item: b, Day: day("2026-10-01")}}}
	sc.review(3)
	assert.Equal(t, day("2026-12-10"), *sc.Review)
	assert.False(t, sc.Suggested)
	assert.Equal(t, uint64(2), sc.ReviewItemID)

	assert.Equal(t, day("2026-09-26"), WeekOf(day("2026-10-01"))[0])
	assert.Equal(t, []string{"breast_tenderness", "headache", "zz"}, ordered([]string{"zz", "headache", "breast_tenderness"}, SideEffectCodes))
}
