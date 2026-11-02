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
// fertility suppression (markers = false) no ovulation is computed (§2.42), so BBT/LH never move next. A skip anchor
// (noPeriod, E-15) has no period: its follicular range starts on the anchor day.
func (s *State) layoutFor(a anchor, markers, noPeriod bool) layout {
	end, confirmed := a.loggedEnd, a.loggedEnd != NoDay
	switch {
	case noPeriod:
		end, confirmed = a.start-1, false
	case !confirmed:
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
		next = a.start + days(s.ecl())
	}
	return newLayout(a.start, end, confirmed, next, next-days(s.p.Luteal.Default), s.p.FertileBefore)
}

// ecl is effective_cycle_length as a step (≥ 1).
func (s *State) ecl() int { return max(1, s.cycleLen) }

// facts are the resolve-time signals the quality layer (grade, quality.go) reads besides the Status itself.
type facts struct {
	anchored    bool
	forcedPoor  bool // a tier forces data_quality = poor
	forcedLow   bool // a tier forces confidence = low
	realOpen    bool // the anchor is a real period with no logged end (confidence_reasons += missing_period_end)
	endInvolved bool // the target's phase depends on current_period_end (menstrual, follicular, a window at end + 1)
	// futureK > 0: the target was resolved from future_cycles[futureK] (E-03); its confidence is capped at
	// futureCap = min(today's confidence, k == 1 ? high : medium) and the predicted_reference low does not apply.
	futureK   int
	futureCap Confidence
}

// layoutMemo lets Range reuse one cycle's layout for every day of that cycle.
type layoutMemo struct {
	ok       bool
	a        anchor
	markers  bool
	noPeriod bool
	l        layout
}

// layout is the cycle's day map, from the memo when it holds the same cycle.
func (s *State) layout(c *cycle, markers bool, m *layoutMemo) layout {
	if m == nil {
		return s.layoutFor(c.a, markers, c.noPeriod)
	}
	if !m.ok || m.a != c.a || m.markers != markers || m.noPeriod != c.noPeriod {
		*m = layoutMemo{ok: true, a: c.a, markers: markers, noPeriod: c.noPeriod,
			l: s.layoutFor(c.a, markers, c.noPeriod)}
	}
	return m.l
}

// Resolve is the engine status of one target date (§2.20–§2.34, §2.41, §2.42, E-03, E-15) for the current cycle and
// the history. today is the user-local day: a target after today on or after today's predicted next start resolves
// inside today's future_cycles projection (E-03, E-04). It allocates nothing but the returned value.
func (s *State) Resolve(target, today Day) Status {
	var ch chain
	if target > today {
		ch = s.chainAt(today)
	}
	return s.resolveDay(target, today, &ch, nil)
}

// resolveDay is Resolve with today's projection precomputed (zero chain when target ≤ today) and an optional layout
// memo (Range).
func (s *State) resolveDay(target, today Day, ch *chain, m *layoutMemo) Status {
	st, f := s.resolveWith(target, ch, m)
	s.grade(&st, f, today)
	return st
}

// resolve is the ungraded status of target (tests).
func (s *State) resolve(target, today Day) (Status, facts) {
	var ch chain
	if target > today {
		ch = s.chainAt(today)
	}
	return s.resolveWith(target, &ch, nil)
}

func (s *State) resolveWith(target Day, ch *chain, m *layoutMemo) (Status, facts) {
	// A logged real day — a period starting after today, or a closed period still running on target (a logged end
	// after today, or a period longer than the cycle) — resolves from the history, never from the projection, so a
	// logged bleeding day is never shown as fertile (§2.40; found by FuzzPrepare, T-M9-22).
	if ch.covers(target) && !s.realStartIn(ch.today, target) && !s.loggedPeriodCovers(target) {
		c, ok := s.futureCycle(target, ch)
		return s.resolveCycle(target, &c, ok, true, m)
	}
	c, ok := s.cycleAt(target)
	return s.resolveCycle(target, &c, ok, false, m)
}

