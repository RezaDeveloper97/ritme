package analysis

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func TestPhi(t *testing.T) {
	assert.InDelta(t, 1, Phi(10, 0, 0, 10), 1e-9)
	assert.InDelta(t, -1, Phi(0, 10, 10, 0), 1e-9)
	assert.InDelta(t, 0, Phi(5, 5, 5, 5), 1e-9)
	assert.InDelta(t, 0, Phi(0, 0, 3, 4), 1e-9, "an empty margin is 0, not NaN")
	// (25·66 − 11·10) / √(36·76·35·77)
	want := (25.0*66 - 11*10) / math.Sqrt(36*76*35*77)
	assert.InDelta(t, want, Phi(25, 11, 10, 66), 1e-12)
}

func TestCramersV(t *testing.T) {
	assert.InDelta(t, 0, CramersV([]int{2, 2, 2, 2}, []int{10, 10, 10, 10}), 1e-9)
	assert.InDelta(t, 1, CramersV([]int{5, 0}, []int{5, 5}), 1e-9)
	assert.InDelta(t, 0, CramersV([]int{0, 0}, []int{5, 5}), 1e-9, "no outcome at all")
	// For a 2 × 2 table V = |φ|.
	assert.InDelta(t, math.Abs(Phi(7, 3, 2, 8)), CramersV([]int{7, 2}, []int{10, 10}), 1e-9)
}

func TestStrength(t *testing.T) {
	for r, want := range map[float64]string{
		0.5: StrengthStrong, -0.62: StrengthStrong, 0.49: StrengthMedium, 0.3: StrengthMedium, -0.3: StrengthMedium,
		0.29: StrengthWeak, 0.1: StrengthWeak, 0.099: "", 0: "",
	} {
		assert.Equal(t, want, Strength(r), "r=%v", r)
	}
}

func TestFIGO(t *testing.T) {
	for age, want := range map[int]int{0: 7, 16: 9, 18: 9, 25: 9, 26: 7, 41: 7, 42: 9, 45: 9} {
		assert.Equal(t, want, FIGOVariationLimit(age), "age %d", age)
	}
	assert.True(t, FIGOApplies(0))
	assert.False(t, FIGOApplies(17))
	assert.True(t, FIGOApplies(18))
	assert.True(t, FIGOApplies(45))
	assert.False(t, FIGOApplies(46))
	assert.Equal(t, 24, FIGOCycleMin)
	assert.Equal(t, 38, FIGOCycleMax)
	assert.Equal(t, 8, FIGOPeriodMax)
}

func TestMovingAverage7(t *testing.T) {
	d := civildate.MustParse("2026-09-23")
	v := map[civildate.Date]float64{d: 60, d.AddDays(-2): 61, d.AddDays(-6): 62, d.AddDays(-7): 99}
	got, ok := MovingAverage7(v, d)
	require.True(t, ok)
	assert.InDelta(t, 61, got, 1e-9, "the mean of the logged days of d−6…d; d−7 is outside, missing days are skipped")
	_, ok = MovingAverage7(v, d.AddDays(20))
	assert.False(t, ok)
}

func TestNewLayout(t *testing.T) {
	l := NewLayout(29, 5)
	assert.Equal(t, 16, l.OvulationDay, "next start − 14")
	assert.Equal(t, 11, l.FertileStart)
	assert.Equal(t, 6, l.FertileDays)
	assert.Equal(t, 5, l.FollicularDays)
	assert.Equal(t, 13, l.LutealDays)
	assert.Equal(t, 29, l.PeriodDays+l.FollicularDays+l.FertileDays+l.LutealDays)
	assert.Equal(t, PhasePeriod, l.Phase(5))
	assert.Equal(t, PhaseFollicular, l.Phase(6))
	assert.Equal(t, PhaseFertile, l.Phase(16))
	assert.Equal(t, PhaseLuteal, l.Phase(17))

	short := NewLayout(21, 7) // the fertile window never overlaps the period
	assert.Equal(t, 8, short.FertileStart)
	assert.Equal(t, 0, short.FollicularDays)
	assert.Equal(t, 21, short.PeriodDays+short.FollicularDays+short.FertileDays+short.LutealDays)
}

func TestRange(t *testing.T) {
	today := civildate.MustParse("2026-09-23")
	r := NewRange(Range3M, today)
	assert.Equal(t, 91, r.Days())
	assert.Equal(t, today, r.To)
	assert.Equal(t, Range6M, NewRange("bogus", today).Key)
	assert.True(t, IsRange(RangeAll))
	assert.False(t, IsRange("5y"))
	assert.True(t, r.Contains(r.From))
	assert.False(t, r.Contains(r.From.AddDays(-1)))
}

func TestMinData_PatternsNeedThreeCycles(t *testing.T) {
	in := ranged(teenFixture(), Range6M) // two completed cycles
	cr := BuildCycle(in)
	assert.Equal(t, 2, cr.BasedOn)
	assert.True(t, cr.Ready(), "two cycles are a trend")
	assert.Nil(t, cr.Variation)
	assert.Equal(t, RegularityNotEnoughData, cr.Regularity)
	sr := BuildSymptoms(in)
	assert.False(t, sr.PatternReady())
	assert.True(t, BuildPeriod(in).CoSymptomsReady, "three periods (two completed cycles + the current one)")

	one := ranged(teenFixture(), Range1M) // one completed cycle in a month
	assert.False(t, BuildCycle(one).Ready(), "no completed cycle in the window is not a trend")
	assert.False(t, BuildPeriod(one).CoSymptomsReady, "one period")
	assert.False(t, BuildPeriod(one).Ready())
}

