package engine

import (
	"slices"
	"time"
)

// The quality layer (§2.27–§2.29, §2.32–§2.33): cycle_regularity, clinical_flags, the data-signal warnings,
// data_quality, confidence and confidence_reasons. The today-independent parts are prepared once
// (prepareQuality); the parts that depend on the target or today (age, the 12-month regularity window, absent) are
// computed in grade without allocating.

// prepareQuality is Prepare part 3: the clinical_flags bases (E-06, E-07) and the outlier signals (§2.7, §2.28).
func (s *State) prepareQuality() {
	w := max(s.p.HistoryWindow, 1)
	var buf [8]int

	// E-06: median of the last ≤ w real complete cycles, outliers included, skipped cycles excluded; else the
	// declared length (ignored outside the supported range, as for effective_cycle_length); else none.
	nonSkipped := 0
	lens := buf[:0]
	for i := len(s.cycles) - 1; i >= 0; i-- {
		c := s.cycles[i]
		if c.Skipped {
			continue
		}
		nonSkipped++
		s.shortOutlier = s.shortOutlier || c.Short
		s.longOutlier = s.longOutlier || c.Long
		if len(lens) < w {
			lens = append(lens, c.Length)
		}
	}
	switch dv, ok := s.in.Declared.CycleLength.Get(); {
	case len(lens) > 0:
		s.freqBasis = Some(median(lens))
	case ok && dv >= s.p.CycleMin && dv <= s.p.CycleMax:
		s.freqBasis = Some(dv)
	}
	// §2.28 poor: "only outlier history" — real cycles exist and none of them is valid.
	s.onlyOutliers = nonSkipped > 0 && s.validCycles == 0

	// E-07 duration: median of the last ≤ w closed real periods > FIGO max → prolonged; none closed → null.
	per := buf[:0]
	for i := len(s.periods) - 1; i >= 0 && len(per) < w; i-- {
		if n := periodLength(s.periods[i]); n > 0 {
			per = append(per, n)
		}
	}
	if len(per) > 0 {
		s.duration = Some(DurationNormal)
		if median(per) > s.p.FIGO.DurationMax {
			s.duration = Some(DurationProlonged)
		}
	}
}

// ageAt is the age in whole years on d (E-27); ok is false without a birth date or for a birth date after d.
func (s *State) ageAt(d Day) (int, bool) {
	if s.birth == NoDay || d == NoDay || d < s.birth {
		return 0, false
	}
	by, bm, bd := civilFromDays(int64(s.birth))
	y, m, dd := civilFromDays(int64(d))
	age := y - by
	if m < bm || (m == bm && dd < bd) {
		age--
	}
	return age, true
}

// teenAt reports the teen rules (E-08, E-27): age < TeenAge on d, or only a teen age band is known.
func (s *State) teenAt(d Day) bool {
	if age, ok := s.ageAt(d); ok {
		return age < s.p.TeenAge
	}
	return s.in.TeenBandOnly
}

// monthsBefore is d minus n calendar months, the day clamped to the target month's length (03-31 − 1 → 02-28).
func monthsBefore(d Day, n int) Day {
	y, m, dd := civilFromDays(int64(d))
	total := y*12 + (m - 1) - n
	ny, nm := total/12, total%12
	if nm < 0 {
		ny, nm = ny-1, nm+12
	}
	if ny < 0 {
		return NoDay
	}
	nm++
	nd := min(dd, daysIn(ny, time.Month(nm)))
	return Day(daysFromCivil(ny, nm, nd)) //nolint:gosec // years 0..9999: |days| < 2^22
}

// regularity is cycle_regularity at ref (§2.27, E-08, E-09): the real complete cycles (next start ≤ ref) whose start
// lies in the RegularityMonths calendar months before ref; skipped cycles excluded, outliers included; at least
// RegularityMinCycles of them; max − min against the FIGO band of the age at ref (< AdultFrom → young, ≥ LateFrom
// → late, unknown age → adult; a teen band without a birth date → young).
func (s *State) regularity(ref Day) CycleRegularity {
	from := monthsBefore(ref, s.p.RegularityMonths)
	if from == NoDay {
		return RegularityInsufficientData
	}
	i, _ := slices.BinarySearchFunc(s.cycles, from, func(c Cycle, d Day) int { return int(c.Start) - int(d) })
	n, lo, hi := 0, 0, 0
	for ; i < len(s.cycles) && s.cycles[i].NextStart <= ref; i++ {
		c := s.cycles[i]
		if c.Skipped {
			continue
		}
		if n == 0 {
			lo, hi = c.Length, c.Length
		}
		lo, hi = min(lo, c.Length), max(hi, c.Length)
		n++
	}
	if n < s.p.RegularityMinCycles {
		return RegularityInsufficientData
	}
	limit := s.p.FIGO.RegularityAdult
	if age, ok := s.ageAt(ref); ok {
		switch {
		case age < s.p.RegularityAdultFromAge:
			limit = s.p.FIGO.RegularityYoung
		case age >= s.p.RegularityLateFromAge:
			limit = s.p.FIGO.RegularityLate
		}
	} else if s.in.TeenBandOnly {
		limit = s.p.FIGO.RegularityYoung
	}
	if hi-lo > limit {
		return RegularityIrregular
	}
	return RegularityRegular
}

