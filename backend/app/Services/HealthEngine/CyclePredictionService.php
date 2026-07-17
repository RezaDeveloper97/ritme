<?php

namespace App\Services\HealthEngine;

use App\Enums\EffectiveSource;
use Carbon\Carbon;

/**
 * Turns a Start anchor and effective values into the calendar prediction for the
 * cycle around a reference date (spec §2, §10, §13). Prediction always advances from
 * a period **Start**, never an End: next start = current start + effective cycle
 * length; ovulation = next start − luteal length; fertile window brackets ovulation.
 */
class CyclePredictionService
{
    /** MVP luteal length — the fixed offset from ovulation to the next period (§2). */
    private const LUTEAL_LENGTH = 14;

    private const FERTILE_DAYS_BEFORE_OVULATION = 5;

    private const FERTILE_DAYS_AFTER_OVULATION = 1;

    /**
     * Predict the cycle containing $referenceDate from the last real Start ($anchorStart).
     * The anchor is projected forward in whole cycles so the returned next-period start
     * is the first one at/after the reference date's cycle boundary.
     */
    public function predict(
        Carbon $anchorStart,
        int $cycleLength,
        int $periodDuration,
        Carbon $referenceDate,
        EffectiveSource $source,
    ): CyclePrediction {
        $cycleLength = max(1, $cycleLength);
        $periodDuration = max(1, $periodDuration);

        $anchor = $anchorStart->copy()->startOfDay();
        $reference = $referenceDate->copy()->startOfDay();

        $daysSinceAnchor = (int) $anchor->diffInDays($reference, false);
        $completedCycles = $daysSinceAnchor > 0 ? intdiv($daysSinceAnchor, $cycleLength) : 0;

        $currentCycleStart = $anchor->copy()->addDays($completedCycles * $cycleLength);
        $nextPeriodStart = $currentCycleStart->copy()->addDays($cycleLength);
        $nextPeriodEnd = $nextPeriodStart->copy()->addDays($periodDuration - 1);

        // Ovulation is anchored to the *next* period start (luteal phase is the stable
        // one), so the fertile window belongs to the current cycle (§2).
        $ovulation = $nextPeriodStart->copy()->subDays(self::LUTEAL_LENGTH);
        $fertileStart = $ovulation->copy()->subDays(self::FERTILE_DAYS_BEFORE_OVULATION);
        $fertileEnd = $ovulation->copy()->addDays(self::FERTILE_DAYS_AFTER_OVULATION);

        return new CyclePrediction(
            currentCycleStart: $currentCycleStart,
            nextPeriodStart: $nextPeriodStart,
            nextPeriodEnd: $nextPeriodEnd,
            estimatedOvulationDate: $ovulation,
            fertileWindowStart: $fertileStart,
            fertileWindowEnd: $fertileEnd,
            source: $source,
        );
    }
}
