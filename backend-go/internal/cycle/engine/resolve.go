package engine

// anchor is the cycle a target falls in (§2.10): its start, where the start comes from, the logged end of that
// period (NoDay = open, or a declared start) and the next real start after it (NoDay for the current cycle;
// set = a completed cycle laid out retrospectively, E-22).
type anchor struct {
	start     Day
	src       StartSource
	loggedEnd Day
	nextReal  Day
	idx       int // index of the real period in State.periods; -1 for a declared start
}

// anchorAt finds the anchor of target: the latest real start ≤ target by binary search (O(log H)); with no real
// start ≤ target, the declared onboarding start when it is ≤ target (I-01: a declared date, never a real period).
func (s *State) anchorAt(target Day) (anchor, bool) {
	ps := s.periods
	lo, hi := 0, len(ps)
	for lo < hi { // lo = number of real starts ≤ target
		m := int(uint(lo+hi) >> 1) //nolint:gosec // lo+hi ≥ 0
		if ps[m].Start <= target {
			lo = m + 1
		} else {
			hi = m
		}
	}
	a := anchor{loggedEnd: NoDay, nextReal: NoDay, idx: -1}
	if lo < len(ps) {
		a.nextReal = ps[lo].Start
	}
	if lo > 0 {
		p := ps[lo-1]
		a.start, a.src, a.loggedEnd, a.idx = p.Start, StartUserLogged, p.End, lo-1
		return a, true
	}
	if d := s.lmp; d != NoDay && d <= target {
		a.start, a.src = d, StartOnboardingDeclared
		return a, true
	}
	return anchor{}, false
}

// layoutFor is the day map of the anchor's cycle: the end model §2.11, the ovulation §2.17 and predicted_next
// §2.16. A real cycle with a BBT or LH ovulation uses it (BBT > LH, E-16); the current cycle then predicts next =
// O + effective_luteal_length, a completed cycle keeps its actual next start (E-22). Otherwise next = start +
// effective_cycle_length (or the actual next start) and the calendar ovulation is next − Luteal.Default. Under
// fertility suppression (markers = false) no ovulation is computed (§2.42), so BBT/LH never move next.
func (s *State) layoutFor(a anchor, markers bool) layout {
	end, confirmed := a.loggedEnd, a.loggedEnd != NoDay
	if !confirmed {
		end = a.start + days(max(1, s.periodLen)) - 1
	}
	next := a.nextReal
	if markers && a.idx >= 0 {
		if o := s.ovulations[a.idx]; o.Source != OvulationUnknown {
			if next == NoDay {
				next = o.Date + days(s.luteal)
			}
			l := newLayout(a.start, end, confirmed, next, o.Date, s.p.FertileBefore)
			l.ovSrc, l.conflict = o.Source, o.Conflict
			return l
		}
	}
	if next == NoDay {
		next = a.start + days(max(1, s.cycleLen))
	}
	return newLayout(a.start, end, confirmed, next, next-days(s.p.Luteal.Default), s.p.FertileBefore)
}

// facts are the resolve-time signals the quality layer (grade, quality.go) reads besides the Status itself.
type facts struct {
	anchored    bool
	forcedPoor  bool // a tier forces data_quality = poor
	forcedLow   bool // a tier forces confidence = low
	realOpen    bool // the anchor is a real period with no logged end (confidence_reasons += missing_period_end)
	endInvolved bool // the target's phase depends on current_period_end (menstrual, follicular, a window at end + 1)
}

// Resolve is the engine status of one target date (§2.20–§2.34, §2.42) for the current cycle and the history. today
// is the user-local day; the future-cycle projection of targets after today (E-03) and the stale anchor and skips
// (§2.41, E-15) arrive in T-M9-20. It allocates nothing but the returned value.
func (s *State) Resolve(target, today Day) Status {
	st, f := s.resolve(target, today)
	s.grade(&st, f, today)
	return st
}

