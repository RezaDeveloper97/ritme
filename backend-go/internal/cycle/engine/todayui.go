package engine

// The §2.43 Today-screen mapping (E-17): the engine owns it so web, the daily card and later clients cannot drift.
// «هنوز نه» / «هنوز ادامه دارد» stay client-local for the day (spec), so they are not part of the block.

// Today is the today payload: today's status with future_cycles filled (§2.16, E-05; Resolve and Range leave it
// nil) and its today_ui block.
func (s *State) Today(today Day) (Status, TodayUI) {
	st := s.Resolve(today, today)
	st.FutureCycles = s.FutureCycles(today)
	return st, s.TodayUI(st)
}

// TodayUI is the §2.43 mapping of one status (normally today's), pure and allocation-free. It reads only the status
// and the State's Params (prompt window, hard cap, PMS days, fertile window, overdue threshold).
//   - ring_action: end when the current real period's end is unconfirmed and its period day (cycle_day, the day the
//     §2.12 caps count) is in [EPL − 1, hard_cap]; start when no real period is open and the target is in
//     [next − PromptBefore, next + PromptAfter]; otherwise edit.
//   - banner (at most one, table order): end_missing (warning or hard cap exceeded) → overdue
//     (predicted_period_overdue with days_late ≥ OverdueDaysLate) → ask_last_period (requires_user_input with
//     stale_onboarding_anchor or cycle_unresolved_after_expected_period) → none.
//   - show_pregnancy_test_button: suggest_pregnancy_test on the overdue banner.
//   - prediction_card: null without a predicted next start or while the engine is suspended; fertile and ovulation
//     null under fertility suppression (§2.42) or an empty display window (§2.19).
func (s *State) TodayUI(st Status) TodayUI {
	p := &s.p
	a := &st.Anchors
	var ui TodayUI

	open := a.CurrentPeriodStartSource == StartUserLogged && !a.CurrentPeriodEndIsConfirmed &&
		a.CurrentPeriodStart.IsSet()
	next := a.PredictedNextPeriodStart
	cd, hasDay := st.CycleDay.Get()
	switch {
	case open && hasDay && cd >= st.Metrics.EffectivePeriodLength-1 && cd <= p.PeriodHardCap:
		ui.RingAction = RingEnd
	case !open && next.IsSet() && st.Date >= next-days(p.PromptBefore) && st.Date <= next+days(p.PromptAfter):
		ui.RingAction = RingStart
	default:
		ui.RingAction = RingEdit
	}

	w := st.Warnings
	switch {
	case w.Has(WarnPeriodEndMissingWarningCapExceeded) || w.Has(WarnPeriodEndMissingHardCapExceeded):
		ui.Banner = BannerEndMissing
	case w.Has(WarnPredictedPeriodOverdue) && st.DaysLate.Or(0) >= p.OverdueDaysLate:
		ui.Banner = BannerOverdue
		ui.ShowPregnancyTestButton = st.SuggestPregnancyTest
	case st.RequiresUserInput && (w.Has(WarnStaleOnboardingAnchor) || w.Has(WarnCycleUnresolvedAfterExpectedPeriod)):
		ui.Banner = BannerAskLastPeriod
	}

	if !next.IsSet() || st.Limits.EngineSuspended {
		return ui
	}
	epl := max(1, st.Metrics.EffectivePeriodLength)
	card := PredictionCard{
		Period:          DayRange{Start: next, End: next + days(epl-1)},
		PMS:             DayRange{Start: next - days(p.PMSDays), End: next - 1},
		CycleLength:     st.Metrics.EffectiveCycleLength,
		CycleRegularity: st.Metrics.CycleRegularity,
	}
	if ov := a.EstimatedOvulationDate; ov.IsSet() && !st.Limits.FertilitySuppressed {
		card.Ovulation = Some(OvulationMark{Date: ov, Source: a.OvulationSource})
		start := ov - days(p.FertileBefore) // §2.19: max(O − before, current_period_end + 1) .. O
		if e := a.CurrentPeriodEnd; e.IsSet() {
			start = max(start, e+1)
		}
		if start <= ov {
			card.Fertile = Some(DayRange{Start: start, End: ov})
		}
	}
	ui.PredictionCard = Some(card)
	return ui
}
