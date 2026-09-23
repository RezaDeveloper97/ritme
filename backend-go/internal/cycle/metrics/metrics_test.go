package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Port of backend/tests/Unit/CycleMetricsCalculatorTest.php — one subtest per PHP method, same name.

// period is a confirmed logged period unless confirmed is overridden (PHP period()).
func period(start string, end string, confirmed bool) model.History {
	h := model.History{PeriodStart: civildate.MustParse(start), IsConfirmed: confirmed}
	if end != "" {
		h.PeriodEnd = civildate.MustParse(end)
	}
	return h
}

func p(start string) model.History           { return period(start, "", true) }
func pe(start, end string) model.History     { return period(start, end, true) }
func unconfirmed(start string) model.History { return period(start, "", false) }
func profile(cycle, duration *int) *model.Profile {
	return &model.Profile{CycleDuration: cycle, PeriodDuration: duration}
}
func metricsFor(periods []model.History, pr *model.Profile) Metrics { return Calculate(periods, pr) }

func TestCycleMetricsCalculator(t *testing.T) {
	t.Run("test_falls_back_to_system_default_when_no_data", func(t *testing.T) {
		m := metricsFor(nil, nil)

		assert.Equal(t, 28, m.EffectiveCycleLength)
		assert.Equal(t, 5, m.EffectivePeriodDuration)
		assert.Equal(t, enums.EffectiveSourceDefault, m.CycleLengthSource)
		assert.Equal(t, enums.EffectiveSourceDefault, m.PeriodDurationSource)
		assert.Nil(t, m.CalculatedCycleLength)
		assert.Equal(t, enums.RegularityStatusNotEnoughData, m.RegularityStatus)
	})

	t.Run("test_uses_median_of_two_valid_cycles_over_the_profile", func(t *testing.T) {
		m := metricsFor([]model.History{p("2026-01-01"), p("2026-01-31"), p("2026-03-02")}, profile(model.Int(28), model.Int(5)))

		assert.Equal(t, 30, m.EffectiveCycleLength)
		assert.Equal(t, enums.EffectiveSourceRecentValidCycles, m.CycleLengthSource)
		require.NotNil(t, m.CalculatedCycleLength)
		assert.Equal(t, 30, *m.CalculatedCycleLength)
		assert.Equal(t, 2, m.ValidCyclesCount)
		require.NotNil(t, m.CycleVariabilityRange)
		assert.Equal(t, 0, *m.CycleVariabilityRange) // 30 − 30
		assert.Equal(t, enums.RegularityStatusNotEnoughData, m.RegularityStatus)
	})

	t.Run("test_single_valid_cycle_is_used_as_is_and_variability_is_null", func(t *testing.T) {
		m := metricsFor([]model.History{p("2026-01-01"), p("2026-01-30")}, profile(model.Int(28), model.Int(5)))

		assert.Equal(t, 29, m.EffectiveCycleLength)
		assert.Equal(t, enums.EffectiveSourceRecentValidCycles, m.CycleLengthSource)
		assert.Equal(t, 1, m.ValidCyclesCount)
		assert.Nil(t, m.CycleVariabilityRange)
	})

	t.Run("test_uses_median_of_last_three_valid_cycles", func(t *testing.T) {
		m := metricsFor([]model.History{
			p("2026-01-01"), p("2026-01-29"), p("2026-02-28"), p("2026-04-04"),
		}, profile(model.Int(28), model.Int(5)))

		require.NotNil(t, m.CalculatedCycleLength)
		assert.Equal(t, 30, *m.CalculatedCycleLength)
		assert.Equal(t, 30, m.EffectiveCycleLength)
		assert.Equal(t, enums.EffectiveSourceRecentValidCycles, m.CycleLengthSource)
		assert.Equal(t, 3, m.ValidCyclesCount)
	})

	t.Run("test_outlier_cycles_are_excluded_from_the_median", func(t *testing.T) {
		m := metricsFor([]model.History{
			p("2026-01-01"), p("2026-02-20"), p("2026-03-20"), p("2026-04-19"), p("2026-05-21"),
		}, profile(model.Int(28), model.Int(5)))

		assert.Equal(t, 3, m.ValidCyclesCount)
		require.NotNil(t, m.CalculatedCycleLength)
		assert.Equal(t, 30, *m.CalculatedCycleLength) // median of [28,30,32]
	})

	t.Run("test_period_duration_is_calculated_independently_of_cycle_length", func(t *testing.T) {
		m := metricsFor([]model.History{
			pe("2026-01-01", "2026-01-04"), pe("2026-01-31", "2026-02-04"), pe("2026-03-02", "2026-03-07"),
		}, profile(model.Int(28), model.Int(9)))

		assert.Equal(t, 2, m.ValidCyclesCount)
		require.NotNil(t, m.CalculatedCycleLength)
		assert.Equal(t, 30, *m.CalculatedCycleLength)
		assert.Equal(t, enums.EffectiveSourceRecentValidCycles, m.CycleLengthSource)

		assert.Equal(t, 3, m.ValidPeriodDurationsCount)
		require.NotNil(t, m.CalculatedPeriodDuration)
		assert.Equal(t, 5, *m.CalculatedPeriodDuration)
		assert.Equal(t, 5, m.EffectivePeriodDuration)
		assert.Equal(t, enums.EffectiveSourceRecentValidCycles, m.PeriodDurationSource)
	})

	t.Run("test_open_and_overlong_periods_do_not_feed_the_duration_median", func(t *testing.T) {
		m := metricsFor([]model.History{
			pe("2026-01-01", "2026-01-05"), p("2026-01-31"), pe("2026-03-02", "2026-03-20"),
		}, profile(model.Int(28), model.Int(5)))

		assert.Equal(t, 1, m.ValidPeriodDurationsCount)
		require.NotNil(t, m.CalculatedPeriodDuration)
		assert.Equal(t, 5, *m.CalculatedPeriodDuration)
		assert.Equal(t, 5, m.EffectivePeriodDuration)
		assert.Equal(t, enums.EffectiveSourceRecentValidCycles, m.PeriodDurationSource)
	})

	t.Run("test_unconfirmed_periods_are_ignored", func(t *testing.T) {
		m := metricsFor([]model.History{
			unconfirmed("2026-01-01"), unconfirmed("2026-01-29"), unconfirmed("2026-02-28"), unconfirmed("2026-04-04"),
		}, profile(model.Int(28), model.Int(5)))

		assert.Equal(t, 0, m.ValidCyclesCount)
		assert.Nil(t, m.CalculatedCycleLength)
		assert.Equal(t, enums.EffectiveSourceProfile, m.CycleLengthSource)
	})

	t.Run("test_regularity_reads_the_range_of_the_last_three_cycles", func(t *testing.T) {
		regular := metricsFor([]model.History{
			p("2026-01-01"), p("2026-01-29"), p("2026-02-27"), p("2026-03-29"),
		}, profile(model.Int(28), model.Int(5)))
		assert.Equal(t, enums.RegularityStatusRelativelyRegular, regular.RegularityStatus)

		irregular := metricsFor([]model.History{
			p("2026-01-01"), p("2026-01-28"), p("2026-03-03"), p("2026-04-14"),
		}, profile(model.Int(28), model.Int(5)))
		assert.Equal(t, enums.RegularityStatusIrregularPossible, irregular.RegularityStatus)
	})
}

