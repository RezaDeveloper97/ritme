package engine

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const qualityCur = "2026-07-01"

// lens is a user whose closed 5-day periods are separated by lengths, the last start on 2026-07-01.
func lens(lengths ...int) Input { return withPeriods(periodsEnding(day(qualityCur), 5, lengths...)) }

// aged sets a birth date so the user is exactly years old on d.
func aged(in Input, d Day, years int) Input {
	in.BirthDate = Some(monthsBefore(d, 12*years))
	return in
}

func reasons(rs ...Reason) ReasonSet {
	var set ReasonSet
	for _, r := range rs {
		set.Add(r)
	}
	return set
}

func TestMonthsBeforeAndAge(t *testing.T) {
	assert.Equal(t, day("2026-02-28"), monthsBefore(day("2026-03-31"), 1))
	assert.Equal(t, day("2023-02-28"), monthsBefore(day("2024-02-29"), 12))
	assert.Equal(t, day("2025-01-15"), monthsBefore(day("2026-01-15"), 12))
	assert.Equal(t, day("2025-12-15"), monthsBefore(day("2026-01-15"), 1))
	assert.Equal(t, NoDay, monthsBefore(day("0000-06-01"), 12))

	s := prep(Input{BirthDate: Some(day("2000-02-29"))})
	for d, want := range map[string]int{"2026-02-28": 25, "2026-03-01": 26, "2000-02-29": 0, "2024-02-29": 24} {
		age, ok := s.ageAt(day(d))
		assert.True(t, ok)
		assert.Equal(t, want, age, d)
	}
	_, ok := s.ageAt(day("1999-01-01"))
	assert.False(t, ok, "born after the date")
	_, ok = prep(none()).ageAt(day("2026-01-01"))
	assert.False(t, ok, "no birth date → no age rules (E-27)")
}

func TestQuality_Regularity(t *testing.T) {
	cur := day(qualityCur)
	target := cur + 10
	skipped := lens(28, 56, 28)
	skipped.Skips = []Day{cur - 28 - 28}
	cases := []struct {
		name string
		in   Input
		want CycleRegularity
	}{
		{"age 30 [27,29,33] → regular", aged(lens(27, 29, 33), target, 30), RegularityRegular},
		{"age 30 [26,30,34] → irregular", aged(lens(26, 30, 34), target, 30), RegularityIrregular},
		{"age 22 [26,30,34] → regular (young band 9)", aged(lens(26, 30, 34), target, 22), RegularityRegular},
		{"age 41 [26,30,34] → irregular (adult band)", aged(lens(26, 30, 34), target, 41), RegularityIrregular},
		{"age 42 [26,30,34] → regular (late band)", aged(lens(26, 30, 34), target, 42), RegularityRegular},
		{"age 16 [26,30,34] → regular (< 18: young band)", aged(lens(26, 30, 34), target, 16), RegularityRegular},
		{"age 16 [24,30,34] → irregular (10 > 9)", aged(lens(24, 30, 34), target, 16), RegularityIrregular},
		{"age 46 [26,30,34] → regular (> 45: late band)", aged(lens(26, 30, 34), target, 46), RegularityRegular},
		{"age 46 [24,30,34] → irregular", aged(lens(24, 30, 34), target, 46), RegularityIrregular},
		{"no birth date → adult band", lens(26, 30, 34), RegularityIrregular},
		{"teen band only → young band", func() Input { in := lens(26, 30, 34); in.TeenBandOnly = true; return in }(),
			RegularityRegular},
		{"2 cycles → insufficient_data", lens(28, 28), RegularityInsufficientData},
		{"3 cycles, one skipped → insufficient_data", skipped, RegularityInsufficientData},
		{"cycles older than 12 months ignored", lens(21, 45, 21, 300, 28, 28, 28), RegularityRegular},
		{"the 300-day cycle starting inside the window counts", lens(21, 45, 21, 200, 28, 28, 28),
			RegularityIrregular},
		{"outlier [28,28,50] → irregular (E-09)", lens(28, 28, 50), RegularityIrregular},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want.String(), prep(c.in).Resolve(target, target).Metrics.CycleRegularity.String())
		})
	}

	// ref = min(target, today): a past target only sees cycles complete by then; a future target sees today's.
	s := prep(lens(28, 28, 28))
	assert.Equal(t, RegularityRegular, s.Resolve(cur+5, cur+5).Metrics.CycleRegularity)
	assert.Equal(t, RegularityInsufficientData, s.Resolve(cur-1, cur+5).Metrics.CycleRegularity,
		"the last cycle completes on cur")
	assert.Equal(t, RegularityRegular, s.Resolve(cur+40, cur+5).Metrics.CycleRegularity)
}

