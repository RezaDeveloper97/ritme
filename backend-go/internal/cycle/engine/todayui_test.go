package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// uiAt is the today_ui block of target with today = target.
func uiAt(s *State, target Day) (Status, TodayUI) {
	st := s.Resolve(target, target)
	return st, s.TodayUI(st)
}

// TestTodayUI_RingStartWindow is §2.43 row 1: start from next − 3 to next + 14, edit outside.
func TestTodayUI_RingStartWindow(t *testing.T) {
	cur := day(julyStart)
	s := prep(regular(cur, 28, 5))
	next := cur + 28
	cases := []struct {
		target Day
		want   RingAction
	}{
		{next - 4, RingEdit},
		{next - 3, RingStart},
		{next, RingStart},
		{next + 14, RingStart},
		{next + 15, RingEdit},
		{cur + 2, RingEdit}, // inside a confirmed period
	}
	for _, c := range cases {
		_, ui := uiAt(s, c.target)
		assert.Equal(t, c.want, ui.RingAction, "target %s", c.target)
	}
}

// TestTodayUI_RingEndWindow is §2.43 row 2: an open period's period day EPL − 1 … hard_cap → end; hard_cap + 1 →
// edit with the end_missing banner.
func TestTodayUI_RingEndWindow(t *testing.T) {
	cur := day(julyStart)
	in := regular(cur, 28, 5)
	in.Periods[len(in.Periods)-1].End = NoDay
	s := prep(in)
	epl, _ := s.PeriodLength()
	hard := s.Params().PeriodHardCap
	require.Equal(t, 5, epl)

	at := func(cycleDay int) (Status, TodayUI) { return uiAt(s, cur+Day(cycleDay-1)) }

	_, ui := at(epl - 2)
	assert.Equal(t, RingEdit, ui.RingAction, "EPL − 2")
	assert.Equal(t, BannerNone, ui.Banner)
	_, ui = at(epl - 1)
	assert.Equal(t, RingEnd, ui.RingAction, "EPL − 1")
	_, ui = at(hard)
	assert.Equal(t, RingEnd, ui.RingAction, "hard cap")
	assert.Equal(t, BannerEndMissing, ui.Banner, "past the warning cap")
	st, ui := at(hard + 1)
	assert.True(t, st.Warnings.Has(WarnPeriodEndMissingHardCapExceeded))
	assert.Equal(t, RingEdit, ui.RingAction, "hard cap + 1")
	assert.Equal(t, BannerEndMissing, ui.Banner)

	// A period with a logged end never shows the end prompt.
	closed := prep(regular(cur, 28, 5))
	_, ui = uiAt(closed, cur+Day(epl-1))
	assert.Equal(t, RingEdit, ui.RingAction)
}

func TestTodayUI_Banners(t *testing.T) {
	cur := day(julyStart)
	next := cur + 28

	// late 7 → no banner; late 8 in cycle mode → overdue, test button from cycle_days_late 8.
	s := prep(regular(cur, 28, 5))
	_, ui := uiAt(s, next+7)
	assert.Equal(t, BannerNone, ui.Banner, "late 7")
	assert.False(t, ui.ShowPregnancyTestButton)
	st, ui := uiAt(s, next+8)
	assert.Equal(t, BannerOverdue, ui.Banner, "late 8")
	assert.Equal(t, st.SuggestPregnancyTest, ui.ShowPregnancyTestButton)

	// late 8 ttc → overdue + test button.
	in := regular(cur, 28, 5)
	in.Mode = ModeTTC
	_, ui = uiAt(prep(in), next+8)
	assert.Equal(t, BannerOverdue, ui.Banner)
	assert.True(t, ui.ShowPregnancyTestButton)

	// A test suggested before the overdue tier (ttc late 1) shows no button: the button lives on the banner.
	_, ui = uiAt(prep(in), next+1)
	assert.Equal(t, BannerNone, ui.Banner)
	assert.False(t, ui.ShowPregnancyTestButton)

	// late 8 on the pill → overdue without the button.
	_, ui = uiAt(prep(pill()), next+8)
	assert.Equal(t, BannerOverdue, ui.Banner)
	assert.False(t, ui.ShowPregnancyTestButton)

	// late 15 (unresolved) carries predicted_period_overdue too, so the overdue banner wins (table order).
	st, ui = uiAt(s, next+15)
	assert.True(t, st.Warnings.Has(WarnCycleUnresolvedAfterExpectedPeriod))
	assert.Equal(t, BannerOverdue, ui.Banner)

	// Stale onboarding anchor → ask_last_period.
	lmp := day("2026-07-01")
	st, ui = uiAt(prep(declared(Declared{LastPeriodStart: Some(lmp)})), lmp+90)
	require.True(t, st.Warnings.Has(WarnStaleOnboardingAnchor))
	assert.Equal(t, BannerAskLastPeriod, ui.Banner)

	// Unresolved without the overdue warning (a hand-built status) → ask_last_period.
	var hand Status
	hand.Warnings.Add(WarnCycleUnresolvedAfterExpectedPeriod)
	hand.RequiresUserInput = true
	assert.Equal(t, BannerAskLastPeriod, s.TodayUI(hand).Banner)
	hand.RequiresUserInput = false
	assert.Equal(t, BannerNone, s.TodayUI(hand).Banner, "needs requires_user_input")

	// end_missing and overdue at once → only end_missing: an open period in 21-day cycles, period day 30 = late 8.
	in = withPeriods(periodsEnding(cur, 5, 21, 21))
	in.Periods[len(in.Periods)-1].End = NoDay
	st, ui = uiAt(prep(in), cur+29)
	require.True(t, st.Warnings.Has(WarnPredictedPeriodOverdue))
	require.Equal(t, Some(8), st.DaysLate)
	require.True(t, st.Warnings.Has(WarnPeriodEndMissingHardCapExceeded))
	assert.Equal(t, BannerEndMissing, ui.Banner)
	assert.False(t, ui.ShowPregnancyTestButton)
	assert.Equal(t, RingEdit, ui.RingAction)
}