func (s *State) resolve(target, _ Day) (Status, facts) {
	p := &s.p
	st := Status{
		Date: target,
		Metrics: Metrics{
			EffectiveCycleLength:  s.cycleLen,
			EffectivePeriodLength: s.periodLen,
			CycleVariability:      s.variability,
		},
		Anchors: Anchors{CurrentPeriodStart: NoDay, CurrentPeriodEnd: NoDay, PredictedNextPeriodStart: NoDay,
			EstimatedOvulationDate: NoDay},
	}
	var f facts

	sup := s.suppressionAt(target)
	// E-14 / E-25: no cycle is predicted at all (only real period days resolve); hormonal (E-13) keeps the period
	// predictions and drops the fertile/ovulation layer only.
	noPrediction := sup == SuppressionPregnancy || sup == SuppressionPostpartumNotReturned
	if sup == SuppressionPostpartumNotReturned {
		st.Warnings.Add(WarnPostpartumCyclesNotReturned)
	}

	a, ok := s.anchorAt(target)
	if !ok {
		st.Limits = s.limitsFor(target, NoDay, sup)
		// §2.20.6 / §2.28 insufficient: no real start and no declared start on or before the target. A suspended or
		// postpartum status is complete without one (E-14, E-25: requires_user_input = false).
		if !noPrediction {
			st.Warnings.Add(WarnInsufficientAnchorData)
		}
		return st, f
	}
	f.anchored = true
	st.Limits = s.limitsFor(target, a.start, sup)
	l := s.layoutFor(a, sup == SuppressionNone)

	cycleDay := int(target-a.start) + 1 // §2.5, 1-based
	raw := int(l.next - target)         // §2.22
	daysLate := max(0, -raw)

	st.CycleDay = Some(cycleDay)
	st.DaysToPeriod = Some(max(0, raw))
	st.DaysLate = Some(daysLate)
	st.Anchors = Anchors{
		CurrentPeriodStart:          a.start,
		CurrentPeriodStartSource:    a.src,
		CurrentPeriodEnd:            l.end,
		CurrentPeriodEndSource:      EndAssumedFromEffectiveDuration,
		CurrentPeriodEndIsConfirmed: l.endConfirmed,
		PredictedNextPeriodStart:    l.next,
		EstimatedOvulationDate:      l.ov,
		OvulationSource:             l.ovSrc,
	}
	if l.endConfirmed {
		st.Anchors.CurrentPeriodEndSource = EndUserLogged
	}
	if sup != SuppressionNone { // §2.42: ovulation is not computed
		st.Anchors.EstimatedOvulationDate, st.Anchors.OvulationSource = NoDay, OvulationUnknown
	}
	if noPrediction {
		st.Anchors.PredictedNextPeriodStart = NoDay
		st.DaysToPeriod, st.DaysLate = None[int](), None[int]()
		daysLate = 0
	}
	if l.conflict { // §2.17: every target of that cycle (never under suppression: the markers are not read)
		st.Warnings.Add(WarnLHBBTConflict)
	}

	// 1. menstrual (§2.21): logged bleeding resolves under every limiting condition (E-13, E-14, E-25).
	sub, res, decided := s.menstrual(&st, &f, a, &l, cycleDay)

	// E-14 / E-25: every other day is unknown with nothing predicted.
	if !decided && noPrediction {
		decided, sub, res = true, SubphaseUnknown, ResolutionUnknown
	}

	// 2. period_expected (§2.22) — never inside a completed cycle (E-22); also under hormonal contraception (E-13).
	unresolvedLate := false
	if !decided && a.nextReal == NoDay && target >= l.next {
		decided, res = true, ResolutionPrediction
		switch {
		case daysLate < p.OverdueDaysLate:
			sub = SubphasePeriodExpected
		case daysLate <= p.MaxLateDays:
			sub = SubphasePeriodExpected
			st.Warnings.Add(WarnPredictedPeriodOverdue)
			f.forcedLow = true
		default: // E-11: unknown main phase, still resolution prediction; anchors and days_late stay.
			sub, unresolvedLate = SubphaseUnknown, true
			st.Warnings.Add(WarnPredictedPeriodOverdue)
			st.Warnings.Add(WarnCycleUnresolvedAfterExpectedPeriod)
			f.forcedLow, f.forcedPoor = true, true
		}
	}

	// E-13: under hormonal contraception every other day is unknown — still a prediction (cycle day and the next
	// withdrawal bleed are estimated), never "insufficient data".
	if !decided && sup == SuppressionHormonalContraception {
		decided, sub, res = true, SubphaseUnknown, ResolutionPrediction
	}

	// 3–5. display fertile window, luteal, follicular (§2.23–§2.25).
	if !decided {
		if sub = l.cyclePhase(target, p); sub != SubphaseUnknown {
			decided, res = true, ResolutionPrediction
		}
	}

	// 6. unknown: no rule covers the day (defensive; unreachable with a consistent layout).
	if !decided {
		res = ResolutionUnknown
		st.Warnings.Add(WarnInsufficientAnchorData)
		f.forcedPoor = true
	}

	// §2.30 / E-10: a declared onboarding anchor resolves every phase as onboarding_based (default_based when both
	// lengths are defaults), period_expected days 0..max_late included; > max_late stays prediction (E-11).
	if a.src == StartOnboardingDeclared && res != ResolutionUnknown && !unresolvedLate {
		res = ResolutionOnboardingBased
		if s.cycleSrc == LengthDefault && s.periodSrc == LengthDefault {
			res = ResolutionDefaultBased
		}
	}

	// §2.32 source warnings. Manual lengths (E-20) are not onboarding data.
	d := s.in.Declared
	if a.src == StartOnboardingDeclared ||
		(!d.Manual && (s.cycleSrc == LengthDeclared || s.periodSrc == LengthDeclared)) {
		st.Warnings.Add(WarnOnboardingDataUsed)
	}
	if s.cycleSrc == LengthDefault || s.periodSrc == LengthDefault {
		st.Warnings.Add(WarnDefaultValuesUsed)
	}
	// prediction_based_output: only for a predicted_reference anchor or a future projection (E-01) — neither
	// exists before T-M9-20, so ordinary prediction days carry no warning (§2.35 example).

	st.Subphase = sub
	st.MainPhase = sub.Main()
	f.realOpen = a.src == StartUserLogged && !l.endConfirmed
	f.endInvolved = st.MainPhase == MainPhaseMenstrual || st.MainPhase == MainPhaseFollicular ||
		(st.MainPhase == MainPhaseFertile && l.fertileStart == l.end+1)
	st.ResolutionSource = res
	if sup != SuppressionNone { // §2.26 / §2.34: suppressed fertility is unknown and has no countdown
		st.FertilityLevel = FertilityUnknown
		if sup == SuppressionPregnancy && st.MainPhase != MainPhaseMenstrual {
			st.CycleDay = None[int]() // E-25: no cycle runs during a pregnancy; logged history keeps its day
		}
		return st, f
	}
	st.FertilityLevel = fertilityOf(sub)
	if daysLate <= p.MaxLateDays { // §2.34
		st.DaysToOvulation = Some(int(l.ov - target))
	}
	st.SuggestPregnancyTest = s.suggestPregnancyTest(daysLate)
	return st, f
}

