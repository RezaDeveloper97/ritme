package resolver

import "github.com/ritme/backend-go/internal/enums"

// PhaseMapper is CyclePhaseMapper.php: pure cycle-day → phase / sub-phase mapping relative to the
// ovulation day O and the next-period day P = cycleLength + 1 (spec §16–17). Used by the legacy
// HealthDataEngine port (T-M2-14).
type PhaseMapper struct{}

// PhaseFor is the four-phase classification (CyclePhaseMapper::phaseFor).
func (PhaseMapper) PhaseFor(cycleDay, ovulationDay, bleedingLength int) enums.CyclePhase {
	switch {
	case cycleDay <= bleedingLength:
		return enums.CyclePhaseMenstruation
	case cycleDay < ovulationDay:
		return enums.CyclePhaseFollicular
	case cycleDay <= ovulationDay+1:
		return enums.CyclePhaseOvulation
	default:
		return enums.CyclePhaseLuteal
	}
}

// SubphaseFor is the twelve-state sub-phase (CyclePhaseMapper::subphaseFor). period_expected is
// never returned here.
func (m PhaseMapper) SubphaseFor(cycleDay, ovulationDay, cycleLength, bleedingLength int) enums.CycleSubphase {
	nextPeriodDay := cycleLength + 1
	switch {
	case cycleDay <= bleedingLength:
		return enums.CycleSubphaseMenstruation
	case cycleDay <= ovulationDay-6:
		return m.follicularHalf(cycleDay, bleedingLength, ovulationDay)
	case cycleDay <= ovulationDay-3:
		return enums.CycleSubphaseFertileRising
	case cycleDay <= ovulationDay-1:
		return enums.CycleSubphaseHighFertility
	case cycleDay <= ovulationDay:
		return enums.CycleSubphaseOvulationLikely
	case cycleDay <= ovulationDay+1:
		return enums.CycleSubphasePostOvulation
	case cycleDay <= ovulationDay+5:
		return enums.CycleSubphaseEarlyLuteal
	case cycleDay <= nextPeriodDay-6:
		return enums.CycleSubphaseMidLuteal
	case cycleDay <= nextPeriodDay-3:
		return enums.CycleSubphaseLateLuteal
	default:
		return enums.CycleSubphasePmsPossible
	}
}

// follicularHalf splits end-of-bleed → O-6 into early / mid halves (intdiv of the span).
func (PhaseMapper) follicularHalf(cycleDay, bleedingLength, ovulationDay int) enums.CycleSubphase {
	start := bleedingLength + 1
	end := ovulationDay - 6
	midpoint := start + max(0, end-start)/2
	if cycleDay <= midpoint {
		return enums.CycleSubphaseEarlyFollicular
	}
	return enums.CycleSubphaseMidFollicular
}
