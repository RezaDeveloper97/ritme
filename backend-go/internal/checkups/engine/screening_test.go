package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cc "github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// T-M10-21 (SPEC v6 §3.4, D-172): registry-driven intervals from screening.uspstf version 1.

var (
	bpType = Type{ID: 21, Key: "bp_screening", Category: CategoryAnnual, IntervalMonths: 12, RemindLeadDays: 14,
		IsActive: true, SortOrder: 21}
	glucoseType = Type{ID: 22, Key: "glucose_screening", Category: CategoryMultiYear, IntervalMonths: 36,
		RemindLeadDays: 14, IsActive: true, SortOrder: 22}
)

// bornAged is a birthday that makes the user `age` on today (2026-09-23).
func bornAged(age int) civildate.Date { return civildate.New(2026-age, 1, 1) }

func ageOf(n int) *int { return &n }

// evalOne evaluates one type for a user of age with the given latest record kind (no record when kind is "-").
func evalOne(t *testing.T, typ Type, age int, bmi float64, last *Record) (Item, bool) {
	t.Helper()
	in := Input{Types: []Type{typ}, Birthday: bornAged(age), BMI: bmi}
	if last != nil {
		in.Records = []Record{*last}
	}
	res := Evaluate(in, at(today))
	if len(res.Items) == 0 {
		return Item{}, false
	}
	require.Len(t, res.Items, 1)
	return res.Items[0], true
}

func TestScreening_Cervical(t *testing.T) {
	cyto := &Record{ID: 1, TypeID: 3, DoneOn: d("2025-09-23"), Result: "normal", TestKind: "cytology"}
	hpv := &Record{ID: 1, TypeID: 3, DoneOn: d("2025-09-23"), Result: "normal", TestKind: "hpv"}
	cotest := &Record{ID: 1, TypeID: 3, DoneOn: d("2025-09-23"), Result: "normal", TestKind: "cotest"}

	it, ok := evalOne(t, pap, 20, 0, nil)
	require.True(t, ok)
	assert.Equal(t, StatusNotYet, it.Status, "20: not screened yet")
	assert.Equal(t, 21, it.Schedule.AgeMin)

	it, ok = evalOne(t, pap, 21, 0, nil)
	require.True(t, ok)
	assert.Equal(t, StatusDue, it.Status, "21 with no record: due (the cytology band counts without a last record)")
	assert.Equal(t, 36, it.Schedule.IntervalMonths)
	assert.True(t, it.Registry)
	assert.Equal(t, []string{"cytology"}, it.TestKinds)
	assert.False(t, it.TestKindRequired, "21–29 has one kind only")

	it, _ = evalOne(t, pap, 29, 0, cyto)
	assert.Equal(t, 36, it.Schedule.IntervalMonths, "29 cytology")
	assert.Equal(t, d("2028-09-23"), it.NextDueOn)

	it, _ = evalOne(t, pap, 30, 0, cyto)
	assert.Equal(t, 36, it.Schedule.IntervalMonths, "30 last cytology")
	assert.Equal(t, []string{"cytology", "hpv", "cotest"}, it.TestKinds)
	assert.True(t, it.TestKindRequired, "from 30 the kind changes the interval")

	it, _ = evalOne(t, pap, 30, 0, hpv)
	assert.Equal(t, 60, it.Schedule.IntervalMonths, "30 last HPV")
	assert.Equal(t, d("2030-09-23"), it.NextDueOn)

	it, _ = evalOne(t, pap, 40, 0, cotest)
	assert.Equal(t, 60, it.Schedule.IntervalMonths, "co-test")

	it, _ = evalOne(t, pap, 40, 0, &Record{ID: 1, TypeID: 3, DoneOn: d("2025-09-23"), Result: "normal"})
	assert.Equal(t, 36, it.Schedule.IntervalMonths, "a record without a kind takes the shorter interval")

	it, _ = evalOne(t, pap, 25, 0, hpv)
	assert.Equal(t, 36, it.Schedule.IntervalMonths, "HPV before 30 has no band row: the shorter interval")

	it, ok = evalOne(t, pap, 65, 0, hpv)
	require.True(t, ok, "65 included")
	assert.Equal(t, 60, it.Schedule.IntervalMonths)

	_, ok = evalOne(t, pap, 66, 0, hpv)
	assert.False(t, ok, "66 excluded")
}

