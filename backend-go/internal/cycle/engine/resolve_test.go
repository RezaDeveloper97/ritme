package engine

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// periodsEnding builds closed periods of periodLen days whose consecutive starts differ by lengths, the last one
// starting on last.
func periodsEnding(last Day, periodLen int, lengths ...int) []Period {
	s := last
	for _, l := range lengths {
		s -= Day(l)
	}
	out := []Period{{Start: s, End: s + Day(periodLen-1)}}
	for _, l := range lengths {
		s += Day(l)
		out = append(out, Period{Start: s, End: s + Day(periodLen-1)})
	}
	return out
}

// regular is a user with three closed periods of periodLen days, two cycles of length l, the last start on cur.
func regular(cur Day, l, periodLen int) Input {
	return withPeriods(periodsEnding(cur, periodLen, l, l))
}

// none is an Input with no data: the zero Input (optional dates are Opt[Day], so Day 0 = 1970-01-01 never sneaks in).
func none() Input { return Input{} }

func withPeriods(ps []Period) Input {
	in := none()
	in.Periods = ps
	return in
}

func declared(d Declared) Input {
	in := none()
	in.Declared = d
	return in
}

// specFertility is the §2.26 table, written out independently of fertilityOf.
func specFertility(s Subphase) FertilityLevel {
	return map[Subphase]FertilityLevel{
		SubphaseMenstrual: FertilityLow, SubphaseMenstrualPossible: FertilityLow,
		SubphaseEarlyFollicular: FertilityLow, SubphaseMidFollicular: FertilityLow,
		SubphaseLateFollicularTransition: FertilityMedium, SubphaseFertileRising: FertilityMedium,
		SubphaseHighFertility: FertilityHigh, SubphaseOvulationLikely: FertilityPeak,
		SubphasePostOvulation: FertilityLow, SubphaseEarlyLuteal: FertilityLow, SubphaseMidLuteal: FertilityLow,
		SubphaseLateLuteal: FertilityLow, SubphasePMSPossible: FertilityLow,
		SubphasePeriodExpected: FertilityUnknown, SubphaseUnknown: FertilityUnknown,
	}[s]
}

type span struct {
	from, to int
	sub      Subphase
}

// TestResolve_DeterministicTables is §2.36–§2.38 verbatim: a confirmed 5-day period, no LH/BBT, cycles of 21/28/35
// days, every day 1 … L+1. The 35-day follicular range (days 6–16) is split by §2.23 (11 days → 4/4/3).
func TestResolve_DeterministicTables(t *testing.T) {
	tables := map[int][]span{
		21: {
			{1, 5, SubphaseMenstrual},
			{6, 7, SubphaseHighFertility}, {8, 8, SubphaseOvulationLikely},
			{9, 9, SubphasePostOvulation}, {10, 12, SubphaseEarlyLuteal}, {13, 15, SubphaseMidLuteal},
			{16, 18, SubphaseLateLuteal}, {19, 21, SubphasePMSPossible}, {22, 22, SubphasePeriodExpected},
		},
		28: {
			{1, 5, SubphaseMenstrual},
			{6, 6, SubphaseEarlyFollicular}, {7, 7, SubphaseMidFollicular}, {8, 9, SubphaseLateFollicularTransition},
			{10, 12, SubphaseFertileRising}, {13, 14, SubphaseHighFertility}, {15, 15, SubphaseOvulationLikely},
			{16, 16, SubphasePostOvulation}, {17, 19, SubphaseEarlyLuteal}, {20, 22, SubphaseMidLuteal},
			{23, 25, SubphaseLateLuteal}, {26, 28, SubphasePMSPossible}, {29, 29, SubphasePeriodExpected},
		},
		35: {
			{1, 5, SubphaseMenstrual},
			{6, 9, SubphaseEarlyFollicular}, {10, 13, SubphaseMidFollicular}, {14, 16, SubphaseLateFollicularTransition},
			{17, 19, SubphaseFertileRising}, {20, 21, SubphaseHighFertility}, {22, 22, SubphaseOvulationLikely},
			{23, 23, SubphasePostOvulation}, {24, 26, SubphaseEarlyLuteal}, {27, 29, SubphaseMidLuteal},
			{30, 32, SubphaseLateLuteal}, {33, 35, SubphasePMSPossible}, {36, 36, SubphasePeriodExpected},
		},
	}
	ovulationDay := map[int]int{21: 8, 28: 15, 35: 22}
	cur := day("2026-07-01")
	for _, l := range []int{21, 28, 35} {
		s := prep(regular(cur, l, 5))
		for _, sp := range tables[l] {
			for d := sp.from; d <= sp.to; d++ {
				t.Run(fmt.Sprintf("§2.36 L=%d day %d", l, d), func(t *testing.T) {
					target := cur + Day(d-1)
					st := s.Resolve(target, target)
					assert.Equal(t, sp.sub.String(), st.Subphase.String())
					assert.Equal(t, sp.sub.Main(), st.MainPhase)
					assert.Equal(t, specFertility(sp.sub).String(), st.FertilityLevel.String())
					assert.Equal(t, Some(d), st.CycleDay)
					assert.Equal(t, cur+Day(ovulationDay[l]-1), st.Anchors.EstimatedOvulationDate)
					assert.Equal(t, cur+Day(l), st.Anchors.PredictedNextPeriodStart)
					assert.Equal(t, OvulationCalendar, st.Anchors.OvulationSource)
					assert.False(t, st.Warnings.Has(WarnPredictionBasedOutput), "E-01")
					want := ResolutionPrediction
					if sp.sub == SubphaseMenstrual {
						want = ResolutionUserLogged
					}
					assert.Equal(t, want, st.ResolutionSource)
				})
			}
		}
	}
}

