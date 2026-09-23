package view

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

var d = civildate.MustParse

// Port of backend/tests/Unit/CyclePredictionServiceTest.php.
func TestCyclePredictionService(t *testing.T) {
	t.Run("test_predicts_next_period_ovulation_and_fertile_window", func(t *testing.T) {
		p := Predict(d("2026-07-01"), 28, 5, d("2026-07-15"), enums.EffectiveSourceRecentValidCycles)

		assert.Equal(t, "2026-07-01", p.CurrentCycleStart.String())
		assert.Equal(t, "2026-07-29", p.NextPeriodStart.String())
		assert.Equal(t, "2026-08-02", p.NextPeriodEnd.String())
		assert.Equal(t, "2026-07-15", p.EstimatedOvulationDate.String())
		assert.Equal(t, "2026-07-10", p.FertileWindowStart.String())
		assert.Equal(t, "2026-07-16", p.FertileWindowEnd.String())
		assert.Equal(t, enums.EffectiveSourceRecentValidCycles, p.Source)
	})

	t.Run("test_projects_the_anchor_forward_when_several_cycles_have_passed", func(t *testing.T) {
		p := Predict(d("2026-07-01"), 28, 5, d("2026-09-10"), enums.EffectiveSourceProfile)

		assert.Equal(t, "2026-08-26", p.CurrentCycleStart.String())
		assert.Equal(t, "2026-09-23", p.NextPeriodStart.String())
	})

	t.Run("test_uses_the_period_duration_for_the_predicted_end", func(t *testing.T) {
		p := Predict(d("2026-07-01"), 30, 7, d("2026-07-01"), enums.EffectiveSourceRecentValidCycles)

		assert.Equal(t, "2026-07-31", p.NextPeriodStart.String())
		assert.Equal(t, "2026-08-06", p.NextPeriodEnd.String())
		assert.Equal(t, "2026-07-17", p.EstimatedOvulationDate.String())
	})
}

func confidenceMetrics(validCycles int, variability enums.CycleVariability) metrics.Metrics {
	var calculated *int
	if validCycles >= 3 {
		calculated = model.Int(28)
	}
	return metrics.Metrics{
		ProfileCycleLength:      model.Int(28),
		ProfilePeriodDuration:   model.Int(5),
		CalculatedCycleLength:   calculated,
		EffectiveCycleLength:    28,
		EffectivePeriodDuration: 5,
		CycleLengthSource:       enums.EffectiveSourceProfile,
		PeriodDurationSource:    enums.EffectiveSourceProfile,
		ValidCyclesCount:        validCycles,
		RegularityStatus:        enums.RegularityStatusNotEnoughData,
		Variability:             variability,
		RecentValidCycleLengths: []int{},
	}
}

// Port of backend/tests/Unit/CycleConfidenceCalculatorTest.php.
func TestCycleConfidenceCalculator(t *testing.T) {
	t.Run("test_three_regular_cycles_give_high_confidence", func(t *testing.T) {
		c := ConfidenceForPrediction(confidenceMetrics(3, enums.CycleVariabilityRegular), false)

		assert.Equal(t, enums.ConfidenceLevelHigh, c.Level)
		assert.Contains(t, c.Reasons, ReasonThreeValidCycles)
		assert.Contains(t, c.Reasons, ReasonLowVariation)
	})

	t.Run("test_three_irregular_cycles_are_only_medium", func(t *testing.T) {
		c := ConfidenceForPrediction(confidenceMetrics(4, enums.CycleVariabilityIrregular), false)

		assert.Equal(t, enums.ConfidenceLevelMedium, c.Level)
		assert.Contains(t, c.Reasons, ReasonCycleVariation)
	})

	t.Run("test_one_or_two_cycles_are_medium", func(t *testing.T) {
		c := ConfidenceForPrediction(confidenceMetrics(2, enums.CycleVariabilityRegular), false)

		assert.Equal(t, enums.ConfidenceLevelMedium, c.Level)
		assert.Contains(t, c.Reasons, ReasonNotEnoughCycles)
	})

	t.Run("test_profile_only_is_low", func(t *testing.T) {
		c := ConfidenceForPrediction(confidenceMetrics(0, enums.CycleVariabilityRegular), false)

		assert.Equal(t, enums.ConfidenceLevelLow, c.Level)
		assert.Contains(t, c.Reasons, ReasonProfileOnly)
	})

	t.Run("test_a_long_missing_end_downgrades_and_annotates", func(t *testing.T) {
		c := ConfidenceForPrediction(confidenceMetrics(3, enums.CycleVariabilityRegular), true)

		assert.Equal(t, enums.ConfidenceLevelMedium, c.Level)
		assert.Contains(t, c.Reasons, ReasonEndMissingLong)
	})
}