// clinicalFlags is the §2.27 block for target (age rules at the target, E-27) with absent at ref (E-07).
func (s *State) clinicalFlags(target, ref Day) ClinicalFlags {
	f := ClinicalFlags{Duration: s.duration}
	if n, ok := s.freqBasis.Get(); ok {
		band := s.p.FIGO.Frequency
		if s.teenAt(target) {
			band = s.p.TeenFrequency
		}
		switch {
		case n < band.Min:
			f.Frequency = Some(FrequencyFrequent)
		case n > band.Max:
			f.Frequency = Some(FrequencyInfrequent)
		default:
			f.Frequency = Some(FrequencyNormal)
		}
	}
	f.Absent = s.absentAt(ref)
	return f
}

// absentAt is clinical_flags.absent (E-07): null without a real period on or before ref; true when the last real
// start ≤ ref is ≥ AbsentDays before ref, that period is closed, and the mode is neither pregnancy nor postpartum.
func (s *State) absentAt(ref Day) Opt[bool] {
	i := s.lastRealAtOrBefore(ref)
	if i < 0 {
		return None[bool]()
	}
	p := s.periods[i]
	return Some(!p.Open() && int(ref-p.Start) >= s.p.AbsentDays &&
		s.in.Mode != ModePregnancy && s.in.Mode != ModePostpartum)
}

// lastRealAtOrBefore is the index of the latest real start ≤ d, -1 when none.
func (s *State) lastRealAtOrBefore(d Day) int {
	i, found := slices.BinarySearchFunc(s.periods, d, func(p Period, d Day) int { return int(p.Start) - int(d) })
	if found {
		return i
	}
	return i - 1
}

// highVariability is cycle_variability > HighVariabilityDays.
func (s *State) highVariability() bool {
	v, ok := s.variability.Get()
	return ok && v > s.p.HighVariabilityDays
}

