<?php

namespace App\Services\HealthEngine;

use App\Enums\ConfidenceLevel;
use App\Enums\CycleVariability;

/**
 * Grades how much to trust a cycle prediction (spec §18). Confidence rises with the
 * number of valid logged cycles and their regularity, and is degraded when data is
 * incomplete (e.g. a period end missing well past its expected length). The reasons
 * are stable codes so the client can explain the grade.
 */
class CycleConfidenceCalculator
{
    public const REASON_THREE_VALID_CYCLES = 'based_on_three_valid_cycles';

    public const REASON_LOW_VARIATION = 'low_cycle_variation';

    public const REASON_CYCLE_VARIATION = 'cycle_variation';

    public const REASON_NOT_ENOUGH_CYCLES = 'not_enough_logged_cycles';

    public const REASON_PROFILE_ONLY = 'based_on_profile_data';

    public const REASON_END_MISSING_LONG = 'incomplete_end_missing';

    /**
     * Confidence for the next-period / ovulation prediction. $endMissingLong marks an
     * ongoing period whose end is missing past the hard cap, which lowers trust (§18).
     */
    public function forPrediction(CycleMetrics $metrics, bool $endMissingLong = false): CycleConfidence
    {
        [$level, $reasons] = match (true) {
            $metrics->validCyclesCount >= 3 && $metrics->variability === CycleVariability::REGULAR => [
                ConfidenceLevel::HIGH,
                [self::REASON_THREE_VALID_CYCLES, self::REASON_LOW_VARIATION],
            ],
            $metrics->validCyclesCount >= 3 => [
                ConfidenceLevel::MEDIUM,
                [self::REASON_THREE_VALID_CYCLES, self::REASON_CYCLE_VARIATION],
            ],
            $metrics->validCyclesCount >= 1 => [
                ConfidenceLevel::MEDIUM,
                [self::REASON_NOT_ENOUGH_CYCLES],
            ],
            default => [
                ConfidenceLevel::LOW,
                [self::REASON_PROFILE_ONLY, self::REASON_NOT_ENOUGH_CYCLES],
            ],
        };

        if ($endMissingLong) {
            $level = $this->downgrade($level);
            $reasons[] = self::REASON_END_MISSING_LONG;
        }

        return new CycleConfidence($level, array_values(array_unique($reasons)));
    }

    private function downgrade(ConfidenceLevel $level): ConfidenceLevel
    {
        return match ($level) {
            ConfidenceLevel::HIGH => ConfidenceLevel::MEDIUM,
            ConfidenceLevel::MEDIUM, ConfidenceLevel::LOW => ConfidenceLevel::LOW,
        };
    }
}