func TestQuality_ClinicalFlags(t *testing.T) {
	cur := day(qualityCur)
	target := cur + 10
	freq := func(in Input) Opt[Frequency] { return prep(in).Resolve(target, target).ClinicalFlags.Frequency }

	assert.Equal(t, Some(FrequencyFrequent), freq(lens(23, 23, 23)), "23 < 24")
	assert.Equal(t, Some(FrequencyNormal), freq(lens(24, 24, 24)))
	assert.Equal(t, Some(FrequencyNormal), freq(lens(38, 38, 38)))
	assert.Equal(t, Some(FrequencyInfrequent), freq(lens(39, 39, 39)), "39 > 38")
	assert.Equal(t, Some(FrequencyNormal), freq(aged(lens(23, 23, 23), target, 16)), "teen 21–45")
	assert.Equal(t, Some(FrequencyFrequent), freq(aged(lens(20, 20, 20), target, 16)))
	assert.Equal(t, Some(FrequencyInfrequent), freq(aged(lens(46, 46, 46), target, 16)))
	teenBand := lens(23, 23, 23)
	teenBand.TeenBandOnly = true
	assert.Equal(t, Some(FrequencyNormal), freq(teenBand), "teen band only (E-27)")
	assert.Equal(t, Some(FrequencyFrequent), freq(aged(lens(23, 23, 23), target, 18)), "18 is adult")

	real18 := lens(18, 18, 18)
	s := prep(real18)
	n, src := s.CycleLength()
	assert.Equal(t, 28, n, "18-day cycles are outliers for effective_cycle_length")
	assert.Equal(t, LengthDefault, src)
	assert.Equal(t, Some(FrequencyFrequent), freq(real18), "E-06: real complete cycles, outliers included")

	assert.Equal(t, Some(FrequencyNormal), freq(lens(40, 40, 28, 28, 28)), "last ≤ 3 cycles only")
	skipped := lens(28, 56)
	skipped.Skips = []Day{cur - 28}
	assert.Equal(t, Some(FrequencyNormal), freq(skipped), "skipped cycles excluded")

	decl := declared(Declared{LastPeriodStart: Some(cur), CycleLength: Some(40)})
	assert.Equal(t, Some(FrequencyInfrequent), freq(decl), "declared fallback")
	assert.False(t, freq(declared(Declared{LastPeriodStart: Some(cur)})).OK(), "no basis → null")

	dur := func(in Input) Opt[Duration] { return prep(in).Resolve(target, target).ClinicalFlags.Duration }
	assert.Equal(t, Some(DurationProlonged), dur(withPeriods(periodsEnding(cur, 9, 28, 28))), "[9,9,9]")
	assert.Equal(t, Some(DurationNormal), dur(withPeriods(periodsEnding(cur, 8, 28, 28))), "8 is not > 8")
	assert.False(t, dur(withPeriods([]Period{{Start: cur, End: NoDay}})).OK(), "none closed → null")
	assert.False(t, dur(none()).OK())
	assert.Equal(t, Some(DurationProlonged), dur(withPeriods(periodsEnding(cur, 12, 28))),
		"closed real periods count beyond the 10-day validity limit")
}

func TestQuality_Absent(t *testing.T) {
	cur := day(qualityCur)
	in := lens(28, 28)
	st := prep(in).Resolve(cur+91, cur+91)
	assert.Equal(t, Some(true), st.ClinicalFlags.Absent)
	assert.True(t, st.Warnings.Has(WarnLongAbsenceNoPeriod))

	st = prep(in).Resolve(cur+90, cur+90)
	assert.Equal(t, Some(true), st.ClinicalFlags.Absent, "≥ 90 days")
	st = prep(in).Resolve(cur+89, cur+89)
	assert.Equal(t, Some(false), st.ClinicalFlags.Absent)
	assert.False(t, st.Warnings.Has(WarnLongAbsenceNoPeriod))

	st = prep(in).Resolve(cur+200, cur+91)
	assert.Equal(t, Some(true), st.ClinicalFlags.Absent, "a future target reads absent at today")
	st = prep(in).Resolve(cur+91, cur+200)
	assert.Equal(t, Some(true), st.ClinicalFlags.Absent)
	st = prep(in).Resolve(cur+30, cur+200)
	assert.Equal(t, Some(false), st.ClinicalFlags.Absent, "a past target reads absent at the target")

	for _, m := range []Mode{ModePostpartum, ModePregnancy} {
		pm := lens(28, 28)
		pm.Mode = m
		st = prep(pm).Resolve(cur+91, cur+91)
		assert.Equal(t, Some(false), st.ClinicalFlags.Absent, m.String())
		assert.False(t, st.Warnings.Has(WarnLongAbsenceNoPeriod))
	}

	open := withPeriods([]Period{{Start: cur, End: NoDay}})
	assert.Equal(t, Some(false), prep(open).Resolve(cur+91, cur+91).ClinicalFlags.Absent, "an open period")

	decl := declared(Declared{LastPeriodStart: Some(cur)})
	assert.False(t, prep(decl).Resolve(cur+91, cur+91).ClinicalFlags.Absent.OK(), "no real period → null")
}