// resolveCycle resolves target inside cycle c (ok = false: no cycle). quiet marks a future target outside the
// projection (E-04 late beyond max_late, E-28 beyond the 6th cycle): unknown without insufficient_anchor_data.
func (s *State) resolveCycle(target Day, c *cycle, ok, quiet bool, m *layoutMemo) (Status, facts) {
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
	gap := ok && s.pregnancyGap(target, c)
	// E-14 / E-25: no cycle is predicted at all (only real period days resolve); hormonal (E-13) keeps the period
	// predictions and drops the fertile/ovulation layer only. The pregnancy gap before a delivery is unknown the
	// same way, without the postpartum warning or suppression (T-M9-48b).
	noPrediction := sup == SuppressionPregnancy || sup == SuppressionPostpartumNotReturned || gap
	if sup == SuppressionPostpartumNotReturned {
		st.Warnings.Add(WarnPostpartumCyclesNotReturned)
	}

	if !ok {
		st.Limits = s.limitsFor(target, NoDay, sup)
		// §2.20.6 / §2.28 insufficient: no real start and no declared start on or before the target. A suspended or
		// postpartum status is complete without one (E-14, E-25: requires_user_input = false), and so is a future
		// target outside the projection (E-04, E-28).
		if !noPrediction && !quiet {
			st.Warnings.Add(WarnInsufficientAnchorData)
		}
		return st, f
	}
	a := c.a
	f.anchored = true
	f.futureK = c.futureK
	st.FutureK = c.futureK
	st.Limits = s.limitsFor(target, c.promptFrom, sup)
	l := s.layout(c, sup == SuppressionNone && !gap, m)

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
	switch {
	case c.noPeriod: // E-15: a skipped cycle has no period, so no period end
		st.Anchors.CurrentPeriodEnd, st.Anchors.CurrentPeriodEndSource = NoDay, EndUnknown
	case l.endConfirmed:
		st.Anchors.CurrentPeriodEndSource = EndUserLogged
	}
	if sup != SuppressionNone || gap { // §2.42: ovulation is not computed
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
	st.Warnings |= c.warns // stale_onboarding_anchor, skipped_cycle_recorded, prediction_based_output (E-01)

	// 1. menstrual (§2.21): logged bleeding resolves under every limiting condition (E-13, E-14, E-25).
	sub, res, decided := s.menstrual(&st, &f, c, &l, cycleDay)

	// E-14 / E-25: every other day is unknown with nothing predicted.
	if !decided && noPrediction {
		decided, sub, res = true, SubphaseUnknown, ResolutionUnknown
	}

	// 2. period_expected (§2.22) — never inside a completed cycle (E-22); also under hormonal contraception (E-13).
	// A future projection never reaches it (target < its next start), so lateness applies to targets ≤ today (E-03).
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

	// §2.30 / E-10: a cycle derived from the declared onboarding start (the start itself, its stale roll-forward,
	// a skip after it, or today's projection from it) resolves every phase as onboarding_based (default_based when
	// both lengths are defaults), period_expected days 0..max_late included; > max_late stays prediction (E-11).
	if c.declared && res != ResolutionUnknown && !unresolvedLate {
		res = ResolutionOnboardingBased
		if s.cycleSrc == LengthDefault && s.periodSrc == LengthDefault {
			res = ResolutionDefaultBased
		}
	}

	// §2.32 source warnings. Manual lengths (E-20) are not onboarding data.
	d := s.in.Declared
	if c.declared || (!d.Manual && (s.cycleSrc == LengthDeclared || s.periodSrc == LengthDeclared)) {
		st.Warnings.Add(WarnOnboardingDataUsed)
	}
	if s.cycleSrc == LengthDefault || s.periodSrc == LengthDefault {
		st.Warnings.Add(WarnDefaultValuesUsed)
	}

	st.Subphase = sub
	st.MainPhase = sub.Main()
	f.realOpen = a.src == StartUserLogged && !l.endConfirmed
	f.endInvolved = st.MainPhase == MainPhaseMenstrual || st.MainPhase == MainPhaseFollicular ||
		(st.MainPhase == MainPhaseFertile && l.fertileStart == l.end+1)
	st.ResolutionSource = res
	if c.futureK > 0 {
		f.futureCap = c.futureCap
	}
	// E-25 / E-14: no cycle runs during a pregnancy or before cycles return after a delivery, so only logged bleeding
	// keeps its day (T-M9-53: no «روز N چرخه» counted from a pre-delivery start; T-M9-48b: nor inside the pregnancy
	// gap before the delivery).
	if noPrediction && st.MainPhase != MainPhaseMenstrual {
		st.CycleDay = None[int]()
	}
	if sup != SuppressionNone { // §2.26 / §2.34: suppressed fertility is unknown and has no countdown
		st.FertilityLevel = FertilityUnknown
		return st, f
	}
	st.FertilityLevel = fertilityOf(sub)
	if daysLate <= p.MaxLateDays && !gap { // §2.34 (the pregnancy gap has no ovulation)
		st.DaysToOvulation = Some(int(l.ov - target))
	}
	st.SuggestPregnancyTest = s.suggestPregnancyTest(daysLate)
	return st, f
}

// menstrual is priority 1 of §2.20 with the §2.21 tiers and the §2.12 caps (soft = EPL, warning = max(EPL + extra,
// min), hard). decided is false when the day is not menstrual and resolves onward. The hard cap is checked before
// the warning cap, so a warning cap above the hard cap (EPL ≥ 10) never keeps a day menstrual past the hard cap.
func (s *State) menstrual(st *Status, f *facts, c *cycle, l *layout, cycleDay int) (Subphase, ResolutionSource, bool) {
	p := &s.p
	a := c.a
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
	case c.noPeriod: // a skip anchor (E-15): the user said no period came, so no menstrual days
	case c.futureK == 0 && c.warns.Has(WarnStaleOnboardingAnchor):
		// §2.41 + §2.20.1 (T-M9-52): the stale roll-forward is a reference date for counting, not a period — §2.20
		// gives menstrual only to a real or active period, and §2.41 asks for the last period (requires_user_input,
		// the §2.43 ask_last_period banner). Its assumed days are unknown (still a prediction, like E-13), never a
		// period day under the banner that says the predictions are out of date.
		if cycleDay <= soft {
			return SubphaseUnknown, ResolutionPrediction, true
		}
	default: // a declared or predicted start (§2.41, E-03): menstrual over the assumed duration only, no end tiers
		if cycleDay <= soft {
			return SubphaseMenstrual, ResolutionPrediction, true // E-10 turns a declared one into onboarding_based
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
