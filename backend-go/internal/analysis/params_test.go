package analysis

// T-M10-26 (D-175, A-02, A-12): the §4.1 minimums and methods at their boundaries.

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/fertility/bbt"
	"github.com/ritme/backend-go/internal/labs"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func TestParams_SpecValues(t *testing.T) {
	assert.Equal(t, Params{
		CycleLengthMinCycles: 2, RegularityMinCycles: 3, RegularityMonths: 12,
		PeriodIntensityMinPeriods: 2, SymptomPatternMinCycles: 3, MoodPhaseMinDays: 10, MoodPhaseMinPhases: 2,
		CorrMinPairedDays: 30, CorrStrongDays: 30, CorrStrongRatio: 2, CorrMediumRatio: 1.5, CorrWeakRatio: 1.2,
		CorrMinGroupDays: 5, CorrMinOutcome: 3,
		WeightMinLogs: 3, WeightWindowDays: 14, VitalsMinLogs: 2, VitalsMinDays: 2,
		LabTrendMinReports: 2, BBTMinTemps: 10, PregWeightMinLogs: 2, EPDSMinScores: 1, EPDSTrendMinScores: 2,
		MonthlyMinDays: 10, TopFindingHalfLifeDays: 30,
	}, P)
}

func TestCorrelation_PairedDaysBoundary(t *testing.T) {
	c := Binary(CorrSleepMood, Group{Key: "ok", Days: 15, Hits: 3}, Group{Key: "short", Days: 14, Hits: 7})
	assert.Equal(t, CorrNotEnoughData, c.Status, "29 paired days")
	assert.Equal(t, 1, c.DaysNeeded)
	assert.Empty(t, c.Strength)

	c = Binary(CorrSleepMood, Group{Key: "ok", Days: 20, Hits: 5}, Group{Key: "short", Days: 10, Hits: 5})
	assert.Equal(t, CorrReady, c.Status, "30 paired days")
	assert.Equal(t, 0, c.DaysNeeded)
	assert.Equal(t, StrengthStrong, c.Strength, "ratio 2.0 at 30 days")
	require.NotNil(t, c.Ratio)
	assert.InDelta(t, 2.0, *c.Ratio, 1e-9)
	assert.NotZero(t, c.Value, "φ kept as the extra value")
}

func TestCorrelation_RatioBoundary(t *testing.T) {
	c := Binary(CorrSleepMood, Group{Key: "ok", Days: 20, Hits: 5}, Group{Key: "short", Days: 40, Hits: 19})
	assert.Equal(t, CorrReady, c.Status)
	assert.Equal(t, StrengthMedium, c.Strength, "ratio 1.9")

	assert.Equal(t, StrengthStrong, RatioStrength(0.5, 0.25, 30))
	assert.Equal(t, StrengthMedium, RatioStrength(0.475, 0.25, 30))
	assert.Equal(t, StrengthStrong, RatioStrength(0.25, 0.5, 30), "the difference in either direction")
	assert.Equal(t, StrengthMedium, RatioStrength(0.5, 0.25, 29), "strong needs 30 days")
	assert.Equal(t, StrengthWeak, RatioStrength(0.3, 0.25, 30))
	assert.Empty(t, RatioStrength(0.29, 0.25, 30))
	assert.Equal(t, StrengthStrong, RatioStrength(0.2, 0, 40), "an empty-rate group")

	c = Binary(CorrSleepMood, Group{Key: "ok", Days: 20, Hits: 5}, Group{Key: "short", Days: 20, Hits: 5})
	assert.Equal(t, CorrNoAssociation, c.Status)
}

// Binary is one rule for the analysis tab and the menopause patterns (T-M10-26b): the old φ rule's 20 days and
// Cohen bands are gone.
func TestBinary_SharedRuleForMenopause(t *testing.T) {
	c := Binary("x", Group{Key: "ok", Days: 10, Hits: 1}, Group{Key: "short", Days: 10, Hits: 5})
	assert.Equal(t, CorrNotEnoughData, c.Status, "20 days was enough for the old φ cards")
	assert.Equal(t, 10, c.DaysNeeded)
	// φ ≈ 0.12 (old: weak) but a 1.5× rate difference over 40 days → medium.
	c = Binary("x", Group{Key: "ok", Days: 20, Hits: 6}, Group{Key: "short", Days: 20, Hits: 9})
	assert.Equal(t, CorrReady, c.Status)
	assert.Equal(t, StrengthMedium, c.Strength)
}

func moodInput(perPhase map[string]int) (*Input, map[civildate.Date]string) {
	in := &Input{Today: fixtureToday, Days: map[civildate.Date]*Day{}}
	in.Range = NewRange(Range6M, fixtureToday)
	phases := map[civildate.Date]string{}
	d := fixtureToday
	for _, p := range Phases {
		for range perPhase[p] {
			in.Days[d] = &Day{Moods: []string{"happy"}, Symptoms: map[string]float64{}}
			phases[d] = p
			d = d.AddDays(-1)
		}
	}
	return in, phases
}