func TestQuality_DataQuality(t *testing.T) {
	cur := day(qualityCur)
	openIn := lens(28, 28)
	openIn.Periods = append(openIn.Periods, Period{Start: cur + 28, End: NoDay})
	manual := lens(29, 27, 30)
	manual.Declared = Declared{CycleLength: Some(29), PeriodLength: Some(5), Manual: true}
	contradictory := lens(28, 28)
	contradictory.Periods = append(contradictory.Periods, Period{Start: cur + 2, End: cur + 6})

	cases := []struct {
		name          string
		in            Input
		target, today Day
		want          DataQuality
	}{
		{"§2.35 example", lens(27, 29, 30), day("2026-07-18"), day("2026-07-18"), DataQualityGood},
		{"onboarding only", declared(Declared{LastPeriodStart: Some(cur), CycleLength: Some(28), PeriodLength: Some(5)}),
			cur + 10, cur + 10, DataQualityPartial},
		{"1 real cycle", lens(28), cur + 10, cur + 10, DataQualityPartial},
		{"open period day 3 (end involved, assumed)", openIn, cur + 30, cur + 30, DataQualityPartial},
		{"open period day 9 (past the warning cap)", openIn, cur + 36, cur + 36, DataQualityPoor},
		{"late 7", lens(28, 28), cur + 35, cur + 35, DataQualityGood},
		{"late 8", lens(28, 28), cur + 36, cur + 36, DataQualityPartial},
		{"late 15", lens(28, 28), cur + 43, cur + 43, DataQualityPoor},
		{"only outliers", lens(18, 50), cur + 10, cur + 10, DataQualityPoor},
		{"contradictions (E-02)", contradictory, cur + 10, cur + 10, DataQualityPoor},
		{"no anchor", none(), cur, cur, DataQualityInsufficient},
		{"manual lengths (E-20)", manual, cur + 15, cur + 15, DataQualityPartial},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want.String(), prep(c.in).Resolve(c.target, c.today).DataQuality.String())
		})
	}

	st := prep(contradictory).Resolve(cur+10, cur+10)
	assert.True(t, st.Warnings.Has(WarnContradictoryAnchorData))
	assert.True(t, st.RequiresUserInput)
	assert.Equal(t, ConfidenceLow, st.Confidence)

	// E-23: a history cycle length with a declared period length is never good.
	in := withPeriods(periodsEnding(cur, 1, 28, 28)) // 1-day periods: no valid period length
	in.Declared.PeriodLength = Some(5)
	assert.Equal(t, DataQualityPartial, prep(in).Resolve(cur+15, cur+15).DataQuality)
}