// TestResolve_SpecAPIExample is the §2.35 worked example.
func TestResolve_SpecAPIExample(t *testing.T) {
	cur := day("2026-07-01")
	s := prep(withPeriods(periodsEnding(cur, 5, 27, 29, 30)))
	st := s.Resolve(day("2026-07-18"), day("2026-07-18"))

	assert.Equal(t, Some(18), st.CycleDay)
	assert.Equal(t, MainPhaseLuteal, st.MainPhase)
	assert.Equal(t, SubphaseEarlyLuteal, st.Subphase)
	assert.Equal(t, FertilityLow, st.FertilityLevel)
	assert.Equal(t, Some(-2), st.DaysToOvulation)
	assert.Equal(t, Some(12), st.DaysToPeriod)
	assert.Equal(t, Some(0), st.DaysLate)
	assert.Equal(t, Anchors{
		CurrentPeriodStart: cur, CurrentPeriodStartSource: StartUserLogged,
		CurrentPeriodEnd: day("2026-07-05"), CurrentPeriodEndSource: EndUserLogged, CurrentPeriodEndIsConfirmed: true,
		PredictedNextPeriodStart: day("2026-07-30"), EstimatedOvulationDate: day("2026-07-16"),
		OvulationSource: OvulationCalendar,
	}, st.Anchors)
	assert.Equal(t, Metrics{EffectiveCycleLength: 29, EffectivePeriodLength: 5, CycleVariability: Some(3),
		CycleRegularity: RegularityRegular}, st.Metrics)
	assert.Equal(t, ClinicalFlags{Frequency: Some(FrequencyNormal), Duration: Some(DurationNormal), Absent: Some(false)},
		st.ClinicalFlags)
	assert.Equal(t, ConfidenceHigh, st.Confidence)
	assert.Equal(t, DataQualityGood, st.DataQuality)
	assert.False(t, st.SuggestPregnancyTest)
	assert.Equal(t, ResolutionPrediction, st.ResolutionSource)
	assert.True(t, st.IsPredicted())
	assert.False(t, st.RequiresUserInput)
	assert.Zero(t, st.Warnings, "§2.35: warnings []")
	assert.Zero(t, st.ConfidenceReasons)
}

