package engine

// Future cycles (§2.16, E-03…E-05, E-28) and the calendar range API.

// MaxRangeDays is the longest span Range resolves.
const MaxRangeDays = 400

// chain is today's future_cycles projection, computed once from today's status. The zero chain covers nothing.
type chain struct {
	active     bool // today is anchored and not under pregnancy or postpartum-not-returned (E-14, E-25)
	empty      bool // days_late > max_late today: the list is empty and future targets are unknown (E-04)
	today      Day
	from       Day // the first target resolved from the projection: future_cycles[1].period_start (E-03)
	start1     Day // future_cycles[1].period_start: today's predicted next, or today + 1 while late (E-04)
	lastEnd    Day // future_cycles[N].period_end; later targets are unknown (E-28)
	ov1        Day // today's estimated_ovulation_date (NoDay under suppression)
	ov1Src     OvulationSource
	conf       Confidence // today's confidence
	suppressed bool       // hormonal contraception: period dates only (E-13)
	declared   bool       // today's cycle derives from the declared onboarding start (E-10)
	warns      WarningSet // today's stale_onboarding_anchor / skipped_cycle_recorded, carried until a real start
}

// chainAt builds today's projection. It allocates nothing.
func (s *State) chainAt(today Day) chain {
	ch := chain{today: today}
	sup := s.Suppression(today).SuppressionReason // the E-24 flag (T-M9-19)
	if sup == SuppressionPregnancy || sup == SuppressionPostpartumNotReturned {
		return ch
	}
	ch.suppressed = sup != SuppressionNone
	c, ok := s.cycleAt(today)
	if !ok {
		return ch
	}
	t := s.resolveDay(today, today, &chain{}, nil)
	ch.active = true
	ch.declared = c.declared
	ch.warns = c.warns & (1<<WarnStaleOnboardingAnchor | 1<<WarnSkippedCycleRecorded)
	if t.DaysLate.Or(0) > s.p.MaxLateDays {
		ch.empty, ch.from = true, today+1
		return ch
	}
	next := t.Anchors.PredictedNextPeriodStart
	ch.start1 = next
	if today >= next { // E-04 rebase: no new start logged and the period is due or late
		ch.start1 = today + 1
	}
	ch.from = ch.start1
	n := max(1, s.p.FutureCycles)
	ch.lastEnd = ch.start1 + days((n-1)*s.ecl()+max(1, s.periodLen)-1)
	ch.ov1, ch.ov1Src = t.Anchors.EstimatedOvulationDate, t.Anchors.OvulationSource
	ch.conf = t.Confidence
	return ch
}

// covers reports whether target resolves from the projection (a later real start is checked by the caller).
func (ch *chain) covers(target Day) bool { return ch.active && target > ch.today && target >= ch.from }

// futureCap is E-03: min(today's confidence, k == 1 ? high : medium).
func (ch *chain) futureCap(k int) Confidence {
	if k == 1 {
		return ch.conf
	}
	return min(ch.conf, ConfidenceMedium)
}

// futureCycle is the cycle of a covered target: future_cycles[k] with k the last projected start ≤ target. ok is
// false while late beyond max_late (E-04) or after future_cycles[N].period_end (E-28).
func (s *State) futureCycle(target Day, ch *chain) (cycle, bool) {
	if ch.empty || target > ch.lastEnd {
		return cycle{}, false
	}
	ecl := s.ecl()
	k := min(1+int(target-ch.start1)/ecl, max(1, s.p.FutureCycles))
	c := cycle{
		a: anchor{start: ch.start1 + days((k-1)*ecl), src: StartPredictedReference, loggedEnd: NoDay,
			nextReal: NoDay, idx: -1},
		declared:   ch.declared,
		warns:      ch.warns | 1<<WarnPredictionBasedOutput,
		promptFrom: NoDay,
		futureK:    k,
		futureCap:  ch.futureCap(k),
	}
	if a, ok := s.anchorAt(ch.today); ok {
		c.promptFrom = a.start // E-26 counts positive tests from today's real or declared cycle start
	}
	return c, true
}

// FutureCycles is today's future_cycles list (§2.16, E-04, E-05): k = 1..N, each period k with the ovulation that
// precedes it. k = 1 carries today's estimated ovulation and source; k ≥ 2 the calendar ovulation period_start − 14.
// While no new start is logged and today ≥ predicted_next the chain starts tomorrow (E-04); late beyond max_late,
// pregnancy and postpartum-not-returned give an empty list; under hormonal contraception the fertile and ovulation
// days are NoDay (E-13). Confidence: k = 1 today's, k ≥ 2 at most medium (E-03). Display only, never stored (§2.40).
func (s *State) FutureCycles(today Day) []FutureCycle {
	ch := s.chainAt(today)
	if !ch.active || ch.empty {
		return []FutureCycle{}
	}
	n, ecl, epl := max(1, s.p.FutureCycles), s.ecl(), max(1, s.periodLen)
	out := make([]FutureCycle, n)
	for i := range out {
		k := i + 1
		start := ch.start1 + days(i*ecl)
		fc := FutureCycle{
			K: k, PeriodStart: start, PeriodEnd: start + days(epl-1),
			FertileStart: NoDay, FertileEnd: NoDay, Ovulation: NoDay, OvulationSource: OvulationUnknown,
			Confidence: ch.futureCap(k),
		}
		ov, src := start-days(s.p.Luteal.Default), OvulationCalendar
		if k == 1 {
			ov, src = ch.ov1, ch.ov1Src
		}
		if !ch.suppressed && ov != NoDay {
			fc.Ovulation, fc.OvulationSource = ov, src
			fc.FertileStart, fc.FertileEnd = ov-days(s.p.FertileBefore), ov
		}
		out[i] = fc
	}
	return out
}

// Range is Resolve for every day from..to (inclusive), with today's projection and each cycle's layout computed
// once. from and to may cross today and cycle boundaries. It answers nil for to < from or a span over MaxRangeDays.
func (s *State) Range(from, to, today Day) []Status {
	if to < from || int(to-from) >= MaxRangeDays {
		return nil
	}
	out := make([]Status, int(to-from)+1)
	var ch chain
	if to > today {
		ch = s.chainAt(today)
	}
	var m layoutMemo
	for i := range out {
		out[i] = s.resolveDay(from+days(i), today, &ch, &m)
	}
	return out
}