// grade is the quality layer for one resolved status: cycle_regularity, clinical_flags, the data-signal warnings
// (§2.32), data_quality (§2.28), confidence (§2.29, E-19, E-27) and its reasons, requires_user_input (§2.33).
func (s *State) grade(st *Status, f facts, today Day) {
	p := &s.p
	ref := min(st.Date, today)
	st.Metrics.CycleRegularity = s.regularity(ref)
	st.ClinicalFlags = s.clinicalFlags(st.Date, ref)

	if !f.anchored { // §2.28 insufficient, §2.29 unknown
		st.DataQuality = DataQualityInsufficient
		st.Confidence = ConfidenceUnknown
		if st.Warnings.Has(WarnInsufficientAnchorData) { // not raised by a suspended or postpartum status
			st.ConfidenceReasons.Add(ReasonInsufficientAnchorData)
		}
		s.limitGrade(st)
		return
	}

	w := &st.Warnings
	highVar := s.highVariability()
	bbtCycle := st.Anchors.OvulationSource == OvulationBBTConfirmed
	age, hasAge := s.ageAt(st.Date)
	perimenopause := hasAge && age >= p.PerimenopauseAge
	if s.validCycles < 2 {
		w.Add(WarnLowHistory)
	}
	if highVar {
		w.Add(WarnHighCycleVariability)
	}
	if s.shortOutlier {
		w.Add(WarnShortCycleOutlierDetected)
	}
	if s.longOutlier {
		w.Add(WarnLongCycleOutlierDetected)
	}
	if s.contradictory {
		w.Add(WarnContradictoryAnchorData)
	}
	hormonal := s.hormonalAt(st.Date) // E-13 only from the method's start (T-M9-48d)
	if hormonal {
		w.Add(WarnHormonalContraceptionActive)
	}
	if perimenopause {
		w.Add(WarnPerimenopauseAgeRange)
	}
	if a, ok := st.ClinicalFlags.Absent.Get(); ok && a {
		w.Add(WarnLongAbsenceNoPeriod)
	}

	daysLate := st.DaysLate.Or(0)
	startSrc := st.Anchors.CurrentPeriodStartSource
	endOK := !f.endInvolved || st.Anchors.CurrentPeriodEndIsConfirmed
	res := st.ResolutionSource
	declaredRes := res == ResolutionOnboardingBased || res == ResolutionDefaultBased
	// An unknown phase under hormonal contraception (E-13) or postpartum not returned (E-14) is a deliberate
	// suppression, not missing data: it counts as resolved for §2.28 good (T-M9-48c). A day that would be unknown
	// anyway (E-11, the defensive rule) is forced poor above it.
	sup := st.Limits.SuppressionReason
	phaseResolved := st.MainPhase != MainPhaseUnknown ||
		sup == SuppressionHormonalContraception || sup == SuppressionPostpartumNotReturned

	// §2.28: insufficient → poor → good → partial (E-02 contradiction → poor; E-20 + E-23 sources must be history).
	switch {
	case f.forcedPoor || daysLate > p.MaxLateDays || s.contradictory || s.onlyOutliers:
		st.DataQuality = DataQualityPoor
	case startSrc == StartUserLogged && s.validCycles >= 2 && s.cycleSrc == LengthHistory &&
		s.periodSrc == LengthHistory && phaseResolved && daysLate < p.OverdueDaysLate && endOK:
		st.DataQuality = DataQualityGood
	default:
		st.DataQuality = DataQualityPartial
	}

	// §2.29: unknown → low → high → medium; age ≥ PerimenopauseAge caps at medium (E-27).
	varLow := highVar && !bbtCycle // E-19
	switch {
	case res == ResolutionUnknown:
		st.Confidence = ConfidenceUnknown
	case f.forcedLow || st.MainPhase == MainPhaseUnknown || daysLate >= p.OverdueDaysLate ||
		st.DataQuality == DataQualityPoor || s.validCycles < 2 || varLow || declaredRes ||
		startSrc == StartPredictedReference || hormonal:
		st.Confidence = ConfidenceLow
	case st.DataQuality == DataQualityGood && s.validCycles >= 2 && s.validPeriods >= 2 &&
		startSrc == StartUserLogged && endOK && (!highVar || bbtCycle) &&
		st.MainPhase != MainPhasePeriodExpected && st.MainPhase != MainPhaseUnknown && !declaredRes:
		st.Confidence = ConfidenceHigh
	default:
		st.Confidence = ConfidenceMedium
	}
	if f.futureK > 0 && res != ResolutionUnknown {
		// E-03: a projection takes today's confidence, k ≥ 2 at most medium; the predicted_reference low and the
		// day's own data_quality do not apply to it.
		st.Confidence = f.futureCap
	}
	if perimenopause {
		st.Confidence = min(st.Confidence, ConfidenceMedium)
	}

	// confidence_reasons: every active cause (§2.29), in enum order.
	r := &st.ConfidenceReasons
	add := func(cond bool, reason Reason) {
		if cond {
			r.Add(reason)
		}
	}
	add(s.validCycles < 2, ReasonLowHistory)
	add(varLow, ReasonHighCycleVariability)
	add(f.realOpen, ReasonMissingPeriodEnd)
	add(w.Has(WarnPeriodEndMissingWarningCapExceeded), ReasonMissingPeriodEndBeyondWarningCap)
	add(w.Has(WarnPeriodEndMissingHardCapExceeded), ReasonMissingPeriodEndBeyondHardCap)
	add(w.Has(WarnOnboardingDataUsed), ReasonOnboardingDataUsed)
	add(w.Has(WarnDefaultValuesUsed), ReasonDefaultValuesUsed)
	add(w.Has(WarnPredictionBasedOutput) || startSrc == StartPredictedReference, ReasonPredictionBasedOutput)
	add(daysLate > 0, ReasonPeriodOverdue)
	add(w.Has(WarnCycleUnresolvedAfterExpectedPeriod), ReasonUnresolvedCycle)
	add(st.DataQuality == DataQualityPoor, ReasonPoorDataQuality)
	add(w.Has(WarnInsufficientAnchorData) || s.contradictory, ReasonInsufficientAnchorData)
	add(w.Has(WarnStaleOnboardingAnchor), ReasonStaleOnboardingAnchor)
	add(hormonal, ReasonHormonalContraceptionActive)

	s.limitGrade(st)
}

// limitGrade applies the limiting conditions to the graded status (§2.42) and sets requires_user_input (§2.33).
//   - Pregnancy (E-25): the engine is suspended — confidence unknown, no warnings or reasons (nothing to act on, so
//     requires_user_input is false), data_quality insufficient (there is no cycle to grade).
//   - Postpartum not returned (E-14): confidence unknown before the first real period after delivery and low from it
//     on — the documented exception to §2.29, which lists only low.
//   - Hormonal contraception (E-13) is already low through §2.29.
func (s *State) limitGrade(st *Status) {
	switch st.Limits.SuppressionReason {
	case SuppressionPregnancy:
		st.Warnings, st.ConfidenceReasons = 0, 0
		st.Confidence = ConfidenceUnknown
		st.DataQuality = DataQualityInsufficient
	case SuppressionPostpartumNotReturned:
		st.Confidence = ConfidenceUnknown
		if n, _ := s.postpartumPeriods(st.Date); n > 0 {
			st.Confidence = ConfidenceLow
		}
	}
	st.RequiresUserInput = st.Warnings.RequiresUserInput()
}
