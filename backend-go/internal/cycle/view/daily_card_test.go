package view

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/enums"
)

// Port of backend/tests/Unit/DailyCardBuilderTest.php — one subtest per PHP method, same name.

// cardOpts mirrors the PHP test's option array; empty fields take the PHP defaults.
type cardOpts struct {
	selected, today, nextStart, ovulation, locale string
	cycleDay                                      int
	subphase                                      enums.CycleSubphase
	open                                          *OpenPeriodState
	loggedDay                                     *int
	loggedClosed                                  bool
}

func or(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func card(o cardOpts) DailyCard {
	in := CardInput{
		Selected:           d(or(o.selected, "2026-07-15")),
		Today:              d(or(o.today, "2026-07-15")),
		CycleDay:           8,
		Subphase:           enums.CycleSubphaseMidFollicular,
		PredictedNextStart: d(or(o.nextStart, "2026-08-04")),
		OpenPeriod:         NoOpenPeriod(),
		LoggedPeriodDay:    o.loggedDay,
		LoggedPeriodClosed: o.loggedClosed,
		Locale:             or(o.locale, "en"),
	}
	if o.cycleDay != 0 {
		in.CycleDay = o.cycleDay
	}
	if o.subphase != "" {
		in.Subphase = o.subphase
	}
	if o.open != nil {
		in.OpenPeriod = *o.open
	}
	if o.ovulation != "" {
		in.EstimatedOvulation = d(o.ovulation)
	}
	return BuildDailyCard(in)
}

func openState(endOverdue bool) *OpenPeriodState {
	day := 3
	if endOverdue {
		day = 10
	}
	return &OpenPeriodState{
		HasOpenPeriod:            true,
		StartDate:                d("2026-07-13"),
		MenstrualDay:             model.Int(day),
		DisplayAsActiveMenstrual: !endOverdue,
		EndOverdue:               endOverdue,
		PastHardCap:              false,
		AssumedEndDate:           d("2026-07-17"),
		DataQualityFlags:         []string{},
	}
}

func actionTypes(c DailyCard) []string {
	var types []string
	if c.PrimaryAction != nil {
		types = append(types, c.PrimaryAction.Type)
	}
	for _, a := range c.SecondaryActions {
		types = append(types, a.Type)
	}
	return types
}

func TestDailyCardBuilder(t *testing.T) {
	t.Run("test_inside_a_closed_logged_period_is_actual", func(t *testing.T) {
		c := card(cardOpts{loggedDay: model.Int(2), loggedClosed: true})

		assert.Equal(t, enums.DataStatusActual, c.DataStatus)
		assert.Equal(t, "Day 2 of your period", c.Title)
		assert.Equal(t, enums.FertilityLevelLow, c.FertilityLevel)
	})

	t.Run("test_open_period_within_expected_length_is_incomplete", func(t *testing.T) {
		c := card(cardOpts{loggedDay: model.Int(3), open: openState(false)})

		assert.Equal(t, enums.DataStatusIncomplete, c.DataStatus)
		assert.Equal(t, "Day 3 of your period", c.Title)
		assert.Equal(t, []string{"log_symptoms", "log_period_end", "view_details"}, actionTypes(c))
	})

	t.Run("test_open_period_past_expected_length_asks_for_the_end_today", func(t *testing.T) {
		c := card(cardOpts{loggedDay: model.Int(10), open: openState(true), today: "2026-07-15", selected: "2026-07-15"})

		assert.Equal(t, enums.DataStatusIncomplete, c.DataStatus)
		assert.Equal(t, "Period end not logged yet", c.Title)
		require.NotNil(t, c.PrimaryAction)
		assert.Equal(t, "log_period_end", c.PrimaryAction.Type)
	})

	t.Run("test_predicted_period_due_needs_confirmation", func(t *testing.T) {
		c := card(cardOpts{selected: "2026-07-15", today: "2026-07-15", nextStart: "2026-07-15"})

		assert.Equal(t, enums.DataStatusNeedsConfirmation, c.DataStatus)
		assert.Equal(t, "Your predicted period is due", c.Title)
		require.NotNil(t, c.PrimaryAction)
		assert.Equal(t, "confirm_period_start", c.PrimaryAction.Type)
	})

	t.Run("test_one_day_overdue_reports_the_gap", func(t *testing.T) {
		c := card(cardOpts{selected: "2026-07-15", today: "2026-07-15", nextStart: "2026-07-14"})

		assert.Equal(t, enums.DataStatusNeedsConfirmation, c.DataStatus)
		assert.Equal(t, "1 day(s) past the predicted date", c.Title)
	})

	t.Run("test_several_days_overdue_switches_message", func(t *testing.T) {
		c := card(cardOpts{selected: "2026-07-15", today: "2026-07-15", nextStart: "2026-07-11"})

		assert.Equal(t, "Period start not logged yet", c.Title)
		require.NotNil(t, c.PrimaryAction)
		assert.Equal(t, "log_period_start", c.PrimaryAction.Type)
	})

	t.Run("test_countdown_to_the_next_period", func(t *testing.T) {
		c := card(cardOpts{
			selected: "2026-07-20", today: "2026-07-20",
			nextStart: "2026-07-25", ovulation: "2026-07-11",
			subphase: enums.CycleSubphaseLateLuteal,
		})

		assert.Equal(t, enums.DataStatusPredicted, c.DataStatus)
		assert.Equal(t, "About 5 days to your next period", c.Title)
	})

	t.Run("test_fertile_window_uses_the_subphase_fertility", func(t *testing.T) {
		c := card(cardOpts{
			selected: "2026-07-12", today: "2026-07-12",
			ovulation: "2026-07-14", nextStart: "2026-07-28",
			subphase: enums.CycleSubphaseHighFertility,
		})

		assert.Equal(t, "You're likely in your fertile window", c.Title)
		assert.Equal(t, enums.FertilityLevelHigh, c.FertilityLevel)
	})

	t.Run("test_ovulation_day", func(t *testing.T) {
		c := card(cardOpts{
			selected: "2026-07-14", today: "2026-07-14",
			ovulation: "2026-07-14", nextStart: "2026-07-28",
			subphase: enums.CycleSubphaseOvulationLikely,
		})

		assert.Equal(t, "Ovulation is likely near", c.Title)
		assert.Equal(t, enums.FertilityLevelVeryHigh, c.FertilityLevel)
	})

	t.Run("test_plain_day_counts_down_to_the_fertile_window", func(t *testing.T) {
		c := card(cardOpts{
			selected: "2026-07-08", today: "2026-07-08", cycleDay: 8,
			ovulation: "2026-07-14", nextStart: "2026-07-28",
		})

		assert.Equal(t, "1 day(s) to your fertile window", c.Title)
		assert.Equal(t, enums.DataStatusPredicted, c.DataStatus)
	})

	t.Run("test_plain_day_past_the_window_counts_down_to_the_period", func(t *testing.T) {
		c := card(cardOpts{
			selected: "2026-07-18", today: "2026-07-18", cycleDay: 18,
			ovulation: "2026-07-14", nextStart: "2026-07-28",
			subphase: enums.CycleSubphaseEarlyLuteal,
		})

		assert.Equal(t, "10 day(s) to your next period", c.Title)
		assert.Equal(t, enums.DataStatusPredicted, c.DataStatus)
	})

	t.Run("test_predicted_period_day_offers_confirming_the_start", func(t *testing.T) {
		c := card(cardOpts{
			selected: "2026-06-01", today: "2026-07-15", cycleDay: 1,
			nextStart: "2026-07-28", subphase: enums.CycleSubphaseMenstrual,
		})

		assert.Equal(t, enums.DataStatusNeedsConfirmation, c.DataStatus)
		assert.Equal(t, "Day 1 of your predicted period", c.Title)
		assert.Equal(t, []string{"confirm_period_start", "period_not_started", "view_details"}, actionTypes(c))
	})

	t.Run("test_predicted_period_day_in_the_future_stays_a_prediction", func(t *testing.T) {
		c := card(cardOpts{
			selected: "2026-08-25", today: "2026-07-15", cycleDay: 2,
			nextStart: "2026-07-28", subphase: enums.CycleSubphaseMenstrual,
		})

		assert.Equal(t, enums.DataStatusPredicted, c.DataStatus)
		require.NotNil(t, c.PrimaryAction)
		assert.Equal(t, "set_reminder", c.PrimaryAction.Type)
	})

	t.Run("test_future_day_never_says_not_started_and_offers_a_reminder", func(t *testing.T) {
		c := card(cardOpts{
			selected: "2026-08-15", today: "2026-07-15", cycleDay: 12,
			nextStart: "2026-07-28", ovulation: "2026-07-14",
		})

		assert.Equal(t, enums.DataStatusPredicted, c.DataStatus)
		assert.Equal(t, "Day 12 of the predicted cycle", c.Title)
		require.NotNil(t, c.PrimaryAction)
		assert.Equal(t, "set_reminder", c.PrimaryAction.Type)
	})

	t.Run("test_persian_locale_localises_title_and_digits", func(t *testing.T) {
		c := card(cardOpts{loggedDay: model.Int(2), loggedClosed: true, locale: "fa"})

		assert.Equal(t, "روز ۲ پریود", c.Title)
		require.NotNil(t, c.PrimaryAction)
		assert.Equal(t, "ثبت علائم امروز", c.PrimaryAction.Label)
	})
}