func TestQuality_Confidence(t *testing.T) {
	cur := day(qualityCur)
	t.Run("§2.35 example → high, no reasons", func(t *testing.T) {
		st := prep(lens(27, 29, 30)).Resolve(day("2026-07-18"), day("2026-07-18"))
		assert.Equal(t, ConfidenceHigh, st.Confidence)
		assert.Zero(t, st.ConfidenceReasons)
		assert.Zero(t, st.Warnings)
	})
	t.Run("age 46 perfect history → medium + perimenopause_age_range", func(t *testing.T) {
		target := day("2026-07-18")
		st := prep(aged(lens(27, 29, 30), target, 46)).Resolve(target, target)
		assert.Equal(t, ConfidenceMedium, st.Confidence)
		assert.True(t, st.Warnings.Has(WarnPerimenopauseAgeRange))
		assert.False(t, st.RequiresUserInput)
		st = prep(aged(lens(27, 29, 30), target, 44)).Resolve(target, target)
		assert.Equal(t, ConfidenceHigh, st.Confidence)
		assert.False(t, st.Warnings.Has(WarnPerimenopauseAgeRange))
		st = prep(aged(lens(27, 29, 30), target, 45)).Resolve(target, target)
		assert.Equal(t, ConfidenceMedium, st.Confidence, "≥ 45")
	})
	t.Run("variability 7 with bbt_confirmed → not low (E-19)", func(t *testing.T) {
		in := lens(28, 35, 28)
		in.BBT = shiftAt(cur + 14)
		st := prep(in).Resolve(cur+20, cur+20)
		require.Equal(t, OvulationBBTConfirmed, st.Anchors.OvulationSource)
		require.Equal(t, Some(7), st.Metrics.CycleVariability)
		assert.Equal(t, ConfidenceHigh, st.Confidence)
		assert.False(t, st.ConfidenceReasons.Has(ReasonHighCycleVariability))
		assert.True(t, st.Warnings.Has(WarnHighCycleVariability), "the data signal stays")
	})
	t.Run("variability 7 without BBT → low + high_cycle_variability", func(t *testing.T) {
		st := prep(lens(28, 35, 28)).Resolve(cur+20, cur+20)
		assert.Equal(t, ConfidenceLow, st.Confidence)
		assert.Equal(t, reasons(ReasonHighCycleVariability), st.ConfidenceReasons)
		assert.True(t, st.Warnings.Has(WarnHighCycleVariability))
	})
	t.Run("variability 5 → high", func(t *testing.T) {
		assert.Equal(t, ConfidenceHigh, prep(lens(28, 33, 28)).Resolve(cur+20, cur+20).Confidence)
	})
	t.Run("late 8 → low + period_overdue", func(t *testing.T) {
		st := prep(lens(28, 28)).Resolve(cur+36, cur+36)
		assert.Equal(t, ConfidenceLow, st.Confidence)
		assert.Equal(t, reasons(ReasonPeriodOverdue), st.ConfidenceReasons)
	})
	t.Run("late 1..7 → medium (period_expected is never high)", func(t *testing.T) {
		st := prep(lens(28, 28)).Resolve(cur+29, cur+29)
		assert.Equal(t, ConfidenceMedium, st.Confidence)
		assert.Equal(t, reasons(ReasonPeriodOverdue), st.ConfidenceReasons)
		assert.Equal(t, ConfidenceMedium, prep(lens(28, 28)).Resolve(cur+28, cur+28).Confidence, "late 0")
	})
	t.Run("menstrual day of a confirmed period → high", func(t *testing.T) {
		st := prep(lens(28, 28)).Resolve(cur+2, cur+2)
		assert.Equal(t, ResolutionUserLogged, st.ResolutionSource)
		assert.Equal(t, ConfidenceHigh, st.Confidence)
	})
	t.Run("postpartum with < 2 real periods after delivery → low", func(t *testing.T) {
		in := lens(28, 28, 28)
		in.Mode = ModePostpartum
		in.DeliveryDate = Some(cur - 20)
		assert.Equal(t, ConfidenceLow, prep(in).Resolve(cur+20, cur+20).Confidence, "1 period after delivery")
		in.DeliveryDate = Some(cur - 30)
		assert.Equal(t, ConfidenceHigh, prep(in).Resolve(cur+20, cur+20).Confidence, "2 periods after delivery")
		assert.Equal(t, ConfidenceLow, prep(in).Resolve(cur-10, cur+20).Confidence, "1 period by the target")
		in.Mode, in.Breastfeeding = ModeCycle, true
		in.DeliveryDate = Some(cur - 20)
		assert.Equal(t, ConfidenceLow, prep(in).Resolve(cur+20, cur+20).Confidence, "breastfeeding (E-14)")
		in.Breastfeeding = false
		assert.Equal(t, ConfidenceHigh, prep(in).Resolve(cur+20, cur+20).Confidence, "cycle mode: no rule")
	})
	t.Run("onboarding and default resolutions → low", func(t *testing.T) {
		d := declared(Declared{LastPeriodStart: Some(cur), CycleLength: Some(28), PeriodLength: Some(5)})
		assert.Equal(t, ConfidenceLow, prep(d).Resolve(cur+10, cur+10).Confidence)
		d = declared(Declared{LastPeriodStart: Some(cur)})
		assert.Equal(t, ConfidenceLow, prep(d).Resolve(cur+10, cur+10).Confidence)
	})
	t.Run("no anchor → unknown", func(t *testing.T) {
		assert.Equal(t, ConfidenceUnknown, prep(none()).Resolve(cur, cur).Confidence)
	})
}

