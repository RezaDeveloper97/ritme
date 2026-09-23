package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/enums"
)

// Port of backend/tests/Unit/CyclePhaseMapperTest.php — canonical 28-day cycle, O = 14, 5-day bleed.

const (
	mapperOvulation = 14
	mapperCycle     = 28
	mapperBleed     = 5
)

func mapperPhase(day int) enums.CyclePhase {
	return PhaseMapper{}.PhaseFor(day, mapperOvulation, mapperBleed)
}

func mapperSubphase(day int) enums.CycleSubphase {
	return PhaseMapper{}.SubphaseFor(day, mapperOvulation, mapperCycle, mapperBleed)
}

func TestCyclePhaseMapper(t *testing.T) {
	t.Run("test_phase_boundaries_follow_the_ovulation_day", func(t *testing.T) {
		assert.Equal(t, enums.CyclePhaseMenstruation, mapperPhase(3))
		assert.Equal(t, enums.CyclePhaseFollicular, mapperPhase(6))
		assert.Equal(t, enums.CyclePhaseFollicular, mapperPhase(12))
		assert.Equal(t, enums.CyclePhaseFollicular, mapperPhase(13))
		assert.Equal(t, enums.CyclePhaseOvulation, mapperPhase(14))
		assert.Equal(t, enums.CyclePhaseOvulation, mapperPhase(15))
		assert.Equal(t, enums.CyclePhaseLuteal, mapperPhase(16))
		assert.Equal(t, enums.CyclePhaseLuteal, mapperPhase(28))
	})

	t.Run("test_subphase_walks_the_whole_cycle", func(t *testing.T) {
		expected := map[int]enums.CycleSubphase{
			1:  enums.CycleSubphaseMenstruation,
			5:  enums.CycleSubphaseMenstruation,
			6:  enums.CycleSubphaseEarlyFollicular,
			8:  enums.CycleSubphaseMidFollicular,
			9:  enums.CycleSubphaseFertileRising,
			11: enums.CycleSubphaseFertileRising,
			12: enums.CycleSubphaseHighFertility,
			13: enums.CycleSubphaseHighFertility,
			14: enums.CycleSubphaseOvulationLikely,
			15: enums.CycleSubphasePostOvulation,
			16: enums.CycleSubphaseEarlyLuteal,
			19: enums.CycleSubphaseEarlyLuteal,
			20: enums.CycleSubphaseMidLuteal,
			23: enums.CycleSubphaseMidLuteal,
			24: enums.CycleSubphaseLateLuteal,
			26: enums.CycleSubphaseLateLuteal,
			27: enums.CycleSubphasePmsPossible,
			28: enums.CycleSubphasePmsPossible,
		}
		for day, sub := range expected {
			assert.Equal(t, sub, mapperSubphase(day), "cycle day %d", day)
		}
	})

	t.Run("test_every_day_maps_to_some_subphase_without_gaps", func(t *testing.T) {
		for day := 1; day <= mapperCycle; day++ {
			assert.True(t, mapperSubphase(day).IsValid(), "cycle day %d", day)
		}
	})

	t.Run("test_subphase_maps_to_the_spec_fertility_level", func(t *testing.T) {
		assert.Equal(t, enums.FertilityLevelVeryHigh, enums.CycleSubphaseOvulationLikely.FertilityLevel())
		assert.Equal(t, enums.FertilityLevelHigh, enums.CycleSubphaseHighFertility.FertilityLevel())
		assert.Equal(t, enums.FertilityLevelMedium, enums.CycleSubphaseFertileRising.FertilityLevel())
		assert.Equal(t, enums.FertilityLevelMedium, enums.CycleSubphasePostOvulation.FertilityLevel())
		assert.Equal(t, enums.FertilityLevelLow, enums.CycleSubphaseMenstruation.FertilityLevel())
		assert.Equal(t, enums.FertilityLevelLow, enums.CycleSubphaseMidLuteal.FertilityLevel())
		assert.Equal(t, enums.FertilityLevelLow, enums.CycleSubphasePeriodExpected.FertilityLevel())
	})
}