func TestScreening_Mammography(t *testing.T) {
	it, ok := evalOne(t, mammo, 39, 0, nil)
	require.True(t, ok)
	assert.Equal(t, StatusNotYet, it.Status, "39")
	assert.Equal(t, civildate.New(2027, 1, 1), it.NextDueOn, "the day she turns 40")

	it, ok = evalOne(t, mammo, 40, 0, nil)
	require.True(t, ok)
	assert.Equal(t, StatusDue, it.Status, "40")
	assert.Equal(t, 24, it.Schedule.IntervalMonths)
	assert.Zero(t, it.Schedule.IntervalMonthsMax, "the registry interval is not a range")

	_, ok = evalOne(t, mammo, 74, 0, nil)
	assert.True(t, ok, "74 included")
	_, ok = evalOne(t, mammo, 75, 0, nil)
	assert.False(t, ok, "75 excluded")
}

func TestScreening_Glucose(t *testing.T) {
	_, ok := evalOne(t, glucoseType, 35, 24.9, nil)
	assert.False(t, ok, "35 with BMI 24.9: not applicable")

	it, ok := evalOne(t, glucoseType, 35, 25.0, nil)
	require.True(t, ok, "35 with BMI 25.0 (the boundary lands higher)")
	assert.Equal(t, StatusDue, it.Status)
	assert.Equal(t, 36, it.Schedule.IntervalMonths)

	it, ok = evalOne(t, glucoseType, 34, 27, nil)
	require.True(t, ok)
	assert.Equal(t, StatusNotYet, it.Status, "34 overweight: from 35")

	_, ok = evalOne(t, glucoseType, 71, 30, nil)
	assert.False(t, ok, "71 excluded")

	_, ok = evalOne(t, glucoseType, 50, 0, nil)
	assert.True(t, ok, "an unknown BMI does not hide the screening")
}

func TestScreening_BloodPressure(t *testing.T) {
	last := &Record{ID: 1, TypeID: 21, DoneOn: d("2025-09-23"), Result: "normal"}
	it, _ := evalOne(t, bpType, 39, 0, last)
	assert.Equal(t, 36, it.Schedule.IntervalMonths, "39: every 36 months")
	assert.Equal(t, d("2028-09-23"), it.NextDueOn)

	it, _ = evalOne(t, bpType, 40, 0, last)
	assert.Equal(t, 12, it.Schedule.IntervalMonths, "40: every 12 months")
	assert.Equal(t, d("2026-09-23"), it.NextDueOn)

	_, ok := evalOne(t, bpType, 17, 0, nil)
	assert.True(t, ok, "17: not_yet")
}

func TestScreening_UnknownAgeAndOverride(t *testing.T) {
	res := Evaluate(Input{Types: []Type{pap}}, at(today))
	require.Len(t, res.Items, 1)
	assert.Equal(t, 36, res.Items[0].Schedule.IntervalMonths, "no birthday: every band, the shorter interval")
	assert.False(t, res.Items[0].TestKindRequired)

	// A stored version 2 on its effective date replaces the interval (the registry is read on the evaluation day).
	rows := cc.Rows(cc.KeyScreeningUSPSTF, civildate.Date{})
	v2 := make([]cc.Row, 0, len(rows))
	for _, r := range rows {
		c := cc.Row{}
		for k, v := range r {
			c[k] = v
		}
		if c["type_key"] == "dentist" {
			c["interval_months"] = 12.0
		}
		v2 = append(v2, c)
	}
	cc.SetVersions([]cc.Version{{Key: cc.KeyScreeningUSPSTF, Version: 2, Rows: v2, EffectiveFrom: d("2026-09-01")}})
	t.Cleanup(cc.Reset)
	assert.Equal(t, 12, ScheduleFor(dentist, ageOf(30), 0, "", d(today)).Type.IntervalMonths)
	assert.Equal(t, 6, ScheduleFor(dentist, ageOf(30), 0, "", d("2026-08-31")).Type.IntervalMonths)
}

