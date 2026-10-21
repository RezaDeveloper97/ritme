package engine

import "slices"

// The anchors that are not a logged period start (§2.10 predicted_reference): the skipped-cycle anchor (E-15,
// §2.39.8) and the stale onboarding anchor (§2.41). Future-cycle projections (E-03) are in future.go.

// cycle is the cycle a target is resolved in: its anchor plus how the anchor was derived.
type cycle struct {
	a          anchor
	declared   bool       // derived from the declared onboarding start (E-10 resolution, onboarding_data_used)
	noPeriod   bool       // a skip anchor: no period at the start (E-15), so no menstrual days
	warns      WarningSet // stale_onboarding_anchor / skipped_cycle_recorded / prediction_based_output
	promptFrom Day        // the cycle start the positive-test prompt counts from (E-26): the real or declared start
	futureK    int        // > 0: future_cycles[futureK] (E-03)
	futureCap  Confidence // min(today's confidence, k == 1 ? high : medium)
}

// Diagnostics are today-independent notes about the input that change no output field.
type Diagnostics struct {
	// IgnoredSkips are the recorded skips that match no predicted start any more (the data changed since the user
	// marked them, E-15): they create no anchor. A skip still excludes the real cycle it lies in from the medians,
	// variability and regularity (Prepare).
	IgnoredSkips []Day
}

// Diagnostics are the State's diagnostics (do not modify).
func (s *State) Diagnostics() Diagnostics { return Diagnostics{IgnoredSkips: s.ignoredSkips} }

// prepareSkipAnchors is Prepare part 5 (E-15): a skip creates a virtual anchor when it equals the predicted start of
// the cycle it lies in — the base's predicted next (the latest real start before it, or the declared onboarding
// start: start + effective_cycle_length, or O + effective_luteal_length when that real cycle has a BBT/LH
// ovulation), or the previous matching skip + effective_cycle_length (consecutive skips chain). Every other skip is
// ignored and reported in Diagnostics. Matching ignores today, so the State stays today-independent.
func (s *State) prepareSkipAnchors() {
	if len(s.skips) == 0 {
		return
	}
	base, expected := NoDay, NoDay
	for _, sk := range s.skips {
		a, ok := s.anchorAt(sk)
		if !ok || a.start == sk {
			s.ignoredSkips = append(s.ignoredSkips, sk)
			continue
		}
		if a.start != base {
			// The predicted next of the base as if it were the current cycle (no later real start).
			cur := a
			cur.nextReal = NoDay
			base, expected = a.start, s.layoutFor(cur, !s.in.Hormonal, false).next
		}
		if sk != expected {
			s.ignoredSkips = append(s.ignoredSkips, sk)
			continue
		}
		s.skipAnchors = append(s.skipAnchors, sk)
		expected = sk + days(s.ecl())
	}
}

// skipAnchorIn is the latest skip anchor v with after < v ≤ target, NoDay when none.
func (s *State) skipAnchorIn(after, target Day) Day {
	i, found := slices.BinarySearch(s.skipAnchors, target)
	if found {
		i++
	}
	if i > 0 && s.skipAnchors[i-1] > after {
		return s.skipAnchors[i-1]
	}
	return NoDay
}

// cycleAt is the cycle of target from the history: the latest real or declared start (anchorAt), moved to the
// latest skip anchor after it (E-15), then — in a history without real periods — rolled forward when stale
// (§2.41: target − anchor > effective_cycle_length + stale_anchor.extra_days → anchor + n·ECL, n = ⌊diff / ECL⌋).
// The stale rule also applies to a skip anchor of a declared-only history (it is derived from the onboarding date).
func (s *State) cycleAt(target Day) (cycle, bool) {
	a, ok := s.anchorAt(target)
	if !ok {
		return cycle{}, false
	}
	c := cycle{a: a, declared: a.src == StartOnboardingDeclared, promptFrom: a.start}
	if v := s.skipAnchorIn(a.start, target); v != NoDay {
		c.a.start, c.a.src, c.a.loggedEnd, c.a.idx = v, StartPredictedReference, NoDay, -1
		c.noPeriod = true
		c.warns.Add(WarnSkippedCycleRecorded)
		c.warns.Add(WarnPredictionBasedOutput)
	}
	if len(s.periods) == 0 {
		ecl := s.ecl()
		if diff := int(target - c.a.start); diff > ecl+s.p.StaleAnchorExtraDays {
			c.a.start += days(diff / ecl * ecl)
			c.a.src, c.noPeriod = StartPredictedReference, false
			c.warns.Add(WarnStaleOnboardingAnchor)
			c.warns.Add(WarnPredictionBasedOutput)
		}
	}
	return c, true
}

// realStartIn reports whether a real period starts in (after, upto].
func (s *State) realStartIn(after, upto Day) bool {
	i := s.lastRealAtOrBefore(upto)
	return i >= 0 && s.periods[i].Start > after
}
