package engine

import (
	"cmp"
	"slices"
)

// The limiting conditions (§2.42, E-13, E-14, E-24…E-26, E-29): which statuses are fertility-suppressed or
// suspended, and the positive-test prompt. Precedence when several apply: pregnancy mode > postpartum not returned
// > hormonal contraception (the warnings of every active condition are still emitted). Menopause and teen modes
// never suppress (E-29).

// preparePregnancyTests keeps Prepare's copy of the positive pregnancy tests only, sorted by date (E-26); negative
// and undated tests never prompt.
func (s *State) preparePregnancyTests() {
	t := slices.DeleteFunc(s.in.PregnancyTests, func(t PregnancyTest) bool { return !t.Positive || t.Date == NoDay })
	slices.SortStableFunc(t, func(a, b PregnancyTest) int { return cmp.Compare(a.Date, b.Date) })
	s.in.PregnancyTests = t
}

// postpartumPeriods reports whether the postpartum rule is in force for target (E-14: postpartum mode, or any mode
// while breastfeeding, on or after the delivery date) and how many real periods started after the delivery date and
// on or before target. A target before the delivery date resolves normally (T-M9-48b: the pre-pregnancy history keeps
// its phases). Without a delivery date every real period on record counts (breastfeeding with no delivery recorded).
func (s *State) postpartumPeriods(target Day) (n int, applies bool) {
	if s.in.Mode != ModePostpartum && !s.in.Breastfeeding {
		return 0, false
	}
	if s.delivery != NoDay && target < s.delivery {
		return 0, false
	}
	upto := s.lastRealAtOrBefore(target) + 1
	after := 0
	if s.delivery != NoDay {
		after, _ = slices.BinarySearchFunc(s.periods, s.delivery+1,
			func(p Period, d Day) int { return cmp.Compare(p.Start, d) })
	}
	return max(0, upto-after), true
}

// pregnancyGap reports whether target lies in the pregnancy before the delivery date (T-M9-48b): the E-14 rule is
// set up (postpartum mode or breastfeeding, with a delivery date), target is before the delivery, and target's
// cycle is a real or declared one that runs across the delivery (no real start between its start and the delivery).
// Such a day is neither postpartum nor a normal cycle day: logged bleeding stays menstrual, every other day is
// unknown with nothing predicted, no postpartum warning and no suppression. Closed cycles that end before the
// delivery are laid out normally (E-22).
func (s *State) pregnancyGap(target Day, c *cycle) bool {
	if s.delivery == NoDay || target >= s.delivery || c.futureK > 0 ||
		(s.in.Mode != ModePostpartum && !s.in.Breastfeeding) {
		return false
	}
	return c.a.nextReal == NoDay || c.a.nextReal > s.delivery
}

// suppressionAt is the suppression reason of target (E-24), SuppressionNone when the cycle is computed normally.
func (s *State) suppressionAt(target Day) SuppressionReason {
	if s.in.Mode == ModePregnancy {
		return SuppressionPregnancy
	}
	if n, ok := s.postpartumPeriods(target); ok && n < s.p.PostpartumPeriods {
		return SuppressionPostpartumNotReturned
	}
	if s.in.Hormonal {
		return SuppressionHormonalContraception
	}
	return SuppressionNone
}

// promptPregnancy is E-26: mode cycle or ttc and a positive test dated from the cycle start to target. A new real
// start moves the cycle start past the test, which clears the prompt; so does a mode change.
func (s *State) promptPregnancy(cycleStart, target Day) bool {
	if s.in.Mode != ModeCycle && s.in.Mode != ModeTTC {
		return false
	}
	t := s.in.PregnancyTests
	i, _ := slices.BinarySearchFunc(t, cycleStart, func(t PregnancyTest, d Day) int { return cmp.Compare(t.Date, d) })
	return i < len(t) && t[i].Date <= target
}

// limitsFor builds the E-24 block of target from its suppression reason and its cycle start (NoDay: no anchor, so no
// prompt).
func (s *State) limitsFor(target, cycleStart Day, reason SuppressionReason) Limits {
	l := Limits{
		FertilitySuppressed: reason != SuppressionNone,
		SuppressionReason:   reason,
		EngineSuspended:     reason == SuppressionPregnancy,
	}
	if cycleStart != NoDay {
		l.PromptPregnancyMode = s.promptPregnancy(cycleStart, target)
	}
	return l
}

// Suppression is the E-24 limits block of day d (normally today): fertility_suppressed and its reason,
// engine_suspended (pregnancy mode) and prompt_pregnancy_mode. The future-cycle projection (T-M9-20) and every
// consumer read this one flag instead of inferring suppression from warnings. It allocates nothing.
func (s *State) Suppression(d Day) Limits {
	start := NoDay
	if a, ok := s.anchorAt(d); ok {
		start = a.start
	}
	return s.limitsFor(d, start, s.suppressionAt(d))
}