func TestResolve_OpenPeriodTiers(t *testing.T) {
	cur := day("2026-07-01")
	in := regular(cur-28, 28, 5)
	in.Periods = append(in.Periods, Period{Start: cur, End: NoDay})
	s := prep(in)
	require.Equal(t, 5, s.periodLen)

	all := func(ws ...Warning) WarningSet {
		var set WarningSet
		for _, w := range ws {
			set.Add(w)
		}
		return set
	}
	missing := all(WarnPeriodEndMissing)
	warn := all(WarnPeriodEndMissing, WarnPeriodEndMissingWarningCapExceeded)
	hard := all(WarnPeriodEndMissing, WarnPeriodEndMissingWarningCapExceeded, WarnPeriodEndMissingHardCapExceeded)

	cases := []struct {
		day   int
		sub   Subphase
		warns WarningSet
		res   ResolutionSource
		poor  bool
	}{
		{5, SubphaseMenstrual, 0, ResolutionUserLoggedWithAssumedEnd, false},               // 21.2
		{6, SubphaseMenstrualPossible, missing, ResolutionUserLoggedWithAssumedEnd, false}, // 21.3
		{8, SubphaseMenstrualPossible, missing, ResolutionUserLoggedWithAssumedEnd, false}, // warning cap max(5+3, 8)
		{9, SubphaseMenstrualPossible, warn, ResolutionUserLoggedWithAssumedEnd, true},     // 21.4
		{12, SubphaseMenstrualPossible, warn, ResolutionUserLoggedWithAssumedEnd, true},    // hard cap 12
		{13, SubphaseHighFertility, hard, ResolutionPrediction, true},                      // 21.5 onward (E-12)
		{20, SubphaseMidLuteal, hard, ResolutionPrediction, true},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("day %d", c.day), func(t *testing.T) {
			target := cur + Day(c.day-1)
			st := s.Resolve(target, target)
			assert.Equal(t, c.sub, st.Subphase)
			assert.Equal(t, c.warns, st.Warnings)
			assert.Equal(t, c.res, st.ResolutionSource)
			assert.Equal(t, c.poor, st.DataQuality == DataQualityPoor)
			assert.Equal(t, c.poor, st.Confidence == ConfidenceLow)
			assert.Equal(t, c.poor, st.RequiresUserInput)
			assert.Equal(t, c.day >= 13, st.ConfidenceReasons.Has(ReasonMissingPeriodEndBeyondHardCap))
			assert.False(t, st.Anchors.CurrentPeriodEndIsConfirmed)
			assert.Equal(t, EndAssumedFromEffectiveDuration, st.Anchors.CurrentPeriodEndSource)
			assert.Equal(t, cur+4, st.Anchors.CurrentPeriodEnd, "§2.11 assumed end = start + EPL − 1")
		})
	}
}

// With EPL 10 the warning cap (13) is above the hard cap (12): the hard cap wins on day 13.
func TestResolve_HardCapBeforeWarningCap(t *testing.T) {
	cur := day("2026-07-01")
	in := regular(cur-28, 28, 10)
	in.Periods = append(in.Periods, Period{Start: cur, End: NoDay})
	s := prep(in)
	st := s.Resolve(cur+11, cur+11)
	assert.Equal(t, SubphaseMenstrualPossible, st.Subphase)
	st = s.Resolve(cur+12, cur+12)
	assert.NotEqual(t, MainPhaseMenstrual, st.MainPhase)
	assert.True(t, st.Warnings.Has(WarnPeriodEndMissingHardCapExceeded))
}

func TestResolve_DaysLate(t *testing.T) {
	cur := day("2026-07-01")
	s := prep(regular(cur, 28, 5))
	next := cur + 28
	ov := next - 14

	for _, late := range []int{0, 7} {
		st := s.Resolve(next+Day(late), next+Day(late))
		assert.Equal(t, SubphasePeriodExpected, st.Subphase, "late %d", late)
		assert.Equal(t, FertilityUnknown, st.FertilityLevel)
		assert.Equal(t, ResolutionPrediction, st.ResolutionSource)
		assert.Zero(t, st.Warnings, "late %d", late)
		assert.Equal(t, Some(late), st.DaysLate)
		assert.Equal(t, Some(0), st.DaysToPeriod)
		assert.Equal(t, Some(int(ov-next)-late), st.DaysToOvulation)
	}
	for _, late := range []int{8, 14} {
		st := s.Resolve(next+Day(late), next+Day(late))
		assert.Equal(t, SubphasePeriodExpected, st.Subphase, "late %d", late)
		assert.True(t, st.Warnings.Has(WarnPredictedPeriodOverdue))
		assert.False(t, st.Warnings.Has(WarnCycleUnresolvedAfterExpectedPeriod))
		assert.Equal(t, ConfidenceLow, st.Confidence)
		assert.False(t, st.RequiresUserInput)
	}
	st := s.Resolve(next+14, next+14)
	assert.Equal(t, Some(-28), st.DaysToOvulation, "§2.34: late 14 → −(14 + 14)")

	st = s.Resolve(next+15, next+15)
	assert.Equal(t, MainPhaseUnknown, st.MainPhase)
	assert.Equal(t, SubphaseUnknown, st.Subphase)
	assert.Equal(t, FertilityUnknown, st.FertilityLevel)
	assert.Equal(t, ResolutionPrediction, st.ResolutionSource, "E-11")
	assert.Equal(t, Some(0), st.DaysToPeriod)
	assert.Equal(t, Some(15), st.DaysLate)
	assert.False(t, st.DaysToOvulation.OK())
	assert.Equal(t, Some(44), st.CycleDay)
	assert.Equal(t, cur, st.Anchors.CurrentPeriodStart)
	assert.Equal(t, next, st.Anchors.PredictedNextPeriodStart)
	assert.Equal(t, ov, st.Anchors.EstimatedOvulationDate)
	assert.True(t, st.Warnings.Has(WarnPredictedPeriodOverdue))
	assert.True(t, st.Warnings.Has(WarnCycleUnresolvedAfterExpectedPeriod))
	assert.True(t, st.RequiresUserInput)
	assert.Equal(t, ConfidenceLow, st.Confidence)
	assert.Equal(t, DataQualityPoor, st.DataQuality)
}

