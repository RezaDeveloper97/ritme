package resolver

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Port of backend/tests/Unit/CycleStatusResolverTest.php — one subtest per PHP method, same name.

func period(start, end string, confirmed, estimated bool) model.History {
	h := model.History{PeriodStart: civildate.MustParse(start), IsConfirmed: confirmed, IsEstimated: estimated}
	if end != "" {
		h.PeriodEnd = civildate.MustParse(end)
	}
	return h
}

func logged(start, end string) model.History { return period(start, end, true, false) }

func resolveAt(periods []model.History, pr *model.Profile, target string, today ...string) Status {
	ref := target
	if len(today) > 0 {
		ref = today[0]
	}
	return Resolve(periods, pr, civildate.MustParse(target), civildate.MustParse(ref), metrics.Calculate(periods, pr))
}

func profile(cycle, duration *int, lmp string) *model.Profile {
	pr := &model.Profile{CycleDuration: cycle, PeriodDuration: duration}
	if lmp != "" {
		pr.LastPeriodStart = civildate.MustParse(lmp)
	}
	return pr
}

func p28() *model.Profile { return profile(model.Int(28), model.Int(5), "") }

// history21: three 21-day cycles of history, each with a 5-day logged bleed; the current period
// started 2026-01-11 with its end logged on 2026-01-15.
func history21() []model.History {
	return []model.History{
		logged("2025-11-09", "2025-11-13"),
		logged("2025-11-30", "2025-12-04"),
		logged("2025-12-21", "2025-12-25"),
		logged("2026-01-11", "2026-01-15"),
	}
}

func openPeriodHistory() []model.History {
	return []model.History{
		logged("2025-11-09", "2025-11-13"),
		logged("2025-11-30", "2025-12-04"),
		logged("2025-12-21", "2025-12-25"),
		logged("2026-01-11", ""),
	}
}

// day21 is day N of the current cycle (day 1 = 2026-01-11).
func day21(day int) string { return civildate.MustParse("2026-01-11").AddDays(day - 1).String() }

func iv(t *testing.T, v *int) int {
	t.Helper()
	require.NotNil(t, v)
	return *v
}