func TestOutlier_OnlyALoneOneIsExcluded(t *testing.T) {
	rows := func(lens ...int) []CycleRow {
		out := make([]CycleRow, len(lens))
		for i, l := range lens {
			out[i] = CycleRow{Length: l, Counted: true}
		}
		return out
	}
	one := rows(29, 30, 38, 28, 29)
	markOutlier(one, 7)
	assert.False(t, one[2].Counted)
	assert.Equal(t, ExcludedOutlier, one[2].Excluded)

	two := rows(29, 40, 38, 28, 29) // two far cycles: real variability, all counted
	markOutlier(two, 7)
	for _, r := range two {
		assert.True(t, r.Counted)
	}
	few := rows(29, 38, 28) // the rest must still form a pattern
	markOutlier(few, 7)
	assert.True(t, few[1].Counted)
}

func TestCorrelation_MinData(t *testing.T) {
	c := binary(CorrSleepMood, Group{Key: "ok", Days: 30, Hits: 2}, Group{Key: "short", Days: 4, Hits: 4})
	assert.Equal(t, CorrNotEnoughData, c.Status, "a group under 5 days")
	c = binary(CorrSleepMood, Group{Key: "ok", Days: 10, Hits: 1}, Group{Key: "short", Days: 9, Hits: 1})
	assert.Equal(t, CorrNotEnoughData, c.Status, "under 20 paired days")
	c = binary(CorrSleepMood, Group{Key: "ok", Days: 20, Hits: 4}, Group{Key: "short", Days: 20, Hits: 4})
	assert.Equal(t, CorrNoAssociation, c.Status)
	assert.Empty(t, c.Strength)
	c = binary(CorrSleepMood, Group{Key: "ok", Days: 20, Hits: 2}, Group{Key: "short", Days: 20, Hits: 14})
	assert.Equal(t, CorrReady, c.Status)
	assert.Equal(t, StrengthStrong, c.Strength)
	require.NotNil(t, c.Ratio)
	assert.InDelta(t, 7, *c.Ratio, 1e-9)
}

func TestPlusSection_LockedComputesNothing(t *testing.T) {
	called := false
	s := plusSection(false, func() (bool, any) { called = true; return true, "x" })
	assert.False(t, called)
	assert.True(t, s.Locked)
	assert.False(t, s.Ready)
	assert.Nil(t, s.Data)

	// A free user's summary never computes the correlations.
	in := ranged(regularFixture(), Range6M)
	in.DeepAnalysis = false
	assert.Nil(t, BuildSummary(in).Correlations)
}

func TestCopy_NumbersAndLabels(t *testing.T) {
	fa, en := copyFor(t, "fa"), copyFor(t, "en")
	assert.Equal(t, "۵۸٫۴", fa.Number(58.4))
	assert.Equal(t, "−۰٫۶", fa.Number(-0.6))
	assert.Equal(t, "۲۹", fa.Number(29))
	assert.Equal(t, "58.4", en.Number(58.4))
	assert.Equal(t, "2", en.Number(2.0))
	assert.Equal(t, "سردرد", fa.SymptomLabel("pain.location.head"), "analysis override of a pain location")
	assert.Equal(t, "نفخ", fa.SymptomLabel("symptoms.digestive.bloating"), "taxonomy label")
	assert.Equal(t, "x.y.z", fa.SymptomLabel("x.y.z"))
	assert.Equal(t, "a, b and c", en.List([]string{"a", "b", "c"}))
	assert.Empty(t, fa.Render(Phrase{Key: "findings.nope"}))
	var none *Copy
	assert.Empty(t, none.Render(Phrase{Key: "findings.no_data"}))
}

func TestDays_Weights(t *testing.T) {
	days := BuildDays([]DayEntries{{Date: fixtureToday, Entries: []taxonomy.Entry{
		item("pain", "location", "head", "mild"),
		item("symptoms", "general", "fatigue", "no"),
		multi("mood", "moods", "happy"),
		multi("mood", "moods", "sad"),
		single("sleep", "duration", "0_3"),
	}}})
	d := days[fixtureToday]
	assert.InDelta(t, 1.0/3, d.Symptoms["pain.location.head"], 1e-9)
	assert.NotContains(t, d.Symptoms, "symptoms.general.fatigue", "level no is not a symptom")
	assert.NotContains(t, d.Symptoms, "mood.moods.happy", "positive moods are not symptoms")
	assert.Contains(t, d.Symptoms, "mood.moods.sad")
	good, ok := d.GoodMood()
	assert.True(t, ok)
	assert.True(t, good, "one positive, one negative")
	h, _ := d.SleepHours()
	assert.InDelta(t, 1.5, h, 1e-9)
	assert.True(t, d.ShortSleep())
}

func TestInput_IgnoresUnconfirmedAndEstimated(t *testing.T) {
	in := ranged(regularFixture(), Range6M)
	in.Histories = append(in.Histories,
		model.History{PeriodStart: civildate.MustParse("2026-09-20"), IsConfirmed: false},
		model.History{PeriodStart: civildate.MustParse("2026-09-21"), IsConfirmed: true, IsEstimated: true})
	cur, ok := in.currentCycle()
	require.True(t, ok)
	assert.Equal(t, civildate.MustParse("2026-09-14"), cur.Start)
}
