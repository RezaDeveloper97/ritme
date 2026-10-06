package vitals

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/vitals/store"
)

func TestClassifyBP_ACCAHA2017(t *testing.T) {
	cases := []struct {
		sys, dia float64
		want     string
	}{
		{119, 79, ClassNormal},
		{120, 79, ClassElevated},
		{129, 79, ClassElevated},
		{130, 70, ClassStage1},
		{118, 80, ClassStage1}, // diastolic decides
		{139, 89, ClassStage1},
		{140, 70, ClassStage2},
		{125, 90, ClassStage2},
		{180, 120, ClassStage2}, // crisis is strictly above
		{181, 100, ClassCrisis},
		{150, 121, ClassCrisis},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, ClassifyBP(c.sys, c.dia), "%v/%v", c.sys, c.dia)
	}
}

func TestClassifyGlucose_ADA(t *testing.T) {
	cases := []struct {
		mgdl    float64
		context string
		want    string
	}{
		{53.9, ContextFasting, ClassUrgentLow},
		{54, ContextFasting, ClassLow},
		{69, ContextFasting, ClassLow},
		{70, ContextFasting, ClassInRange},
		{99, ContextFasting, ClassInRange},
		{100, ContextFasting, ClassHigh},
		{125, ContextFasting, ClassHigh},
		{126, ContextFasting, ClassVeryHigh},
		{110, ContextBeforeMeal, ClassHigh},
		{139, ContextAfterMeal, ClassInRange},
		{140, ContextAfterMeal, ClassHigh},
		{148, ContextAfterMeal, ClassHigh},
		{200, ContextRandom, ClassVeryHigh},
		{120, ContextBedtime, ClassInRange},
		{120, "unknown", ClassInRange}, // random band
	}
	for _, c := range cases {
		assert.Equal(t, c.want, ClassifyGlucose(c.mgdl, c.context), "%v %s", c.mgdl, c.context)
	}
}

func TestClassifyHR(t *testing.T) {
	assert.Equal(t, ClassLow, ClassifyHR(59, HRResting))
	assert.Equal(t, ClassNormal, ClassifyHR(60, HRResting))
	assert.Equal(t, ClassNormal, ClassifyHR(100, HRAfterWaking))
	assert.Equal(t, ClassHigh, ClassifyHR(101, HRResting))
	assert.Empty(t, ClassifyHR(140, HRAfterExercise), "the resting range does not apply after exercise")
	assert.Empty(t, ClassifyHR(120, HRStress))
	assert.Equal(t, ClassNormal, ClassifyHR(72, ""), "log-sheet values (no context) use the resting range")
}

func TestSafetyThresholds(t *testing.T) {
	assert.False(t, UrgentBP(180, 120))
	assert.True(t, UrgentBP(181, 80))
	assert.True(t, UrgentBP(150, 121))
	assert.False(t, UrgentGlucose(54))
	assert.True(t, UrgentGlucose(53.9))

	assert.Equal(t, RuleBPCrisis, AlertRule(Reading{Type: TypeBP, Systolic: 185, Diastolic: 100}))
	assert.Equal(t, RuleGlucoseLow, AlertRule(Reading{Type: TypeGlucose, MgDl: MgDlFrom(2.9, UnitMmolL)}))
	assert.Empty(t, AlertRule(Reading{Type: TypeGlucose, MgDl: 400}), "high glucose is not the urgent-low rule")
	assert.Empty(t, AlertRule(Reading{Type: TypeHR, Pulse: 200}))
}

func TestUnitConversion(t *testing.T) {
	assert.InDelta(t, 93.6, MgDlFrom(5.2, UnitMmolL), 1e-9)
	assert.InDelta(t, 94.0, MgDlFrom(94, UnitMgDl), 1e-9)
	assert.InDelta(t, 5.2, MmolL(93.6), 1e-9)
	assert.InDelta(t, 5.2, MmolL(94), 1e-9)
	assert.InDelta(t, 3.0, MmolL(54), 1e-9)
	assert.InDelta(t, 5.2, InUnit(MgDlFrom(5.2, UnitMmolL), UnitMmolL), 1e-9, "mmol/L round-trips")
	assert.InDelta(t, 94.0, InUnit(93.6, UnitMgDl), 1e-9)
	assert.InDelta(t, 54.0, MgDlFrom(3.0, UnitMmolL), 1e-9)
}

func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, civildate.Tehran)
	if err != nil {
		panic(err)
	}
	return t
}

func bp(id uint64, when string, sys, dia, pulse float64) Reading {
	m := at(when)
	return Reading{ID: id, Source: SourceVitals, Type: TypeBP, Date: civildate.FromTime(m), MeasuredAt: m,
		Systolic: sys, Diastolic: dia, Pulse: pulse}
}

