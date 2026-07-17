<?php

namespace App\Services\HealthEngine;

use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;

/**
 * Pure mapping from a cycle day to its phase and sub-phase, positioned relative to
 * the estimated ovulation day O and the next-period day P (spec §16–17). No I/O and
 * no dependencies, so the calendar's phase logic can be unit-tested directly.
 */
class CyclePhaseMapper
{
    /**
     * Four-phase classification derived from the ovulation day (§16), so a non-28-day
     * cycle gets a correct split instead of the old fixed day-12 / 13–15 boundaries.
     */
    public function phaseFor(int $cycleDay, int $ovulationDay, int $bleedingLength): CyclePhase
    {
        return match (true) {
            $cycleDay <= $bleedingLength => CyclePhase::MENSTRUATION,
            $cycleDay < $ovulationDay => CyclePhase::FOLLICULAR,
            $cycleDay <= $ovulationDay + 1 => CyclePhase::OVULATION,
            default => CyclePhase::LUTEAL,
        };
    }

    /**
     * Twelve-state sub-phase (§17). period_expected is intentionally never returned
     * here — that is a data-status decision the daily card makes when a predicted
     * period is due but no Start has been logged.
     */
    public function subphaseFor(int $cycleDay, int $ovulationDay, int $cycleLength, int $bleedingLength): CycleSubphase
    {
        $nextPeriodDay = $cycleLength + 1; // "P": day 1 of the next (predicted) cycle

        return match (true) {
            $cycleDay <= $bleedingLength => CycleSubphase::MENSTRUATION,
            $cycleDay <= $ovulationDay - 6 => $this->follicularHalf($cycleDay, $bleedingLength, $ovulationDay),
            $cycleDay <= $ovulationDay - 3 => CycleSubphase::FERTILE_RISING,   // O-5..O-3
            $cycleDay <= $ovulationDay - 1 => CycleSubphase::HIGH_FERTILITY,   // O-2..O-1
            $cycleDay <= $ovulationDay => CycleSubphase::OVULATION_LIKELY,     // O
            $cycleDay <= $ovulationDay + 1 => CycleSubphase::POST_OVULATION,   // O+1
            $cycleDay <= $ovulationDay + 5 => CycleSubphase::EARLY_LUTEAL,     // O+2..O+5
            $cycleDay <= $nextPeriodDay - 6 => CycleSubphase::MID_LUTEAL,      // O+6..P-6
            $cycleDay <= $nextPeriodDay - 3 => CycleSubphase::LATE_LUTEAL,     // P-5..P-3
            default => CycleSubphase::PMS_POSSIBLE,                            // P-2..P-1
        };
    }

    /**
     * Split the pre-fertile follicular stretch (end of bleeding → O-6) into an early
     * and a mid half so the calendar shows the energy ramp, not one flat block.
     */
    private function follicularHalf(int $cycleDay, int $bleedingLength, int $ovulationDay): CycleSubphase
    {
        $start = $bleedingLength + 1;
        $end = $ovulationDay - 6;
        $midpoint = $start + intdiv(max(0, $end - $start), 2);

        return $cycleDay <= $midpoint ? CycleSubphase::EARLY_FOLLICULAR : CycleSubphase::MID_FOLLICULAR;
    }
}
