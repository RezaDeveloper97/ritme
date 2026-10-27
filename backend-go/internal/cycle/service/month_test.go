package service

// T-M9-33 (D-121): the v6 month payload from engine states built in memory (no DB): cycle boundaries, the O+1 fix,
// month lengths, lateness (E-04), late beyond max_late, hormonal contraception (E-13), the declared onboarding range
// (I-01), pregnancy (E-25), completed cycles (E-22) and the month shadow classes.

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/recommendation"
	"github.com/ritme/backend-go/internal/cycle/shadow"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

type noTips struct{}

func (noTips) ActiveRows(context.Context) ([]recommendation.Row, error) { return nil, nil }
func (noTips) AnyExists(context.Context) (bool, error)                  { return false, nil }

type monthDayJSON struct {
	Date         string  `json:"calculation_date"`
	CycleDay     *int    `json:"cycle_day"`
	Phase        *string `json:"phase"`
	Subphase     *string `json:"subphase"`
	Fertile      bool    `json:"is_fertile_window"`
	PMS          bool    `json:"is_pms_window"`
	Tomorrow     bool    `json:"is_period_tomorrow"`
	MainPhase    string  `json:"main_phase"`
	Fertility    string  `json:"fertility_level"`
	IsPredicted  bool    `json:"is_predicted"`
	IsPeriod     bool    `json:"is_period"`
	PeriodKind   *string `json:"period_kind"`
	DaysToPeriod *int    `json:"days_to_period"`
	DaysLate     *int    `json:"days_late"`
}

type monthJSONBody struct {
	Success bool `json:"success"`
	Data    struct {
		Calculations []monthDayJSON `json:"calculations"`
		Status       string         `json:"calculation_status"`
		Summary      struct {
			Fertile int `json:"fertile_days"`
			Period  int `json:"period_days"`
			PMS     int `json:"pms_days"`
		} `json:"month_summary"`
		Future []monthFutureCycle `json:"future_cycles"`
		Limits monthLimits        `json:"limits"`
	} `json:"data"`
}

func d(s string) engine.Day { return engine.DayOf(civildate.MustParse(s)) }

// periods are closed 5-day periods starting on each date.
func periods(starts ...string) []engine.Period {
	out := make([]engine.Period, len(starts))
	for i, s := range starts {
		out[i] = engine.Period{Start: d(s), End: d(s) + 4}
	}
	return out
}

func monthOf(t *testing.T, in engine.Input, year, month int, viewName, today string) (monthJSONBody, map[string]monthDayJSON) {
	t.Helper()
	e := &Engine{tips: noTips{}}
	st := engine.Prepare(in, engine.DefaultParams())
	raw, err := e.monthJSON(context.Background(), st, year, month, viewName, civildate.MustParse(today), "en")
	require.NoError(t, err)
	var b monthJSONBody
	require.NoError(t, json.Unmarshal(raw, &b), string(raw))
	require.True(t, b.Success)
	byDate := map[string]monthDayJSON{}
	for _, c := range b.Data.Calculations {
		byDate[c.Date] = c
	}
	return b, byDate
}

func kind(c monthDayJSON) string {
	if c.PeriodKind == nil {
		return ""
	}
	return *c.PeriodKind
}