func TestResolve_ShortAndEmptyWindows(t *testing.T) {
	cur := day("2026-07-01")

	s := prep(regular(cur, 21, 8)) // O = day 8 inside the period → empty display window
	st := s.Resolve(cur+7, cur+7)
	assert.Equal(t, SubphaseMenstrual, st.Subphase, "day 8 menstrual")
	st = s.Resolve(cur+8, cur+8)
	assert.Equal(t, SubphasePostOvulation, st.Subphase, "day 9 post_ovulation")

	s = prep(regular(cur, 21, 7)) // window = day 8 only
	st = s.Resolve(cur+7, cur+7)
	assert.Equal(t, SubphaseOvulationLikely, st.Subphase, "day 8 ovulation_likely")
	assert.Equal(t, FertilityPeak, st.FertilityLevel)
}

// O+1 is never fertile and a menstrual day inside the biological window stays menstrual, for every supported
// cycle and period length.
func TestResolve_WindowInvariants(t *testing.T) {
	cur := day("2026-07-01")
	for l := 21; l <= 45; l++ {
		for pl := 2; pl <= 10; pl++ {
			s := prep(regular(cur, l, pl))
			ov := cur + Day(l-14)
			st := s.Resolve(ov+1, ov+1)
			assert.NotEqual(t, MainPhaseFertile, st.MainPhase, "L=%d P=%d O+1", l, pl)
			for d := ov - 5; d <= ov; d++ {
				st = s.Resolve(d, d)
				if d <= cur+Day(pl-1) {
					assert.Equal(t, MainPhaseMenstrual, st.MainPhase, "L=%d P=%d %s", l, pl, d)
				} else {
					assert.Equal(t, MainPhaseFertile, st.MainPhase, "L=%d P=%d %s", l, pl, d)
				}
			}
		}
	}
}

func TestLayout_LutealPriority(t *testing.T) {
	p := DefaultParams()
	start := day("2026-07-01")
	next := start + 28
	mk := func(ov Day) layout { return newLayout(start, start+4, true, next, ov, p.FertileBefore) }

	l := mk(next - 4) // O+1 = next − 3
	assert.Equal(t, SubphasePMSPossible, l.cyclePhase(l.ov+1, &p), "pms_possible > post_ovulation")
	l = mk(next - 7) // O+1 = next − 6
	assert.Equal(t, SubphaseLateLuteal, l.cyclePhase(l.ov+1, &p), "late_luteal > post_ovulation")

	l = mk(next - 10) // assignable O+2 .. next−7 = 2 days
	assert.Equal(t, SubphasePostOvulation, l.cyclePhase(l.ov+1, &p))
	assert.Equal(t, SubphaseEarlyLuteal, l.cyclePhase(l.ov+2, &p))
	assert.Equal(t, SubphaseMidLuteal, l.cyclePhase(l.ov+3, &p))
	assert.Equal(t, SubphaseLateLuteal, l.cyclePhase(l.ov+4, &p))

	l = mk(next - 15) // 7 days → ⌈7/2⌉ = 4 early, 3 mid
	for i, want := range []Subphase{SubphaseEarlyLuteal, SubphaseEarlyLuteal, SubphaseEarlyLuteal,
		SubphaseEarlyLuteal, SubphaseMidLuteal, SubphaseMidLuteal, SubphaseMidLuteal} {
		assert.Equal(t, want, l.cyclePhase(l.ov+2+Day(i), &p), "O+%d", i+2)
	}

	// An empty assignable range (O+2 > next−7) → mid_luteal.
	ov := next - 8
	assert.Equal(t, SubphaseMidLuteal, assignableLuteal(ov+2, ov, next, &p))
}