// menstrual is priority 1 of §2.20 with the §2.21 tiers and the §2.12 caps (soft = EPL, warning = max(EPL + extra,
// min), hard). decided is false when the day is not menstrual and resolves onward. The hard cap is checked before
// the warning cap, so a warning cap above the hard cap (EPL ≥ 10) never keeps a day menstrual past the hard cap.
func (s *State) menstrual(st *Status, f *facts, a anchor, l *layout, cycleDay int) (Subphase, ResolutionSource, bool) {
	p := &s.p
	soft := s.periodLen
	warning := max(s.periodLen+p.WarningCapExtra, p.WarningCapMin)
	switch {
	case l.endConfirmed: // 21.1
		if st.Date <= l.end {
			return SubphaseMenstrual, ResolutionUserLogged, true
		}
	case a.src == StartUserLogged: // a real open period: 21.2–21.5
		switch {
		case cycleDay <= soft:
			return SubphaseMenstrual, ResolutionUserLoggedWithAssumedEnd, true
		case cycleDay > p.PeriodHardCap: // 21.5: resolves onward; warnings cumulative (E-12)
			st.Warnings.Add(WarnPeriodEndMissing)
			st.Warnings.Add(WarnPeriodEndMissingWarningCapExceeded)
			st.Warnings.Add(WarnPeriodEndMissingHardCapExceeded)
			f.forcedPoor, f.forcedLow = true, true
		case cycleDay <= warning: // 21.3
			st.Warnings.Add(WarnPeriodEndMissing)
			return SubphaseMenstrualPossible, ResolutionUserLoggedWithAssumedEnd, true
		default: // 21.4
			st.Warnings.Add(WarnPeriodEndMissing)
			st.Warnings.Add(WarnPeriodEndMissingWarningCapExceeded)
			f.forcedPoor, f.forcedLow = true, true
			return SubphaseMenstrualPossible, ResolutionUserLoggedWithAssumedEnd, true
		}
	default: // a declared onboarding start: menstrual over the assumed duration only, no end-missing tiers
		if cycleDay <= soft {
			return SubphaseMenstrual, ResolutionOnboardingBased, true
		}
	}
	return SubphaseUnknown, ResolutionUnknown, false
}

// suggestPregnancyTest is §2.22: mode cycle or ttc, no hormonal contraception, and days_late ≥ the mode's
// threshold (Params). A late period is the premise, so days_late 0 never suggests a test.
func (s *State) suggestPregnancyTest(daysLate int) bool {
	if s.in.Hormonal || daysLate < 1 {
		return false
	}
	switch s.in.Mode {
	case ModeTTC:
		return daysLate >= s.p.PregnancyTestTTCDaysLate
	case ModeCycle:
		return daysLate >= s.p.PregnancyTestCycleDaysLate
	default:
		return false
	}
}
