<?php

namespace App\Services\HealthEngine;

use App\Enums\CycleVariability;
use App\Enums\EffectiveSource;
use App\Enums\RegularityStatus;

/**
 * Immutable snapshot of a user's cycle metrics across the spec's three layers
 * (profile / calculated / effective, §12) plus the regularity read-out the daily
 * card and predictions rely on. Produced by {@see CycleMetricsCalculator}.
 */
final class CycleMetrics
{
    /**
     * @param  array<int>  $recentValidCycleLengths  most recent valid lengths (oldest→newest), ≤3
     */
    public function __construct(
        public readonly ?int $profileCycleLength,
        public readonly ?int $profilePeriodDuration,
        public readonly ?int $calculatedCycleLength,
        public readonly ?int $calculatedPeriodDuration,
        public readonly int $effectiveCycleLength,
        public readonly int $effectivePeriodDuration,
        public readonly EffectiveSource $cycleLengthSource,
        public readonly EffectiveSource $periodDurationSource,
        public readonly int $validCyclesCount,
        public readonly int $validPeriodDurationsCount,
        public readonly RegularityStatus $regularityStatus,
        public readonly CycleVariability $variability,
        public readonly array $recentValidCycleLengths,
    ) {}

    /**
     * The spec §19 three-layer values block for the API payload. `based_on_cycles`
     * stays null until at least three valid cycles exist (calculated values undefined).
     */
    public function toApiArray(): array
    {
        return [
            'profile_values' => [
                'cycle_length' => $this->profileCycleLength,
                'period_duration' => $this->profilePeriodDuration,
            ],
            'calculated_values' => [
                'cycle_length' => $this->calculatedCycleLength,
                'period_duration' => $this->calculatedPeriodDuration,
                'based_on_cycles' => $this->validCyclesCount >= 3 ? $this->validCyclesCount : null,
            ],
            'effective_values' => [
                'cycle_length' => $this->effectiveCycleLength,
                'period_duration' => $this->effectivePeriodDuration,
                'source' => $this->cycleLengthSource->value,
                'period_duration_source' => $this->periodDurationSource->value,
            ],
        ];
    }
}