// Go-only edge cases of the PHP semantics (not in the PHP suite).
func TestCalculateEdgeCases(t *testing.T) {
	t.Run("even median rounds half away from zero", func(t *testing.T) {
		m := metricsFor([]model.History{p("2026-01-01"), p("2026-01-29"), p("2026-02-27")}, nil) // 28, 29
		assert.Equal(t, 29, m.EffectiveCycleLength)
	})
	t.Run("stored zero profile values are missing", func(t *testing.T) {
		m := metricsFor(nil, profile(model.Int(0), model.Int(0)))
		assert.Nil(t, m.ProfileCycleLength)
		assert.Equal(t, enums.EffectiveSourceDefault, m.CycleLengthSource)
	})
	t.Run("negative profile value is floored at one", func(t *testing.T) {
		m := metricsFor(nil, profile(model.Int(-3), nil))
		assert.Equal(t, 1, m.EffectiveCycleLength)
		assert.Equal(t, enums.EffectiveSourceProfile, m.CycleLengthSource)
	})
	t.Run("only outliers", func(t *testing.T) {
		m := metricsFor([]model.History{p("2026-01-01"), p("2026-01-10"), p("2026-04-10")}, nil)
		assert.True(t, m.HasShortCycleOutlier)
		assert.True(t, m.HasLongCycleOutlier)
		assert.True(t, m.OnlyOutlierHistory)
	})
	t.Run("layers json shape", func(t *testing.T) {
		l := metricsFor(nil, profile(model.Int(28), nil)).Layers()
		assert.Nil(t, l.CalculatedValues.BasedOnCycles)
		assert.Equal(t, enums.EffectiveSourceProfile, l.EffectiveValues.Source)
		assert.Equal(t, enums.EffectiveSourceDefault, l.EffectiveValues.PeriodDurationSource)
	})
}