func TestMoodByPhase_TenDaysPerPhase(t *testing.T) {
	in, ph := moodInput(map[string]int{PhasePeriod: 10, PhaseLuteal: 9})
	m := in.moodByPhase(ph)
	assert.True(t, m.Phases[0].Ready())
	assert.False(t, m.Phases[3].Ready(), "9 days")
	assert.False(t, m.Ready(), "one qualifying phase")
	assert.Equal(t, 1, m.DaysNeeded)
	assert.Empty(t, m.Phases[3].Shares)

	in, ph = moodInput(map[string]int{PhasePeriod: 10, PhaseLuteal: 10})
	m = in.moodByPhase(ph)
	assert.True(t, m.Ready())
	assert.Equal(t, []MoodShare{{Mood: "happy", Days: 10}}, m.Phases[3].Shares)
	assert.Equal(t, 100, m.Phases[3].pct(m.Phases[3].Shares[0].Days))
}

func weightInput(daysBack ...int) *Input {
	in := &Input{Today: fixtureToday, Days: map[civildate.Date]*Day{}}
	in.Range = NewRange(Range1M, fixtureToday)
	for _, b := range daysBack {
		w := 60.0
		in.Days[fixtureToday.AddDays(-b)] = &Day{Weight: &w, Symptoms: map[string]float64{}}
	}
	return in
}

func TestWeight_ThreeLogsInFourteenDays(t *testing.T) {
	r := BuildBody(weightInput(0, 5))
	assert.False(t, r.WeightReady(), "2 logs in 14 days")
	assert.Equal(t, 1, r.WeightDaysNeeded())
	assert.Nil(t, r.Delta7)
	assert.Nil(t, r.Current)

	r = BuildBody(weightInput(0, 5, 13))
	assert.True(t, r.WeightReady(), "3 logs in 14 days")
	assert.NotNil(t, r.Current)

	r = BuildBody(weightInput(0, 7, 14))
	assert.False(t, r.WeightReady(), "3 logs over 15 days")
}

func TestVitals_TwoLogsOnTwoDays(t *testing.T) {
	in := &Input{Today: fixtureToday, VitalsLoaded: true, VitalReadings: []VitalReading{
		{Date: fixtureToday, Systolic: 120, Diastolic: 80, Class: "normal"},
		{Date: fixtureToday, Systolic: 130, Diastolic: 85, Class: "stage_1"},
	}}
	v := in.vitals(fixtureToday.AddDays(-30), fixtureToday)
	assert.False(t, v.BP.Ready(), "2 logs on 1 day")
	assert.Equal(t, 1, v.BP.ReadingsNeeded())

	in.VitalReadings[1].Date = fixtureToday.AddDays(-1)
	v = in.vitals(fixtureToday.AddDays(-30), fixtureToday)
	require.True(t, v.BP.Ready(), "2 logs on 2 days")
	assert.Equal(t, []float64{125, 83}, v.BP.Mean)
	assert.Equal(t, []float64{120, 80}, v.BP.Min)
	assert.Equal(t, []float64{130, 85}, v.BP.Max)
	assert.Equal(t, map[string]int{"normal": 1, "stage_1": 1}, v.BP.Classes)
}

func TestBBT_TenTemperatures(t *testing.T) {
	report := func(n int) TTCReport {
		pts := make([]bbt.Point, n)
		for i := range pts {
			pts[i] = bbt.Point{CycleDay: i + 1, Value: 3640}
		}
		return TTCReport{Cycles: []TTCCycle{{Cycle: Cycle{Start: fixtureToday.AddDays(-n), Current: true}, BBT: bbt.Cycle{Points: pts}}}}
	}
	assert.False(t, report(9).bbtCard().Ready, "9 temperatures")
	assert.True(t, report(10).bbtCard().Ready, "10 temperatures")
	r := report(9)
	assert.Equal(t, "ttc.fertility.how.few", explanation(&r.Cycles[0]).Key)
}

func TestMonthly_TenDays(t *testing.T) {
	r := MonthlyReport{This: MonthMetrics{DaysLogged: 9}}
	assert.False(t, r.Ready())
	h, _, _ := r.copy()
	assert.Equal(t, "monthly.headline.little_data", h.Key)
	r.This.DaysLogged = 10
	assert.True(t, r.Ready())
	h, _, _ = r.copy()
	assert.NotEqual(t, "monthly.headline.little_data", h.Key)
}