func TestScreening_CustomAndUnlistedKeepTheCatalog(t *testing.T) {
	s := ScheduleFor(blood, ageOf(30), 0, "", d(today))
	assert.False(t, s.Registry)
	assert.Equal(t, blood, s.Type)
	custom := Type{ID: 9, Key: "pap_smear", UserID: 42, Category: CategoryCustom, IntervalMonths: 3, IsActive: true}
	assert.False(t, ScheduleFor(custom, ageOf(30), 0, "", d(today)).Registry)
	assert.Equal(t, []string{"cytology", "hpv", "cotest"}, TestKindsOf("pap_smear", d(today)))
	assert.Empty(t, TestKindsOf("mammography", d(today)))
}

func TestEvaluate_FollowUpInformationalPregnancy(t *testing.T) {
	t.Run("follow_up: no default next due", func(t *testing.T) {
		res := Evaluate(Input{Types: []Type{blood}, Records: []Record{
			{ID: 1, TypeID: 4, DoneOn: d("2026-02-01"), Result: ResultFollowUp},
		}}, at(today))
		it := res.Items[0]
		assert.Equal(t, StatusFollowUp, it.Status)
		assert.True(t, it.NextDueOn.IsZero())
		assert.True(t, it.DueBy.IsZero())
		assert.Equal(t, Summary{Total: 1, Due: 1}, res.Summary, "a follow-up needs action")
	})
	t.Run("follow_up with an explicit next_due_on uses it", func(t *testing.T) {
		it := Evaluate(Input{Types: []Type{blood}, Records: []Record{
			{ID: 1, TypeID: 4, DoneOn: d("2026-02-01"), Result: ResultFollowUp, NextDueOn: d("2027-01-01")},
		}}, at(today)).Items[0]
		assert.Equal(t, StatusUpToDate, it.Status)
		assert.Equal(t, d("2027-01-01"), it.NextDueOn)
	})
	t.Run("informational: no due date, no reminder, not counted", func(t *testing.T) {
		aware := selfExam
		aware.Informational = true
		res := Evaluate(Input{Types: []Type{aware, dentist}, Cycle: cycle28, Records: []Record{rec(1, 1, "2026-01-01")}}, at(today))
		it := find(t, res, 1)
		assert.Equal(t, StatusInformational, it.Status)
		assert.True(t, it.Informational)
		assert.True(t, it.NextDueOn.IsZero())
		assert.False(t, it.Remind)
		assert.Equal(t, Section(CategoryMonthly), it.Section)
		assert.Equal(t, d("2026-01-01"), it.LastDoneOn, "the history stays readable")
		assert.Equal(t, 1, res.Summary.Total)
	})
	t.Run("pregnant: after_pregnancy, no reminder, even when overdue", func(t *testing.T) {
		res := Evaluate(Input{Types: []Type{mammo}, Birthday: born46, Pregnant: true,
			Records: []Record{rec(1, 6, "2023-01-01")}}, at(today))
		require.Len(t, res.Items, 1)
		it := res.Items[0]
		assert.Equal(t, LabelAfterPregnancy, it.Label)
		assert.False(t, it.Remind)
		assert.Equal(t, Section(CategoryAgeBased), it.Section)
		assert.Equal(t, Summary{}, res.Summary)
	})
	t.Run("EvaluateType keeps a type the plan leaves out", func(t *testing.T) {
		it, ok := EvaluateType(mammo, Input{Birthday: d("1940-01-01")}, at(today))
		require.True(t, ok)
		assert.Equal(t, int64(6), int64(it.TypeID))
	})
}

func TestNextDue_EndOfMonthClamp(t *testing.T) {
	monthly := Type{ID: 30, Category: CategoryCustom, IntervalMonths: 1, IsActive: true}
	assert.Equal(t, d("2026-02-28"), NextDueAfter(monthly, d("2026-01-31"), Cycle{}).From)
	it := Evaluate(Input{Types: []Type{monthly}, Records: []Record{rec(1, 30, "2026-01-31")}}, at("2026-02-10")).Items[0]
	assert.Equal(t, d("2026-02-28"), it.NextDueOn)
}
