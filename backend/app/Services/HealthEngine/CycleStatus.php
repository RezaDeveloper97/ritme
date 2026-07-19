<?php

namespace App\Services\HealthEngine;

use App\Enums\ConfidenceLevel;
use App\Enums\CycleSubphase;
use App\Enums\DataQualityLevel;
use App\Enums\FertilityLevel;
use App\Enums\MainPhase;
use App\Enums\PeriodEndSource;
use App\Enums\PeriodStartSource;
use App\Enums\ResolutionSource;
use Carbon\Carbon;

/**
 * The resolved v1.1 status of one target date (task.md §35): the authoritative
 * main phase / subphase, the day counters, the anchors with their sources, and
 * the deterministic quality/confidence read-outs. Produced by
 * {@see CycleStatusResolver}; predicted values here are never written back to
 * period logs (§2, §40).
 */
final class CycleStatus
{
    /**
     * @param  array<string>  $confidenceReasons
     * @param  array<string>  $warnings
     */
    public function __construct(
        public readonly Carbon $date,
        public readonly ?int $cycleDay,
        public readonly MainPhase $mainPhase,
        public readonly CycleSubphase $subphase,
        public readonly FertilityLevel $fertilityLevel,
        public readonly ?int $daysToOvulation,
        public readonly ?int $daysToPeriod,
        public readonly ?int $daysLate,
        public readonly ?Carbon $currentPeriodStart,
        public readonly PeriodStartSource $currentPeriodStartSource,
        public readonly ?Carbon $currentPeriodEnd,
        public readonly PeriodEndSource $currentPeriodEndSource,
        public readonly bool $currentPeriodEndIsConfirmed,
        public readonly ?Carbon $predictedNextPeriodStart,
        public readonly ?Carbon $estimatedOvulationDate,
        public readonly int $effectiveCycleLength,
        public readonly int $effectivePeriodLength,
        public readonly ?int $cycleVariability,
        public readonly ConfidenceLevel $confidence,
        public readonly array $confidenceReasons,
        public readonly DataQualityLevel $dataQuality,
        public readonly ResolutionSource $resolutionSource,
        public readonly bool $requiresUserInput,
        public readonly array $warnings,
    ) {}

    /** The exact task.md §35 output structure. */
    public function toApiArray(): array
    {
        return [
            'date' => $this->date->toDateString(),
            'cycle_day' => $this->cycleDay,

            'main_phase' => $this->mainPhase->value,
            'subphase' => $this->subphase->value,
            'fertility_level' => $this->fertilityLevel->value,

            'days_to_ovulation' => $this->daysToOvulation,
            'days_to_period' => $this->daysToPeriod,
            'days_late' => $this->daysLate,

            'anchors' => [
                'current_period_start' => $this->currentPeriodStart?->toDateString(),
                'current_period_start_source' => $this->currentPeriodStartSource->value,

                'current_period_end' => $this->currentPeriodEnd?->toDateString(),
                'current_period_end_source' => $this->currentPeriodEndSource->value,
                'current_period_end_is_confirmed' => $this->currentPeriodEndIsConfirmed,

                'predicted_next_period_start' => $this->predictedNextPeriodStart?->toDateString(),
                'estimated_ovulation_date' => $this->estimatedOvulationDate?->toDateString(),
            ],

            'metrics' => [
                'effective_cycle_length' => $this->effectiveCycleLength,
                'effective_period_length' => $this->effectivePeriodLength,
                'cycle_variability' => $this->cycleVariability,
            ],

            'confidence' => $this->confidence->value,
            'confidence_reasons' => array_values($this->confidenceReasons),
            'data_quality' => $this->dataQuality->value,

            'resolution_source' => $this->resolutionSource->value,
            'is_predicted' => $this->resolutionSource->isPredicted(),

            'requires_user_input' => $this->requiresUserInput,
            'warnings' => array_values($this->warnings),
        ];
    }
}