func TestLabTrend_TwoReports(t *testing.T) {
	one := LabSeries{Key: "ferritin", Points: []labs.Point{{Date: fixtureToday, Value: 30}}}
	assert.False(t, one.Ready())
	assert.Equal(t, "labs.trend.not_enough", one.phrase().Key)
	two := NewLabSeries(labs.Series{Key: "ferritin", Points: []labs.Point{{Date: fixtureToday.AddDays(-90), Value: 30}, {Date: fixtureToday, Value: 12}}}, "fa", "fa")
	assert.True(t, two.Ready())
	assert.Equal(t, labs.TrendFalling, two.Direction)
	assert.Equal(t, "labs.trend.falling", two.phrase().Key)

	ready, body := LabTrendBody([]LabSeries{one, two}, "", nil)
	assert.True(t, ready)
	assert.NotNil(t, body)
	ready, _ = LabTrendBody([]LabSeries{one, two}, "ferritin", nil)
	assert.False(t, ready, "the first series of that key: 1 report")
	ready, _ = LabTrendBody(nil, "unknown", nil)
	assert.False(t, ready)
}

// Regularity counts the 3 cycles within 12 months of the report day.
func TestRegularity_TwelveMonths(t *testing.T) {
	build := func(current string) CycleReport {
		hist, _ := history(civildate.MustParse(current), []int{29, 29, 29}, func(int) int { return 5 })
		in := &Input{Today: fixtureToday, Histories: hist, Profile: &model.Profile{}, Days: map[civildate.Date]*Day{}}
		in.Range = NewRange(RangeAll, fixtureToday)
		return BuildCycle(in)
	}
	inside := build("2026-09-13")
	assert.Equal(t, 3, inside.BasedOn)
	assert.Equal(t, RegularityRegular, inside.Regularity, "3 cycles inside 12 months")

	outside := build("2025-09-01") // the 3 completed cycles started in June–August 2025
	assert.Equal(t, 3, outside.BasedOn)
	assert.Equal(t, RegularityNotEnoughData, outside.Regularity, "3 cycles outside 12 months")
}

// An older strong finding vs a recent medium one: score = strength × 0.5^(age/30).
func TestTopFinding_StrengthTimesRecency(t *testing.T) {
	old := Candidate{Kind: FindingCycleIrregular, Strength: 0.8, Evidence: fixtureToday.AddDays(-60)} // 0.8 × 0.25 = 0.2
	recent := Candidate{Kind: FindingCorrelation, Strength: 0.5, Evidence: fixtureToday.AddDays(-2)}  // 0.5 × 0.955 ≈ 0.477
	got, ok := PickTopFinding([]Candidate{old, recent}, fixtureToday)
	require.True(t, ok)
	assert.Equal(t, FindingCorrelation, got.Kind)
	assert.InDelta(t, 0.2, old.Score(fixtureToday), 1e-9)

	old.Evidence = fixtureToday.AddDays(-10) // 0.8 × 0.794 ≈ 0.635
	got, _ = PickTopFinding([]Candidate{recent, old}, fixtureToday)
	assert.Equal(t, FindingCycleIrregular, got.Kind)

	// ties: the fixed order
	a := Candidate{Kind: FindingCycleRegular, Strength: 0.5, Evidence: fixtureToday}
	b := Candidate{Kind: FindingPattern, Strength: 0.5, Evidence: fixtureToday}
	got, _ = PickTopFinding([]Candidate{a, b}, fixtureToday)
	assert.Equal(t, FindingPattern, got.Kind)

	_, ok = PickTopFinding(nil, fixtureToday)
	assert.False(t, ok)
}

// A correlation with 18 paired days is the nearest unmet minimum: «۱۲ روز دیگر».
func TestNudge_CorrelationDays(t *testing.T) {
	in := ranged(emptyFixture(), Range6M)
	corr := &CorrelationsReport{Items: []Correlation{
		Binary(CorrSleepMood, Group{Key: "sleep_6_plus", Days: 9, Hits: 2}, Group{Key: "sleep_under_6", Days: 9, Hits: 3}),
	}}
	assert.Equal(t, 12, corr.Items[0].DaysNeeded)
	f := BuildTopFinding(in, CycleReport{}, SymptomsReport{}, corr)
	assert.Equal(t, FindingNotEnoughData, f.Kind)
	assert.Equal(t, 12, f.DaysNeeded)
	assert.Equal(t, []Arg{Num("days", 12)}, f.Parts[0].Args)

	// Without Plus the nearest is the cycle-length minimum: 2 cycles × the effective length.
	cycleLen, _ := in.typical()
	f = BuildTopFinding(in, CycleReport{}, SymptomsReport{}, nil)
	assert.Equal(t, 2*cycleLen, f.DaysNeeded)
}

// The rendered nudge and the not-causal line (the main session reviews these strings).
func TestCopy_SpecStrings(t *testing.T) {
	fa := copyFor(t, "fa")
	assert.Equal(t, "برای دیدن الگوها ۱۲ روز دیگر ثبت کن.", fa.Render(Phrase{Key: "findings.not_enough_days", Args: []Arg{Num("days", 12)}}))
	assert.Equal(t, "هم\u200cزمانی در داده\u200cهای تو، نه علت", fa.Render(NotCausalPhrase()))
}