func TestFollicularSplit(t *testing.T) {
	split := func(n int) (e, m, l int) {
		for pos := 1; pos <= n; pos++ {
			switch follicularSubphase(pos, n) {
			case SubphaseEarlyFollicular:
				e++
			case SubphaseMidFollicular:
				m++
			case SubphaseLateFollicularTransition:
				l++
			}
		}
		return e, m, l
	}
	assert.Equal(t, SubphaseMidFollicular, follicularSubphase(1, 1))
	assert.Equal(t, SubphaseEarlyFollicular, follicularSubphase(1, 2))
	assert.Equal(t, SubphaseLateFollicularTransition, follicularSubphase(2, 2))
	for n, want := range map[int][3]int{3: {1, 1, 1}, 4: {1, 1, 2}, 5: {2, 2, 1}, 11: {4, 4, 3}} {
		e, m, l := split(n)
		assert.Equal(t, want, [3]int{e, m, l}, "n=%d", n)
	}
}

func TestResolve_Onboarding(t *testing.T) {
	decl := day("2026-07-01")
	s := prep(declared(Declared{LastPeriodStart: Some(decl), CycleLength: Some(28), PeriodLength: Some(5)}))
	for d := 1; d <= 29+14; d++ {
		st := s.Resolve(decl+Day(d-1), decl+Day(d-1))
		assert.Equal(t, ResolutionOnboardingBased, st.ResolutionSource, "day %d (%s)", d, st.Subphase)
		assert.Equal(t, StartOnboardingDeclared, st.Anchors.CurrentPeriodStartSource)
		assert.True(t, st.Warnings.Has(WarnOnboardingDataUsed))
		assert.False(t, st.Warnings.Has(WarnDefaultValuesUsed))
		assert.False(t, st.Warnings.Has(WarnPeriodEndMissing), "no end-missing tiers on a declared start")
	}
	st := s.Resolve(decl+28, decl+28)
	assert.Equal(t, SubphasePeriodExpected, st.Subphase)
	st = s.Resolve(decl, decl)
	assert.Equal(t, SubphaseMenstrual, st.Subphase)
	assert.False(t, st.Anchors.CurrentPeriodEndIsConfirmed)
	assert.Equal(t, decl+4, st.Anchors.CurrentPeriodEnd)
	st = s.Resolve(decl+5, decl+5)
	assert.Equal(t, MainPhaseFollicular, st.MainPhase)

	st = s.Resolve(decl+43, decl+43) // late 15 → unknown, prediction (E-11)
	assert.Equal(t, MainPhaseUnknown, st.MainPhase)
	assert.Equal(t, ResolutionPrediction, st.ResolutionSource)

	// Declared date without lengths → default_based.
	s = prep(declared(Declared{LastPeriodStart: Some(decl)}))
	st = s.Resolve(decl+10, decl+10)
	assert.Equal(t, ResolutionDefaultBased, st.ResolutionSource)
	assert.True(t, st.Warnings.Has(WarnDefaultValuesUsed))
	assert.True(t, st.Warnings.Has(WarnOnboardingDataUsed))

	// Only one length declared → onboarding_based.
	s = prep(declared(Declared{LastPeriodStart: Some(decl), CycleLength: Some(30)}))
	assert.Equal(t, ResolutionOnboardingBased, s.Resolve(decl+10, decl+10).ResolutionSource)

	// A declared start after the target does not anchor it.
	s = prep(declared(Declared{LastPeriodStart: Some(decl)}))
	assert.Equal(t, ResolutionUnknown, s.Resolve(decl-1, decl).ResolutionSource)
}