// Each of the 14 reasons from one fixture, alone where the rules allow it.
func TestQuality_EveryReason(t *testing.T) {
	cur := day(qualityCur)
	openIn := lens(28, 28)
	openIn.Periods = append(openIn.Periods, Period{Start: cur + 28, End: NoDay})
	shortPeriods := withPeriods(periodsEnding(cur, 1, 28, 28)) // no valid period length
	withDeclPeriod := shortPeriods
	withDeclPeriod.Declared.PeriodLength = Some(5)
	contradictory := lens(28, 28)
	contradictory.Periods = append(contradictory.Periods, Period{Start: cur + 2, End: cur + 6})
	hormonal := lens(28, 28)
	hormonal.Hormonal = true

	cases := []struct {
		reason        Reason
		in            Input
		target, today Day
		want          ReasonSet // the whole set
	}{
		{ReasonLowHistory, lens(28), cur + 20, cur + 20, reasons(ReasonLowHistory)},
		{ReasonHighCycleVariability, lens(28, 35, 28), cur + 20, cur + 20, reasons(ReasonHighCycleVariability)},
		{ReasonMissingPeriodEnd, openIn, cur + 30, cur + 30, reasons(ReasonMissingPeriodEnd)},
		{ReasonMissingPeriodEndBeyondWarningCap, openIn, cur + 36, cur + 36,
			reasons(ReasonMissingPeriodEnd, ReasonMissingPeriodEndBeyondWarningCap, ReasonPoorDataQuality)},
		{ReasonMissingPeriodEndBeyondHardCap, openIn, cur + 40, cur + 40,
			reasons(ReasonMissingPeriodEnd, ReasonMissingPeriodEndBeyondWarningCap,
				ReasonMissingPeriodEndBeyondHardCap, ReasonPoorDataQuality)},
		{ReasonOnboardingDataUsed, withDeclPeriod, cur + 20, cur + 20, reasons(ReasonOnboardingDataUsed)},
		{ReasonDefaultValuesUsed, shortPeriods, cur + 20, cur + 20, reasons(ReasonDefaultValuesUsed)},
		{ReasonPeriodOverdue, lens(28, 28), cur + 30, cur + 30, reasons(ReasonPeriodOverdue)},
		{ReasonUnresolvedCycle, lens(28, 28), cur + 43, cur + 43,
			reasons(ReasonPeriodOverdue, ReasonUnresolvedCycle, ReasonPoorDataQuality)},
		{ReasonPoorDataQuality, lens(18, 50), cur + 20, cur + 20, reasons(ReasonLowHistory, ReasonDefaultValuesUsed, ReasonPoorDataQuality)},
		{ReasonInsufficientAnchorData, none(), cur, cur, reasons(ReasonInsufficientAnchorData)},
		{ReasonHormonalContraceptionActive, hormonal, cur + 20, cur + 20, reasons(ReasonHormonalContraceptionActive)},
	}
	seen := reasons()
	for _, c := range cases {
		t.Run(c.reason.String(), func(t *testing.T) {
			st := prep(c.in).Resolve(c.target, c.today)
			assert.Equal(t, fmt.Sprint(reasonNames(c.want)), fmt.Sprint(reasonNames(st.ConfidenceReasons)))
		})
		seen.Add(c.reason)
	}

	// Contradictions: poor + insufficient_anchor_data (v1.1 parity; there is no contradiction reason).
	st := prep(contradictory).Resolve(cur+20, cur+20)
	assert.Equal(t, reasons(ReasonPoorDataQuality, ReasonInsufficientAnchorData), st.ConfidenceReasons)

	// prediction_based_output and stale_onboarding_anchor come from T-M9-20 (future projection, §2.41); the
	// quality layer maps their warnings / the predicted_reference start.
	base := prep(lens(28, 28))
	for _, c := range []struct {
		reason Reason
		set    func(*Status)
	}{
		{ReasonPredictionBasedOutput, func(st *Status) { st.Warnings.Add(WarnPredictionBasedOutput) }},
		{ReasonPredictionBasedOutput, func(st *Status) {
			st.Anchors.CurrentPeriodStartSource = StartPredictedReference
		}},
		{ReasonStaleOnboardingAnchor, func(st *Status) { st.Warnings.Add(WarnStaleOnboardingAnchor) }},
	} {
		st, f := base.resolve(cur+20, cur+20)
		c.set(&st)
		base.grade(&st, f, cur+20)
		assert.True(t, st.ConfidenceReasons.Has(c.reason), c.reason.String())
		seen.Add(c.reason)
	}
	st, f := base.resolve(cur+20, cur+20)
	st.Anchors.CurrentPeriodStartSource = StartPredictedReference
	base.grade(&st, f, cur+20)
	assert.Equal(t, ConfidenceLow, st.Confidence, "§2.29: a predicted_reference start is low")

	assert.Equal(t, int(reasonCount), seen.Len(), "every reason has a fixture")
}