// Port of backend/tests/Unit/OpenPeriodEvaluatorTest.php.
func TestOpenPeriodEvaluator(t *testing.T) {
	t.Run("test_no_open_period_returns_the_empty_state", func(t *testing.T) {
		s := EvaluateOpenPeriod(civildate.Date{}, 5, model.Int(5), d("2026-07-10"))

		assert.False(t, s.HasOpenPeriod)
		assert.Nil(t, s.MenstrualDay)
		assert.False(t, s.DisplayAsActiveMenstrual)
	})

	t.Run("test_within_expected_length_is_shown_as_active", func(t *testing.T) {
		s := EvaluateOpenPeriod(d("2026-07-01"), 5, model.Int(5), d("2026-07-03"))

		assert.True(t, s.HasOpenPeriod)
		require.NotNil(t, s.MenstrualDay)
		assert.Equal(t, 3, *s.MenstrualDay)
		assert.True(t, s.DisplayAsActiveMenstrual)
		assert.False(t, s.EndOverdue)
		assert.False(t, s.PastHardCap)
		assert.Equal(t, "2026-07-05", s.AssumedEndDate.String())
		assert.Equal(t, []string{}, s.DataQualityFlags)
	})

	t.Run("test_past_expected_length_flags_incomplete_and_asks_for_the_end", func(t *testing.T) {
		s := EvaluateOpenPeriod(d("2026-07-01"), 5, model.Int(5), d("2026-07-10"))

		require.NotNil(t, s.MenstrualDay)
		assert.Equal(t, 10, *s.MenstrualDay)
		assert.False(t, s.DisplayAsActiveMenstrual)
		assert.True(t, s.EndOverdue)
		assert.False(t, s.PastHardCap)
		assert.True(t, s.HasFlag(enums.DataQualityFlagIncompleteEndMissing))
	})

	t.Run("test_past_hard_cap_stops_presenting_an_active_period", func(t *testing.T) {
		s := EvaluateOpenPeriod(d("2026-07-01"), 5, model.Int(5), d("2026-07-14"))

		require.NotNil(t, s.MenstrualDay)
		assert.Equal(t, 14, *s.MenstrualDay)
		assert.False(t, s.DisplayAsActiveMenstrual)
		assert.True(t, s.EndOverdue)
		assert.True(t, s.PastHardCap)
	})

	t.Run("test_max_expected_follows_the_profile_period_duration", func(t *testing.T) {
		s := EvaluateOpenPeriod(d("2026-07-01"), 5, model.Int(7), d("2026-07-09"))

		require.NotNil(t, s.MenstrualDay)
		assert.Equal(t, 9, *s.MenstrualDay)
		assert.True(t, s.DisplayAsActiveMenstrual)
	})

	t.Run("test_a_future_start_is_not_an_ongoing_bleed", func(t *testing.T) {
		s := EvaluateOpenPeriod(d("2026-07-20"), 5, model.Int(5), d("2026-07-10"))

		assert.False(t, s.HasOpenPeriod)
	})

	t.Run("test_hard_cap_dominates_a_long_profile_period_duration", func(t *testing.T) {
		s := EvaluateOpenPeriod(d("2026-07-01"), 5, model.Int(12), d("2026-07-13"))

		require.NotNil(t, s.MenstrualDay)
		assert.Equal(t, 13, *s.MenstrualDay)
		assert.False(t, s.DisplayAsActiveMenstrual)
		assert.True(t, s.EndOverdue)
		assert.True(t, s.PastHardCap)
	})
}