func TestResolve_NoAnchor(t *testing.T) {
	st := prep(none()).Resolve(day("2026-07-01"), day("2026-07-01"))
	assert.Equal(t, MainPhaseUnknown, st.MainPhase)
	assert.Equal(t, SubphaseUnknown, st.Subphase)
	assert.Equal(t, FertilityUnknown, st.FertilityLevel)
	assert.Equal(t, ResolutionUnknown, st.ResolutionSource)
	assert.False(t, st.CycleDay.OK())
	assert.False(t, st.DaysToPeriod.OK())
	assert.False(t, st.DaysLate.OK())
	assert.False(t, st.DaysToOvulation.OK())
	assert.Equal(t, NoDay, st.Anchors.CurrentPeriodStart)
	assert.Equal(t, NoDay, st.Anchors.CurrentPeriodEnd)
	assert.Equal(t, NoDay, st.Anchors.PredictedNextPeriodStart)
	assert.Equal(t, NoDay, st.Anchors.EstimatedOvulationDate)
	assert.Equal(t, StartUnknown, st.Anchors.CurrentPeriodStartSource)
	assert.Equal(t, EndUnknown, st.Anchors.CurrentPeriodEndSource)
	assert.Equal(t, OvulationUnknown, st.Anchors.OvulationSource)
	var want WarningSet
	want.Add(WarnInsufficientAnchorData)
	assert.Equal(t, want, st.Warnings)
	assert.True(t, st.RequiresUserInput)
	assert.Equal(t, DataQualityInsufficient, st.DataQuality)
	assert.Equal(t, ConfidenceUnknown, st.Confidence)
	assert.True(t, st.ConfidenceReasons.Has(ReasonInsufficientAnchorData))
	assert.False(t, st.SuggestPregnancyTest)

	// Real periods only after the target, no declared start → still no anchor.
	cur := day("2026-07-01")
	st = prep(regular(cur, 28, 5)).Resolve(cur-100, cur)
	assert.Equal(t, ResolutionUnknown, st.ResolutionSource)
}

func TestResolve_RealStartBeatsDeclared(t *testing.T) {
	cur := day("2026-07-01")
	in := regular(cur, 28, 5)
	in.Declared = Declared{LastPeriodStart: Some(cur + 5), CycleLength: Some(30), PeriodLength: Some(4)}
	st := prep(in).Resolve(cur+10, cur+10)
	assert.Equal(t, cur, st.Anchors.CurrentPeriodStart)
	assert.Equal(t, StartUserLogged, st.Anchors.CurrentPeriodStartSource)
	assert.False(t, st.Warnings.Has(WarnOnboardingDataUsed))
}

func TestResolve_SuggestPregnancyTest(t *testing.T) {
	cur := day("2026-07-01")
	next := cur + 28
	cases := []struct {
		name     string
		mode     Mode
		hormonal bool
		late     int
		want     bool
	}{
		{"ttc + 1", ModeTTC, false, 1, true},
		{"cycle + 7", ModeCycle, false, 7, false},
		{"cycle + 8", ModeCycle, false, 8, true},
		{"cycle + 8 hormonal", ModeCycle, true, 8, false},
		{"pregnancy + 8", ModePregnancy, false, 8, false},
		{"cycle late 0", ModeCycle, false, 0, false},
		{"ttc late 0", ModeTTC, false, 0, false},
		{"cycle + 15 (unknown phase)", ModeCycle, false, 15, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := regular(cur, 28, 5)
			in.Mode, in.Hormonal = c.mode, c.hormonal
			target := next + Day(c.late)
			assert.Equal(t, c.want, prep(in).Resolve(target, target).SuggestPregnancyTest)
		})
	}
}

// E-22: completed cycles are laid out over their real length.
func TestResolve_Retrospective(t *testing.T) {
	p0 := day("2026-01-01")
	p1 := p0 + 28
	p2 := p1 + 35
	s := prep(withPeriods([]Period{{p0, p0 + 4, PeriodUserLogged}, {p1, p1 + 4, PeriodUserLogged},
		{p2, p2 + 4, PeriodUserLogged}}))

	st := s.Resolve(p1+21, p2+10)
	assert.Equal(t, SubphaseOvulationLikely, st.Subphase, "ovulation on day 22 of the 35-day cycle")
	assert.Equal(t, Some(22), st.CycleDay)
	assert.Equal(t, p2, st.Anchors.PredictedNextPeriodStart, "the actual next start replaces the prediction")
	assert.Equal(t, p1+21, st.Anchors.EstimatedOvulationDate)
	for d := p1; d < p2; d++ {
		st = s.Resolve(d, p2+10)
		assert.NotEqual(t, MainPhasePeriodExpected, st.MainPhase, "%s", d)
		assert.NotEqual(t, MainPhaseUnknown, st.MainPhase, "%s", d)
		assert.Equal(t, Some(0), st.DaysLate)
	}
	st = s.Resolve(p2-1, p2+10)
	assert.Equal(t, SubphasePMSPossible, st.Subphase)

	// The 28-day cycle before it: ovulation on day 15.
	assert.Equal(t, SubphaseOvulationLikely, s.Resolve(p0+14, p2).Subphase)

	// The current cycle still predicts from the effective length: median(28, 35) = 31.5 → 32.
	st = s.Resolve(p2+3, p2+3)
	assert.Equal(t, p2+32, st.Anchors.PredictedNextPeriodStart)
	assert.Equal(t, p2+18, st.Anchors.EstimatedOvulationDate)
	assert.Equal(t, SubphasePeriodExpected, s.Resolve(p2+32, p2+32).Subphase)
}