func TestCycleStatusResolver(t *testing.T) {
	t.Run("test_21_day_cycle_walk_matches_the_spec_example", func(t *testing.T) {
		expected := map[int]enums.CycleSubphase{
			1: enums.CycleSubphaseMenstrual, 2: enums.CycleSubphaseMenstrual,
			3: enums.CycleSubphaseMenstrual, 4: enums.CycleSubphaseMenstrual,
			5: enums.CycleSubphaseMenstrual,
			6: enums.CycleSubphaseHighFertility, 7: enums.CycleSubphaseHighFertility,
			8:  enums.CycleSubphaseOvulationLikely,
			9:  enums.CycleSubphasePostOvulation,
			10: enums.CycleSubphaseEarlyLuteal, 11: enums.CycleSubphaseEarlyLuteal,
			12: enums.CycleSubphaseEarlyLuteal,
			13: enums.CycleSubphaseMidLuteal, 14: enums.CycleSubphaseMidLuteal,
			15: enums.CycleSubphaseMidLuteal,
			16: enums.CycleSubphaseLateLuteal, 17: enums.CycleSubphaseLateLuteal,
			18: enums.CycleSubphaseLateLuteal,
			19: enums.CycleSubphasePmsPossible, 20: enums.CycleSubphasePmsPossible,
			21: enums.CycleSubphasePmsPossible,
		}
		for day, sub := range expected {
			s := resolveAt(history21(), p28(), day21(day))
			assert.Equal(t, sub, s.Subphase, "day %d", day)
			assert.Equal(t, day, iv(t, s.CycleDay), "cycle_day on day %d", day)
		}

		s := resolveAt(history21(), p28(), day21(22))
		assert.Equal(t, enums.MainPhasePeriodExpected, s.MainPhase)
		assert.Equal(t, 0, iv(t, s.DaysLate))
		assert.Equal(t, 0, iv(t, s.DaysToPeriod))
	})

	t.Run("test_menstrual_day_with_logged_end_is_user_logged_and_high_confidence", func(t *testing.T) {
		s := resolveAt(history21(), p28(), day21(3))

		assert.Equal(t, enums.MainPhaseMenstrual, s.MainPhase)
		assert.Equal(t, enums.ResolutionSourceUserLogged, s.ResolutionSource)
		assert.False(t, s.ResolutionSource.IsPredicted())
		assert.True(t, s.CurrentPeriodEndIsConfirmed)
		assert.Equal(t, enums.DataQualityLevelGood, s.DataQuality)
		assert.Equal(t, enums.ConfidenceLevelHigh, s.Confidence)
		assert.Equal(t, enums.FertilityLevelLow, s.FertilityLevel)
	})

	t.Run("test_luteal_day_resolves_via_prediction_even_with_full_logs", func(t *testing.T) {
		s := resolveAt(history21(), p28(), day21(13))

		assert.Equal(t, enums.MainPhaseLuteal, s.MainPhase)
		assert.Equal(t, enums.ResolutionSourcePrediction, s.ResolutionSource)
		assert.True(t, s.ResolutionSource.IsPredicted())
		assert.Equal(t, enums.ConfidenceLevelHigh, s.Confidence)
		assert.Contains(t, s.Warnings, string(enums.CycleWarningPredictionBasedOutput))
	})

	t.Run("test_open_period_within_soft_cap_is_menstrual_with_assumed_end", func(t *testing.T) {
		s := resolveAt(openPeriodHistory(), p28(), day21(4))

		assert.Equal(t, enums.MainPhaseMenstrual, s.MainPhase)
		assert.Equal(t, enums.CycleSubphaseMenstrual, s.Subphase)
		assert.Equal(t, enums.ResolutionSourceUserLoggedWithAssumedEnd, s.ResolutionSource)
		assert.False(t, s.CurrentPeriodEndIsConfirmed)
		assert.Equal(t, "assumed_from_effective_duration", string(s.CurrentPeriodEndSource))
		assert.NotContains(t, s.Warnings, string(enums.CycleWarningPeriodEndMissing))
	})

	t.Run("test_open_period_within_warning_cap_is_menstrual_possible", func(t *testing.T) {
		s := resolveAt(openPeriodHistory(), p28(), day21(7))

		assert.Equal(t, enums.MainPhaseMenstrual, s.MainPhase)
		assert.Equal(t, enums.CycleSubphaseMenstrualPossible, s.Subphase)
		assert.Contains(t, s.Warnings, string(enums.CycleWarningPeriodEndMissing))
		assert.NotContains(t, s.Warnings, string(enums.CycleWarningPeriodEndMissingWarningCapExceeded))
		assert.False(t, s.RequiresUserInput)
	})

	t.Run("test_open_period_past_warning_cap_degrades_quality_and_asks_for_input", func(t *testing.T) {
		s := resolveAt(openPeriodHistory(), p28(), day21(10))

		assert.Equal(t, enums.MainPhaseMenstrual, s.MainPhase)
		assert.Equal(t, enums.CycleSubphaseMenstrualPossible, s.Subphase)
		assert.Contains(t, s.Warnings, string(enums.CycleWarningPeriodEndMissingWarningCapExceeded))
		assert.Equal(t, enums.DataQualityLevelPoor, s.DataQuality)
		assert.Equal(t, enums.ConfidenceLevelLow, s.Confidence)
		assert.True(t, s.RequiresUserInput)
	})

	t.Run("test_open_period_past_hard_cap_resolves_the_cycle_onward", func(t *testing.T) {
		s := resolveAt(openPeriodHistory(), p28(), day21(13))

		assert.NotEqual(t, enums.MainPhaseMenstrual, s.MainPhase)
		assert.NotEqual(t, enums.MainPhaseUnknown, s.MainPhase)
		assert.Equal(t, enums.MainPhaseLuteal, s.MainPhase)
		assert.Contains(t, s.Warnings, string(enums.CycleWarningPeriodEndMissingHardCapExceeded))
		assert.Contains(t, s.ConfidenceReasons, ReasonMissingEndBeyondHardCap)
		assert.Equal(t, enums.DataQualityLevelPoor, s.DataQuality)
		assert.True(t, s.RequiresUserInput)
	})

	t.Run("test_period_expected_at_seven_days_late_is_calm", func(t *testing.T) {
		s := resolveAt(history21(), p28(), day21(29))

		assert.Equal(t, enums.MainPhasePeriodExpected, s.MainPhase)
		assert.Equal(t, enums.CycleSubphasePeriodExpected, s.Subphase)
		assert.Equal(t, 7, iv(t, s.DaysLate))
		assert.Equal(t, enums.ResolutionSourcePrediction, s.ResolutionSource)
		assert.NotContains(t, s.Warnings, string(enums.CycleWarningPredictedPeriodOverdue))
		assert.Equal(t, enums.FertilityLevelUnknown, s.FertilityLevel)
	})

	t.Run("test_period_expected_at_eight_days_late_warns_and_drops_confidence", func(t *testing.T) {
		s := resolveAt(history21(), p28(), day21(30))

		assert.Equal(t, enums.MainPhasePeriodExpected, s.MainPhase)
		assert.Equal(t, 8, iv(t, s.DaysLate))
		assert.Contains(t, s.Warnings, string(enums.CycleWarningPredictedPeriodOverdue))
		assert.Equal(t, enums.ConfidenceLevelLow, s.Confidence)
		assert.Contains(t, s.ConfidenceReasons, ReasonPeriodOverdue)
	})

	t.Run("test_period_expected_past_fourteen_days_late_goes_unknown_but_keeps_anchors", func(t *testing.T) {
		s := resolveAt(history21(), p28(), day21(37))

		assert.Equal(t, enums.MainPhaseUnknown, s.MainPhase)
		assert.Equal(t, 15, iv(t, s.DaysLate))
		assert.Equal(t, 0, iv(t, s.DaysToPeriod))
		assert.Nil(t, s.DaysToOvulation)
		assert.Equal(t, 37, iv(t, s.CycleDay))
		assert.False(t, s.PredictedNextPeriodStart.IsZero())
		assert.False(t, s.EstimatedOvulationDate.IsZero())
		assert.Contains(t, s.Warnings, string(enums.CycleWarningCycleUnresolvedAfterExpectedPeriod))
		assert.True(t, s.RequiresUserInput)
		assert.Equal(t, enums.DataQualityLevelPoor, s.DataQuality)
		assert.Equal(t, enums.ConfidenceLevelLow, s.Confidence)
	})

	t.Run("test_a_newly_logged_period_cancels_period_expected", func(t *testing.T) {
		periods := append(history21(), logged(day21(23), ""))
		s := resolveAt(periods, p28(), day21(23))

		assert.Equal(t, enums.MainPhaseMenstrual, s.MainPhase)
		assert.Equal(t, 1, iv(t, s.CycleDay))
		assert.Equal(t, 0, iv(t, s.DaysLate))
		assert.Equal(t, enums.PeriodStartSourceUserLogged, s.CurrentPeriodStartSource)
	})

	t.Run("test_long_bleed_empties_the_display_fertile_window", func(t *testing.T) {
		periods := []model.History{
			logged("2025-11-09", "2025-11-18"),
			logged("2025-11-30", "2025-12-09"),
			logged("2025-12-21", "2025-12-30"),
			logged("2026-01-11", "2026-01-20"),
		}
		for day := 1; day <= 21; day++ {
			s := resolveAt(periods, p28(), day21(day))
			assert.NotEqual(t, enums.MainPhaseFertile, s.MainPhase, "day %d", day)
		}

		day8 := resolveAt(periods, p28(), day21(8))
		assert.Equal(t, enums.MainPhaseMenstrual, day8.MainPhase)
	})

	t.Run("test_28_day_cycle_follicular_split_and_fertile_tiers", func(t *testing.T) {
		periods := []model.History{
			logged("2025-10-19", "2025-10-23"),
			logged("2025-11-16", "2025-11-20"),
			logged("2025-12-14", "2025-12-18"),
			logged("2026-01-11", "2026-01-15"),
		}
		expected := map[int]enums.CycleSubphase{
			6:  enums.CycleSubphaseEarlyFollicular,
			7:  enums.CycleSubphaseMidFollicular,
			8:  enums.CycleSubphaseLateFollicularTransition,
			9:  enums.CycleSubphaseLateFollicularTransition,
			10: enums.CycleSubphaseFertileRising,
			11: enums.CycleSubphaseFertileRising,
			12: enums.CycleSubphaseFertileRising,
			13: enums.CycleSubphaseHighFertility,
			14: enums.CycleSubphaseHighFertility,
			15: enums.CycleSubphaseOvulationLikely,
			16: enums.CycleSubphasePostOvulation,
			17: enums.CycleSubphaseEarlyLuteal,
			19: enums.CycleSubphaseEarlyLuteal,
			20: enums.CycleSubphaseMidLuteal,
			22: enums.CycleSubphaseMidLuteal,
			23: enums.CycleSubphaseLateLuteal,
			25: enums.CycleSubphaseLateLuteal,
			26: enums.CycleSubphasePmsPossible,
			28: enums.CycleSubphasePmsPossible,
		}
		for day, sub := range expected {
			s := resolveAt(periods, p28(), day21(day))
			assert.Equal(t, sub, s.Subphase, "day %d", day)
		}

		peak := resolveAt(periods, p28(), day21(15))
		assert.Equal(t, enums.FertilityLevelPeak, peak.FertilityLevel)
		assert.Equal(t, 0, iv(t, peak.DaysToOvulation))
	})

	t.Run("test_onboarding_only_resolve_is_onboarding_based_and_low_confidence", func(t *testing.T) {
		s := resolveAt(nil, profile(model.Int(28), model.Int(5), "2026-01-11"), day21(13))

		assert.Equal(t, enums.PeriodStartSourceOnboardingDeclared, s.CurrentPeriodStartSource)
		assert.Equal(t, enums.ResolutionSourceOnboardingBased, s.ResolutionSource)
		assert.True(t, s.ResolutionSource.IsPredicted())
		assert.Equal(t, enums.ConfidenceLevelLow, s.Confidence)
		assert.Equal(t, enums.DataQualityLevelPartial, s.DataQuality)
		assert.Contains(t, s.Warnings, string(enums.CycleWarningOnboardingDataUsed))
		assert.Contains(t, s.Warnings, string(enums.CycleWarningLowHistory))
	})

	t.Run("test_defaults_only_resolve_is_default_based", func(t *testing.T) {
		s := resolveAt(nil, profile(nil, nil, "2026-01-11"), day21(2))

		assert.Equal(t, enums.MainPhaseMenstrual, s.MainPhase)
		assert.Equal(t, enums.ResolutionSourceDefaultBased, s.ResolutionSource)
		assert.Equal(t, 28, s.EffectiveCycleLength)
		assert.Equal(t, 5, s.EffectivePeriodLength)
		assert.Contains(t, s.Warnings, string(enums.CycleWarningDefaultValuesUsed))
		assert.Equal(t, enums.ConfidenceLevelLow, s.Confidence)
	})

	t.Run("test_no_anchor_at_all_is_insufficient_and_unknown", func(t *testing.T) {
		s := resolveAt(nil, nil, "2026-01-15")

		assert.Equal(t, enums.MainPhaseUnknown, s.MainPhase)
		assert.Nil(t, s.CycleDay)
		assert.Equal(t, enums.DataQualityLevelInsufficient, s.DataQuality)
		assert.Equal(t, enums.ConfidenceLevelUnknown, s.Confidence)
		assert.Equal(t, enums.ResolutionSourceUnknown, s.ResolutionSource)
		assert.True(t, s.RequiresUserInput)
		assert.Contains(t, s.Warnings, string(enums.CycleWarningInsufficientAnchorData))
		assert.True(t, s.CurrentPeriodStart.IsZero())
	})

	t.Run("test_future_dates_roll_the_anchor_forward_as_predicted_reference", func(t *testing.T) {
		s := resolveAt(history21(), p28(), day21(23), day21(13))

		assert.Equal(t, enums.PeriodStartSourcePredictedReference, s.CurrentPeriodStartSource)
		assert.Equal(t, 2, iv(t, s.CycleDay))
		assert.Equal(t, enums.MainPhaseMenstrual, s.MainPhase)
		assert.Equal(t, enums.ResolutionSourcePrediction, s.ResolutionSource)
		assert.True(t, s.ResolutionSource.IsPredicted())
		assert.Equal(t, 0, iv(t, s.DaysLate))
	})

	t.Run("test_cycle_day_is_one_based", func(t *testing.T) {
		s := resolveAt(history21(), p28(), day21(1))

		assert.Equal(t, 1, iv(t, s.CycleDay))
		assert.Equal(t, enums.MainPhaseMenstrual, s.MainPhase)
	})
}

func TestStatusToAPI(t *testing.T) {
	s := resolveAt(nil, nil, "2026-01-15")
	api := s.ToAPI()
	assert.Equal(t, []string{ReasonInsufficientAnchorData}, api.ConfidenceReasons)
	assert.Equal(t, []string{string(enums.CycleWarningInsufficientAnchorData)}, api.Warnings)
	assert.True(t, api.IsPredicted)
	assert.Equal(t, "2026-01-15", fmt.Sprint(api.Date))
}