func glu(id uint64, when string, mgdl float64, ctx string) Reading {
	m := at(when)
	return Reading{ID: id, Source: SourceVitals, Type: TypeGlucose, Date: civildate.FromTime(m), MeasuredAt: m,
		MgDl: mgdl, Unit: UnitMgDl, Context: ctx}
}

func decode(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

func TestReport_BP(t *testing.T) {
	today := civildate.MustParse("2026-10-04")
	rs := []Reading{
		bp(1, "2026-10-04 08:10", 118, 76, 70),
		bp(2, "2026-10-03 22:00", 132, 86, 78),
		bp(3, "2026-10-03 08:00", 120, 78, 68),
		bp(4, "2026-09-30 07:30", 116, 74, 0),
		bp(5, "2026-09-20 08:00", 190, 100, 80), // outside 7d
		{Source: SourceLog, Type: TypeBP, Date: civildate.MustParse("2026-10-01"), Systolic: 124, Diastolic: 80},
	}
	m := decode(t, Report(TypeBP, rs, NewRange("7d", today), ""))
	assert.Equal(t, map[string]any{"key": "7d", "from": "2026-09-28", "to": "2026-10-04", "days": float64(7)}, m["range"])
	assert.Equal(t, float64(5), m["readings"])
	avg := m["average"].(map[string]any)
	assert.Equal(t, float64(122), avg["systolic"]) // (118+132+120+116+124)/5 = 122
	assert.Equal(t, float64(79), avg["diastolic"]) // 394/5 = 78.8
	assert.Equal(t, "elevated", avg["classification"].(map[string]any)["code"])
	assert.Equal(t, float64(72), m["pulse"].(map[string]any)["bpm"]) // (70+78+68)/3
	assert.Equal(t, float64(116), m["min"].(map[string]any)["blood_pressure"].(map[string]any)["systolic"])
	assert.Equal(t, float64(132), m["max"].(map[string]any)["blood_pressure"].(map[string]any)["systolic"])

	dist := map[string]float64{}
	for _, d := range m["distribution"].([]any) {
		dm := d.(map[string]any)
		dist[dm["code"].(string)] = dm["count"].(float64)
	}
	assert.Equal(t, map[string]float64{"normal": 2, "elevated": 1, "stage1": 2, "stage2": 0, "crisis": 0}, dist)

	mvn := m["morning_vs_night"].(map[string]any)
	assert.Equal(t, float64(118), mvn["morning"].(map[string]any)["systolic"]) // (118+120+116)/3
	assert.Equal(t, float64(132), mvn["night"].(map[string]any)["systolic"])
	assert.Equal(t, float64(1), mvn["night_out_of_range"])
	tir := m["time_in_range"].(map[string]any)
	assert.Equal(t, float64(2), tir["in_range"])
	assert.Equal(t, float64(40), tir["percent"])
	assert.Len(t, m["series"], 4)
	assert.Equal(t, "2026-09-30", m["series"].([]any)[0].(map[string]any)["date"], "series ascending")
}

func TestReport_GlucoseTimeInRangeAndFilter(t *testing.T) {
	today := civildate.MustParse("2026-10-04")
	rs := []Reading{
		glu(1, "2026-10-04 07:45", 94, ContextFasting),
		glu(2, "2026-10-03 15:20", 148, ContextAfterMeal),
		glu(3, "2026-10-03 07:30", 89, ContextFasting),
		glu(4, "2026-10-02 14:00", 118, ContextAfterMeal),
		glu(5, "2026-10-01 07:00", 50, ContextFasting),
	}
	m := decode(t, Report(TypeGlucose, rs, NewRange("14d", today), GlucoseFilterAll))
	tir := m["time_in_range"].(map[string]any)
	assert.Equal(t, float64(3), tir["in_range"])
	assert.Equal(t, float64(1), tir["below"])
	assert.Equal(t, float64(1), tir["above"])
	assert.Equal(t, float64(60), tir["percent"])
	byCtx := m["by_context"].([]any)
	require.Len(t, byCtx, 2)
	fasting := byCtx[0].(map[string]any)
	assert.Equal(t, "fasting", fasting["context"])
	assert.Equal(t, float64(78), fasting["average"].(map[string]any)["mg_dl"]) // (94+89+50)/3 = 77.67
	assert.Equal(t, map[string]any{"min": float64(70), "max": float64(100)}, fasting["target"])
	after := byCtx[1].(map[string]any)
	assert.Equal(t, float64(1), after["above_target"])

	f := decode(t, Report(TypeGlucose, rs, NewRange("14d", today), ContextAfterMeal))
	assert.Equal(t, float64(2), f["readings"])
	assert.Equal(t, "after_meal", f["filter"])
	assert.Equal(t, float64(133), f["average"].(map[string]any)["mg_dl"])
	assert.InDelta(t, 7.4, f["average"].(map[string]any)["mmol_l"], 1e-9)
}

func TestReport_Empty(t *testing.T) {
	m := decode(t, Report(TypeHR, nil, NewRange("30d", civildate.MustParse("2026-10-04")), ""))
	assert.Equal(t, float64(0), m["readings"])
	assert.Nil(t, m["average"])
	assert.Nil(t, m["min"])
	assert.Equal(t, []any{}, m["series"])
}

func TestMerge_LogReadingsHiddenByTimedSameDay(t *testing.T) {
	timed := []Reading{bp(1, "2026-10-04 08:10", 118, 76, 70)}
	num := func(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }
	rows := []store.ListLogMeasurementsRow{
		{LogDate: civildate.MustParse("2026-10-04"), Param: "bp_systolic", ValueNum: num("130")},
		{LogDate: civildate.MustParse("2026-10-04"), Param: "bp_diastolic", ValueNum: num("85")},
		{LogDate: civildate.MustParse("2026-10-04"), Param: "heart_rate", ValueNum: num("72")},
		{LogDate: civildate.MustParse("2026-10-03"), Param: "bp_systolic", ValueNum: num("125")}, // no diastolic
		{LogDate: civildate.MustParse("2026-10-02"), Param: "blood_sugar", ValueNum: num("101.5")},
	}
	out := Merge(timed, rows)
	require.Len(t, out, 3)
	assert.Equal(t, uint64(1), out[0].ID, "the timed BP of the 4th wins over the sheet value")
	assert.Equal(t, TypeHR, out[1].Type)
	assert.Equal(t, SourceLog, out[1].Source)
	assert.Equal(t, TypeGlucose, out[2].Type)
	assert.Equal(t, ContextRandom, out[2].Context)
	j := decode(t, ReadingJSON(out[2]))
	assert.Nil(t, j["id"])
	assert.Nil(t, j["measured_at"])
	assert.Equal(t, false, j["editable"])
	assert.Equal(t, "in_range", j["classification"].(map[string]any)["code"]) // random band 70–140
}

func TestPlan_WeekAndReminders(t *testing.T) {
	today := civildate.MustParse("2026-10-06") // Tuesday; week Sat 10-03 … Fri 10-09
	items := []PlanItem{
		{Type: TypeBP, Slot: SlotMorning, Days: AllDays, RemindAt: "08:00"},
		{Type: TypeBP, Slot: SlotEvening, Days: 1<<0 | 1<<3, RemindAt: ""}, // Sat, Tue
		{Type: TypeGlucose, Slot: ContextFasting, Days: 1 << 3, RemindAt: "07:30"},
	}
	rs := []Reading{
		bp(1, "2026-10-03 08:10", 118, 76, 0), // Sat morning
		bp(2, "2026-10-03 22:00", 130, 85, 0), // Sat evening
		bp(3, "2026-10-04 13:00", 120, 80, 0), // Sun afternoon: no slot
		glu(4, "2026-10-06 07:40", 94, ContextFasting),
	}
	w := decode(t, WeekJSON(items, rs, today))
	assert.Equal(t, "2026-10-03", w["from"])
	assert.Equal(t, "2026-10-09", w["to"])
	assert.Equal(t, float64(10), w["planned"])
	assert.Equal(t, float64(3), w["done"])
	first := w["items"].([]any)[0].(map[string]any)
	days := first["days"].([]any)
	assert.Equal(t, "done", days[0].(map[string]any)["state"])
	assert.Equal(t, "missed", days[1].(map[string]any)["state"])
	assert.Equal(t, "due", days[3].(map[string]any)["state"])

	assert.Equal(t, []int{0, 3}, items[1].DayList())
	assert.Equal(t, 0, WeekdayIndex(civildate.MustParse("2026-10-03")))
	assert.Equal(t, 6, WeekdayIndex(civildate.MustParse("2026-10-09")))

	prefs := notifications.Defaults()
	occ := PlanReminders(prefs, items, rs, today, "en")
	require.Len(t, occ, 1, "glucose fasting is done today, bp evening has no reminder time")
	assert.Equal(t, SlotMorning, occ[0].Item.Slot)
	assert.Equal(t, at("2026-10-06 08:00"), occ[0].At)
	assert.False(t, occ[0].Decision.Send, "vitals reminders are opt-in")
	assert.Equal(t, notifications.ReasonCategoryOff, occ[0].Decision.Reason)
	assert.Equal(t, "Time to measure your blood pressure", occ[0].Message.Title)
	assert.Equal(t, "/vitals", occ[0].Message.URL)

	on := prefs
	on.Categories = map[notifications.Category]bool{notifications.Vitals: true}
	assert.True(t, PlanReminders(on, items, nil, today, "fa")[0].Decision.Send)
}

func TestAlertCopy_FallbackAndOverride(t *testing.T) {
	r := Reading{Type: TypeBP, Systolic: 185, Diastolic: 110}
	a := decode(t, AlertCopy{}.AlertJSON(r, "fa", "fa"))
	assert.Equal(t, "bp_crisis", a["rule"])
	assert.Equal(t, "urgent", a["level"])
	assert.Equal(t, true, a["modal"])
	assert.Contains(t, a["what_we_saw"], "۱۸۵/۱۱۰ mmHg")
	actions := a["actions"].([]any)
	require.Len(t, actions, 2)
	assert.Equal(t, "call", actions[0].(map[string]any)["key"])
	assert.Equal(t, "115", actions[0].(map[string]any)["phone"])

	cp := AlertCopy{RuleGlucoseLow: {"en": {"title": "Low!", "what_we_saw": "Saw {value}",
		"actions": []any{map[string]any{"key": "ack", "label": "OK"}}}}}
	g := Reading{Type: TypeGlucose, MgDl: MgDlFrom(2.8, UnitMmolL), Unit: UnitMmolL}
	b := decode(t, cp.AlertJSON(g, "de", "en"))
	assert.Equal(t, "Low!", b["title"], "unknown locale → default-language row")
	assert.Equal(t, "Saw 2.8 mmol/L", b["what_we_saw"])
	assert.Nil(t, b["advice"])
	assert.Nil(t, AlertCopy{}.AlertJSON(Reading{Type: TypeBP, Systolic: 120, Diastolic: 80}, "en", "en"))
}

func body(kv ...any) phpval.Map {
	m := phpval.NewMap()
	for i := 0; i < len(kv); i += 2 {
		m.Set(kv[i].(string), kv[i+1])
	}
	return m
}

func TestValidateReading(t *testing.T) {
	now := at("2026-10-06 09:00")
	in, err := ValidateReading(body("type", "glucose", "value", "5.2", "unit", "mmol_l", "context", "fasting"), "en", now)
	require.NoError(t, err)
	assert.InDelta(t, 93.6, in.MgDl, 1e-9)
	assert.Equal(t, now, in.MeasuredAt)

	_, err = ValidateReading(body("type", "glucose", "value", "40", "unit", "mmol_l", "context", "fasting"), "en", now)
	require.Error(t, err)
	_, err = ValidateReading(body("type", "bp", "systolic", 80, "diastolic", 90), "en", now)
	require.Error(t, err, "systolic must exceed diastolic")
	_, err = ValidateReading(body("type", "hr", "bpm", 72, "context", "resting", "measured_at", "2026-10-06 09:30"), "en", now)
	require.Error(t, err, "future time")
	_, err = ValidateReading(body("type", "weight"), "en", now)
	require.Error(t, err)
	in, err = ValidateReading(body("type", "bp", "systolic", "118", "diastolic", "76", "arm", "left", "measured_at", "2026-10-06 08:10"), "en", now)
	require.NoError(t, err)
	assert.Equal(t, at("2026-10-06 08:10"), in.MeasuredAt)
}

func TestValidatePlan(t *testing.T) {
	now := at("2026-10-06 09:00")
	items, err := ValidatePlan(body("items", []any{
		body("type", "bp", "slot", "morning", "days", []any{0, 1, 2, 3, 4, 5, 6}, "remind_at", "08:00"),
		body("type", "glucose", "slot", "fasting", "days", []any{3}),
	}), "en", now)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, AllDays, items[0].Days)
	assert.Equal(t, uint8(1<<3), items[1].Days)

	_, err = ValidatePlan(body("items", []any{body("type", "glucose", "slot", "morning", "days", []any{1})}), "en", now)
	require.Error(t, err, "slot of another type")
	_, err = ValidatePlan(body("items", []any{
		body("type", "bp", "slot", "morning", "days", []any{1}),
		body("type", "bp", "slot", "morning", "days", []any{2}),
	}), "en", now)
	require.Error(t, err, "duplicate slot")
	items, err = ValidatePlan(body("items", []any{}), "en", now)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestLangFilesLoad(t *testing.T) {
	assert.Equal(t, "Saved", T("messages.saved", "en"))
	assert.Equal(t, "ثبت شد", T("messages.saved", "fa"))
	assert.NotEmpty(t, attributes("fa"))
	assert.Equal(t, "۵٫۲ mmol/L", Digits("5.2 mmol/L", "fa"))
}