// A declared start followed by a real start is laid out up to that real start.
func TestResolve_DeclaredThenReal(t *testing.T) {
	decl := day("2026-06-01")
	real := day("2026-07-10") // 39 days later
	in := declared(Declared{LastPeriodStart: Some(decl), CycleLength: Some(28), PeriodLength: Some(5)})
	in.Periods = []Period{{Start: real, End: real + 4}}
	s := prep(in)
	st := s.Resolve(decl+30, real)
	assert.Equal(t, real, st.Anchors.PredictedNextPeriodStart)
	assert.Equal(t, MainPhaseLuteal, st.MainPhase)
	assert.Equal(t, ResolutionOnboardingBased, st.ResolutionSource)
	st = s.Resolve(real+1, real+1)
	assert.Equal(t, StartUserLogged, st.Anchors.CurrentPeriodStartSource)
}

func TestResolve_ManualLengthsAreNotOnboarding(t *testing.T) {
	cur := day("2026-07-01")
	in := regular(cur, 28, 5)
	in.Declared = Declared{CycleLength: Some(30), PeriodLength: Some(6), Manual: true}
	st := prep(in).Resolve(cur+10, cur+10)
	assert.Equal(t, cur+30, st.Anchors.PredictedNextPeriodStart)
	assert.False(t, st.Warnings.Has(WarnOnboardingDataUsed), "E-20")
}

// Gregorian arithmetic across Jalali boundaries: Esfand 30 1403 (leap) → Farvardin 1 1404 is 2025-03-20/21;
// Shahrivar 31 → Mehr 1 1405 is 2026-09-22/23.
func TestResolve_JalaliBoundaries(t *testing.T) {
	cases := []struct {
		start   string
		targets map[string]int
	}{
		{"2025-03-10", map[string]int{"2025-03-19": 10, "2025-03-20": 11, "2025-03-21": 12, "2025-03-22": 13,
			"2025-04-06": 28}},
		{"2026-09-15", map[string]int{"2026-09-21": 7, "2026-09-22": 8, "2026-09-23": 9, "2026-09-24": 10,
			"2026-10-12": 28}},
	}
	for _, c := range cases {
		cur := day(c.start)
		s := prep(regular(cur, 28, 5))
		for target, want := range c.targets {
			st := s.Resolve(day(target), day(target))
			assert.Equal(t, Some(want), st.CycleDay, "%s from %s", target, c.start)
		}
		assert.Equal(t, cur+28, s.Resolve(cur, cur).Anchors.PredictedNextPeriodStart)
	}
	assert.Equal(t, "2025-04-07", (day("2025-03-10") + 28).String())
	assert.Equal(t, "2026-10-13", (day("2026-09-15") + 28).String())
}

func TestResolve_NoAllocs(t *testing.T) {
	cur := day("2026-07-01")
	s := prep(withPeriods(periodsEnding(cur, 5, 28, 29, 27, 30, 28, 28)))
	var st Status
	targets := []Day{cur + 3, cur + 12, cur + 20, cur + 40, cur - 30, cur - 400}
	allocs := testing.AllocsPerRun(200, func() {
		for _, t := range targets {
			st = s.Resolve(t, cur)
		}
	})
	assert.Zero(t, allocs)
	_ = st
}

func BenchmarkResolve(b *testing.B) {
	lengths := make([]int, 23)
	for i := range lengths {
		lengths[i] = 26 + i%5
	}
	cur := day("2026-07-01")
	s := prep(withPeriods(periodsEnding(cur, 5, lengths...)))
	require.Len(b, s.Periods(), 24)
	b.ReportAllocs()
	b.ResetTimer()
	var st Status
	for i := range b.N {
		t := cur - 300 + Day(i%330)
		st = s.Resolve(t, cur)
	}
	_ = st
}