// A completed July (28-day cycles from 06-10, 07-08, 08-05; today 08-20): the cycle boundary restarts the day count,
// the retrospective layout puts ovulation on next − 14 = 07-22 (E-22), the window is 07-17..07-22 and O+1 is not
// fertile; no period_expected day exists inside the completed cycle.
func TestMonthV6_CycleBoundaryRetrospectiveAndOPlusOne(t *testing.T) {
	in := engine.Input{Periods: periods("2026-06-10", "2026-07-08", "2026-08-05")}
	b, days := monthOf(t, in, 2026, 7, "full", "2026-08-20")
	require.Len(t, b.Data.Calculations, 31, "July has 31 days")
	assert.Equal(t, 22, *days["2026-07-01"].CycleDay)
	assert.Equal(t, 1, *days["2026-07-08"].CycleDay)
	for _, s := range []string{"2026-07-08", "2026-07-12"} {
		assert.True(t, days[s].IsPeriod, s)
		assert.Equal(t, "logged", kind(days[s]), s)
		assert.False(t, days[s].IsPredicted, s)
	}
	assert.False(t, days["2026-07-13"].IsPeriod)
	for day := 17; day <= 22; day++ {
		s := civildate.New(2026, time.July, day).String()
		assert.True(t, days[s].Fertile, s)
		assert.Equal(t, "fertile", days[s].MainPhase, s)
	}
	assert.False(t, days["2026-07-16"].Fertile)
	assert.False(t, days["2026-07-23"].Fertile, "O+1 is not fertile (D-30)")
	assert.Equal(t, "luteal", days["2026-07-23"].MainPhase)
	for _, c := range b.Data.Calculations {
		assert.NotEqual(t, "period_expected", c.MainPhase, c.Date)
	}
	assert.Equal(t, 6, b.Data.Summary.Fertile, "07-17..07-22 (June's window is 06-19..06-24)")
	assert.Equal(t, 5, b.Data.Summary.Period)
	assert.Equal(t, "completed", b.Data.Status)
	// Full view keeps the full-only keys; the calendar view drops them.
	assert.Contains(t, rawDay(t, in, "full"), "daily_tips")
	assert.NotContains(t, rawDay(t, in, "calendar"), "daily_tips")
	assert.Contains(t, rawDay(t, in, "calendar"), "final_probability")
}

