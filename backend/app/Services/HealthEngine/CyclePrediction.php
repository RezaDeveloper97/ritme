<?php

namespace App\Services\HealthEngine;

use App\Enums\EffectiveSource;
use Carbon\Carbon;

/**
 * Structured, Start-anchored prediction for the cycle containing a reference date
 * (spec §2, §10, §13): the upcoming period, the estimated ovulation and the fertile
 * window. These are calendar estimates regenerated from effective values — never
 * stored as actual period logs.
 */
final class CyclePrediction
{
    public function __construct(
        /** Predicted start of the cycle the reference date falls in. */
        public readonly Carbon $currentCycleStart,
        public readonly Carbon $nextPeriodStart,
        public readonly Carbon $nextPeriodEnd,
        public readonly Carbon $estimatedOvulationDate,
        public readonly Carbon $fertileWindowStart,
        public readonly Carbon $fertileWindowEnd,
        public readonly EffectiveSource $source,
    ) {}

    /**
     * The spec §13/§19 predictions block (dates only; confidence is attached by the
     * payload builder from the confidence model).
     */
    public function toApiArray(): array
    {
        return [
            'next_period_start' => $this->nextPeriodStart->toDateString(),
            'next_period_end' => $this->nextPeriodEnd->toDateString(),
            'estimated_ovulation_date' => $this->estimatedOvulationDate->toDateString(),
            'fertile_window_start' => $this->fertileWindowStart->toDateString(),
            'fertile_window_end' => $this->fertileWindowEnd->toDateString(),
            'source' => $this->source->value,
        ];
    }
}