func TestTodayUI_PredictionCard(t *testing.T) {
	cur := day(julyStart)

	// §2.35 example: next 07-30, EPL 5, ovulation 07-16 (calendar), period 07-01..07-05.
	s := prep(withPeriods(periodsEnding(cur, 5, 27, 29, 30)))
	_, ui := uiAt(s, day("2026-07-18"))
	assert.Equal(t, RingEdit, ui.RingAction)
	assert.Equal(t, BannerNone, ui.Banner)
	assert.Equal(t, Some(PredictionCard{
		Period:          DayRange{Start: day("2026-07-30"), End: day("2026-08-03")},
		PMS:             DayRange{Start: day("2026-07-27"), End: day("2026-07-29")},
		Fertile:         Some(DayRange{Start: day("2026-07-11"), End: day("2026-07-16")}),
		Ovulation:       Some(OvulationMark{Date: day("2026-07-16"), Source: OvulationCalendar}),
		CycleLength:     29,
		CycleRegularity: RegularityRegular,
	}), ui.PredictionCard)

	// §2.19: the display window starts after the period end (21-day cycle: O = day 8, period 1..5 → 6..8).
	s21 := prep(regular(cur, 21, 5))
	_, ui = uiAt(s21, cur+10)
	c, ok := ui.PredictionCard.Get()
	require.True(t, ok)
	assert.Equal(t, Some(DayRange{Start: cur + 5, End: cur + 7}), c.Fertile)

	// An empty display window is null (a 9-day period in a 21-day cycle covers the whole biological window).
	long := Status{Date: cur, Anchors: Anchors{CurrentPeriodStart: cur, CurrentPeriodEnd: cur + 8,
		PredictedNextPeriodStart: cur + 21, EstimatedOvulationDate: cur + 7, OvulationSource: OvulationCalendar}}
	c, ok = s21.TodayUI(long).PredictionCard.Get()
	require.True(t, ok)
	assert.False(t, c.Fertile.OK())
	assert.True(t, c.Ovulation.OK())

	// Under the pill: period and PMS stay, fertile and ovulation are null.
	_, ui = uiAt(prep(pill()), cur+10)
	c, ok = ui.PredictionCard.Get()
	require.True(t, ok)
	assert.Equal(t, DayRange{Start: cur + 28, End: cur + 32}, c.Period)
	assert.False(t, c.Fertile.OK())
	assert.False(t, c.Ovulation.OK())

	// Pregnancy → no card; postpartum before cycles return → no card; no anchor → no card.
	in := regular(cur, 28, 5)
	in.Mode = ModePregnancy
	_, ui = uiAt(prep(in), cur+10)
	assert.False(t, ui.PredictionCard.OK())
	assert.Equal(t, RingEdit, ui.RingAction)

	in = regular(cur, 28, 5)
	in.Mode, in.DeliveryDate = ModePostpartum, Some(cur+3)
	_, ui = uiAt(prep(in), cur+40)
	assert.False(t, ui.PredictionCard.OK())

	st, ui := uiAt(prep(none()), cur)
	assert.Equal(t, TodayUI{}, ui)
	assert.True(t, st.RequiresUserInput, "insufficient_anchor_data alone raises no banner")
}

func TestState_Today(t *testing.T) {
	cur := day(julyStart)
	s := prep(regular(cur, 28, 5))
	st, ui := s.Today(cur + 10)
	assert.Len(t, st.FutureCycles, 6)
	assert.Equal(t, s.TodayUI(s.Resolve(cur+10, cur+10)), ui)
	assert.Nil(t, s.Resolve(cur+10, cur+10).FutureCycles, "Resolve leaves future_cycles nil")

	in := regular(cur, 28, 5)
	in.Mode = ModePregnancy
	st, _ = prep(in).Today(cur + 10)
	assert.NotNil(t, st.FutureCycles)
	assert.Empty(t, st.FutureCycles)
}

func TestTodayUI_NoAllocs(t *testing.T) {
	cur := day(julyStart)
	s := prep(withPeriods(periodsEnding(cur, 5, 27, 29, 30)))
	st := s.Resolve(cur+17, cur+17)
	var ui TodayUI
	assert.Zero(t, testing.AllocsPerRun(200, func() { ui = s.TodayUI(st) }))
	_ = ui
}

func BenchmarkTodayUI(b *testing.B) {
	cur := day(julyStart)
	s := prep(withPeriods(periodsEnding(cur, 5, 27, 29, 30)))
	st := s.Resolve(cur+17, cur+17)
	b.ReportAllocs()
	var ui TodayUI
	for range b.N {
		ui = s.TodayUI(st)
	}
	_ = ui
}