func rawDay(t *testing.T, in engine.Input, viewName string) map[string]any {
	t.Helper()
	e := &Engine{tips: noTips{}}
	raw, err := e.monthJSON(context.Background(), engine.Prepare(in, engine.DefaultParams()), 2026, 7, viewName,
		civildate.MustParse("2026-08-20"), "en")
	require.NoError(t, err)
	var b struct {
		Data struct {
			Calculations []map[string]any `json:"calculations"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &b))
	return b.Data.Calculations[10]
}

func TestMonthV6_MonthLengths(t *testing.T) {
	in := engine.Input{Periods: periods("2026-01-05")}
	b, _ := monthOf(t, in, 2026, 2, "calendar", "2026-01-10")
	assert.Len(t, b.Data.Calculations, 28)
	b, _ = monthOf(t, in, 2028, 2, "calendar", "2026-01-10")
	assert.Len(t, b.Data.Calculations, 29)
}

// 8 days late (next 09-23, today 10-01): 09-23..10-01 are period_expected with days_late counting up and the cycle
// day not wrapping; from tomorrow the rebased chain predicts a period 10-02..10-06 (E-04).
func TestMonthV6_LateUser(t *testing.T) {
	in := engine.Input{Periods: periods("2026-07-01", "2026-07-29", "2026-08-26")}
	b, sep := monthOf(t, in, 2026, 9, "calendar", "2026-10-01")
	assert.Equal(t, "luteal", sep["2026-09-22"].MainPhase)
	for day := 23; day <= 30; day++ {
		s := civildate.New(2026, time.September, day).String()
		assert.Equal(t, "period_expected", sep[s].MainPhase, s)
		assert.Equal(t, day-23, *sep[s].DaysLate, s)
		assert.Equal(t, day+6, *sep[s].CycleDay, "%s: no wrap, the cycle keeps counting", s)
		assert.False(t, sep[s].IsPeriod, s)
	}
	assert.Equal(t, "2026-10-02", *b.Data.Future[0].PeriodStart)

	b, oct := monthOf(t, in, 2026, 10, "calendar", "2026-10-01")
	assert.Equal(t, "period_expected", oct["2026-10-01"].MainPhase)
	assert.Equal(t, 8, *oct["2026-10-01"].DaysLate)
	assert.Equal(t, 37, *oct["2026-10-01"].CycleDay)
	for day := 2; day <= 6; day++ {
		s := civildate.New(2026, time.October, day).String()
		assert.True(t, oct[s].IsPeriod, s)
		assert.Equal(t, "predicted", kind(oct[s]), s)
		assert.True(t, oct[s].IsPredicted, s)
	}
	assert.Len(t, b.Data.Future, 6)
	assert.Equal(t, "2026-10-02", *b.Data.Future[0].PeriodStart)
}

// 15 days late (> max_late 14): the list is empty and nothing after today is painted (E-04).
func TestMonthV6_LateBeyondMaxLate(t *testing.T) {
	in := engine.Input{Periods: periods("2026-07-01", "2026-07-29", "2026-08-26")}
	b, oct := monthOf(t, in, 2026, 10, "calendar", "2026-10-08")
	assert.Empty(t, b.Data.Future)
	for day := 9; day <= 31; day++ {
		s := civildate.New(2026, time.October, day).String()
		assert.False(t, oct[s].IsPeriod, s)
		assert.False(t, oct[s].Fertile, s)
		assert.Equal(t, "unknown", oct[s].MainPhase, s)
	}
	b, nov := monthOf(t, in, 2026, 11, "calendar", "2026-10-08")
	assert.Zero(t, b.Data.Summary.Period)
	assert.Nil(t, nov["2026-11-10"].CycleDay)
}

// E-28: nothing is predicted beyond future_cycles[6].period_end.
func TestMonthV6_NothingBeyondSixthCycle(t *testing.T) {
	in := engine.Input{Periods: periods("2026-09-01")}
	b, _ := monthOf(t, in, 2026, 9, "calendar", "2026-09-10")
	require.Len(t, b.Data.Future, 6)
	assert.Equal(t, "2027-02-20", *b.Data.Future[5].PeriodEnd) // 09-29 + 5·28 = 02-16, +4
	_, feb := monthOf(t, in, 2027, 2, "calendar", "2026-09-10")
	assert.True(t, feb["2027-02-20"].IsPeriod)
	assert.Equal(t, "predicted", kind(feb["2027-02-20"]))
	assert.False(t, feb["2027-02-21"].IsPeriod)
	assert.Equal(t, "unknown", feb["2027-02-25"].MainPhase)
	b, _ = monthOf(t, in, 2027, 3, "calendar", "2026-09-10")
	for _, c := range b.Data.Calculations {
		assert.Equal(t, "unknown", c.MainPhase, c.Date)
		assert.False(t, c.Fertile || c.IsPeriod, c.Date)
	}
}

// E-13: hormonal contraception has no fertile day and no fertile/ovulation dates in future_cycles.
func TestMonthV6_PillHasNoFertileDays(t *testing.T) {
	in := engine.Input{Periods: periods("2026-08-03", "2026-08-31"), Hormonal: true}
	for _, m := range []int{8, 9, 10} {
		b, _ := monthOf(t, in, 2026, m, "calendar", "2026-09-10")
		assert.Zero(t, b.Data.Summary.Fertile, "month %d", m)
		for _, c := range b.Data.Calculations {
			assert.False(t, c.Fertile, c.Date)
			assert.NotEqual(t, "fertile", c.MainPhase, c.Date)
		}
		assert.True(t, b.Data.Limits.FertilitySuppressed)
		assert.Equal(t, "hormonal_contraception", *b.Data.Limits.SuppressionReason)
		for _, f := range b.Data.Future {
			assert.Nil(t, f.FertileStart)
			assert.Nil(t, f.Ovulation)
			assert.NotNil(t, f.PeriodStart)
		}
	}
}

// I-01: the declared onboarding period is an assumed range (never logged).
func TestMonthV6_OnboardingOnlyIsAssumed(t *testing.T) {
	in := engine.Input{Declared: engine.Declared{LastPeriodStart: engine.Some(d("2026-09-10")),
		CycleLength: engine.Some(28), PeriodLength: engine.Some(5)}}
	_, sep := monthOf(t, in, 2026, 9, "calendar", "2026-09-20")
	for day := 10; day <= 14; day++ {
		s := civildate.New(2026, time.September, day).String()
		assert.True(t, sep[s].IsPeriod, s)
		assert.Equal(t, "assumed", kind(sep[s]), s)
	}
	assert.Nil(t, sep["2026-09-09"].CycleDay, "before the declared start")
	_, oct := monthOf(t, in, 2026, 10, "calendar", "2026-09-20")
	assert.Equal(t, "predicted", kind(oct["2026-10-08"]))
}

// An open period: the logged start is logged, the following assumed days are assumed.
func TestMonthV6_OpenPeriodAssumedDays(t *testing.T) {
	in := engine.Input{Periods: append(periods("2026-08-20"), engine.Period{Start: d("2026-09-17"), End: engine.NoDay})}
	_, sep := monthOf(t, in, 2026, 9, "calendar", "2026-09-19")
	assert.Equal(t, "logged", kind(sep["2026-09-17"]))
	assert.Equal(t, "assumed", kind(sep["2026-09-18"]))
	assert.Equal(t, "assumed", kind(sep["2026-09-21"]))
	assert.Equal(t, "assumed", kind(sep["2026-09-28"]), "menstrual_possible up to the hard cap (day 12)")
	assert.False(t, sep["2026-09-29"].IsPeriod)
}

// E-25: pregnancy shows the logged history only; no predicted day, no future cycles, the engine suspended.
func TestMonthV6_PregnancyLoggedHistoryOnly(t *testing.T) {
	in := engine.Input{Periods: periods("2026-07-01", "2026-07-29"), Mode: engine.ModePregnancy}
	for _, m := range []int{7, 8, 9} {
		b, _ := monthOf(t, in, 2026, m, "full", "2026-09-10")
		assert.Empty(t, b.Data.Future)
		assert.True(t, b.Data.Limits.EngineSuspended)
		for _, c := range b.Data.Calculations {
			if c.IsPeriod {
				assert.Equal(t, "logged", kind(c), c.Date)
				continue
			}
			assert.Equal(t, "unknown", c.MainPhase, c.Date)
			assert.False(t, c.Fertile, c.Date)
			assert.Nil(t, c.CycleDay, c.Date)
		}
		assert.Equal(t, map[int]int{7: 8, 8: 2, 9: 0}[m], b.Data.Summary.Period, "month %d", m)
	}
}

// The month shadow classes: O+1, lateness (the legacy cycle day wraps), suppression, agreement.
func TestClassifyMonthDay(t *testing.T) {
	st := engine.Prepare(engine.Input{Periods: periods("2026-07-01", "2026-07-29", "2026-08-26")}, engine.DefaultParams())
	today := d("2026-10-01")
	i := func(n int) *int { return &n }
	s := func(v string) *string { return &v }

	o1 := st.Resolve(d("2026-09-10"), today) // next 09-23 → O 09-09, O+1 09-10
	require.Equal(t, engine.MainPhaseLuteal, o1.MainPhase)
	assert.Contains(t, ClassifyMonthDay(&v1MonthDay{CycleDay: i(16), OvDay: i(15), Fertile: true, Phase: s("ovulation")}, &o1),
		shadow.ClassOPlusOne)

	late := st.Resolve(d("2026-09-30"), today)
	assert.Contains(t, ClassifyMonthDay(&v1MonthDay{CycleDay: i(8), Phase: s("follicular")}, &late), shadow.ClassLateness)

	same := st.Resolve(d("2026-09-15"), today)
	cd, _ := same.CycleDay.Get()
	assert.Equal(t, []shadow.Class{shadow.ClassNewFieldsOnly},
		ClassifyMonthDay(&v1MonthDay{CycleDay: &cd, Phase: s("luteal")}, &same))

	pill := engine.Prepare(engine.Input{Periods: periods("2026-08-26"), Hormonal: true}, engine.DefaultParams())
	sd := pill.Resolve(d("2026-09-08"), d("2026-09-10"))
	assert.Contains(t, ClassifyMonthDay(&v1MonthDay{CycleDay: i(14), Fertile: true, Phase: s("ovulation")}, &sd),
		shadow.ClassSuppression)
}