func reasonNames(s ReasonSet) []string {
	var out []string
	for r := range s.All() {
		out = append(out, r.String())
	}
	return out
}

func TestQuality_SignalWarnings(t *testing.T) {
	cur := day(qualityCur)
	has := func(in Input, w Warning) bool { return prep(in).Resolve(cur+20, cur+20).Warnings.Has(w) }

	assert.True(t, has(lens(28), WarnLowHistory))
	assert.False(t, has(lens(28, 28), WarnLowHistory))
	assert.True(t, has(lens(28, 18, 28), WarnShortCycleOutlierDetected))
	assert.False(t, has(lens(28, 18, 28), WarnLongCycleOutlierDetected))
	assert.True(t, has(lens(28, 50, 28), WarnLongCycleOutlierDetected))
	assert.False(t, has(lens(28, 50, 28), WarnShortCycleOutlierDetected))

	skipped := lens(28, 56, 28)
	skipped.Skips = []Day{cur - 28 - 28}
	assert.False(t, has(skipped, WarnLongCycleOutlierDetected), "a skipped cycle is never an outlier (E-15)")
	assert.False(t, prep(skipped).Resolve(cur+20, cur+20).DataQuality == DataQualityPoor)

	hormonal := lens(28, 28)
	hormonal.Hormonal = true
	assert.True(t, has(hormonal, WarnHormonalContraceptionActive))
	assert.Equal(t, ConfidenceLow, prep(hormonal).Resolve(cur+20, cur+20).Confidence)

	// No anchor: only insufficient_anchor_data (data signals need a resolvable cycle).
	var want WarningSet
	want.Add(WarnInsufficientAnchorData)
	assert.Equal(t, want, prep(none()).Resolve(cur, cur).Warnings)
}

// requires_user_input is true exactly for the six §2.33 warnings (incl. contradictory_anchor_data, E-02).
func TestQuality_RequiresUserInput(t *testing.T) {
	cur := day(qualityCur)
	s := prep(lens(28, 28))
	six := map[Warning]bool{
		WarnPeriodEndMissingWarningCapExceeded: true, WarnPeriodEndMissingHardCapExceeded: true,
		WarnCycleUnresolvedAfterExpectedPeriod: true, WarnInsufficientAnchorData: true,
		WarnContradictoryAnchorData: true, WarnStaleOnboardingAnchor: true,
	}
	for w := range warningCount {
		st, f := s.resolve(cur+20, cur+20)
		require.Zero(t, st.Warnings)
		st.Warnings.Add(w)
		s.grade(&st, f, cur+20)
		assert.Equal(t, six[w], st.RequiresUserInput, w.String())
	}
}

func TestQuality_ZeroEndIsOpen(t *testing.T) {
	cur := day(qualityCur)
	in := lens(28, 28)
	in.Periods = append(in.Periods, Period{Start: cur + 28}) // End: Day(0) — the loader trap
	s := prep(in)
	assert.False(t, s.Contradictory())
	require.Len(t, s.Periods(), 4)
	assert.True(t, s.Periods()[3].Open())
}

func TestQuality_NoAllocs(t *testing.T) {
	cur := day(qualityCur)
	in := aged(lens(28, 29, 27, 30, 28, 18, 28), cur, 46)
	in.Periods = append(in.Periods, Period{Start: cur + 29, End: NoDay})
	in.Mode, in.DeliveryDate = ModePostpartum, Some(cur-300)
	in.BBT = shiftAt(cur + 14)
	s := prep(in)
	var st Status
	targets := []Day{cur + 3, cur + 12, cur + 20, cur + 35, cur + 60, cur + 200, cur - 30, cur - 400}
	allocs := testing.AllocsPerRun(200, func() {
		for _, t := range targets {
			st = s.Resolve(t, cur+30)
		}
	})
	assert.Zero(t, allocs)
	_ = st
}
