<?php

namespace App\Services\HealthEngine;

use App\Enums\DataStatus;
use App\Enums\FertilityLevel;

/**
 * A render-ready status card for one day (spec §19). The backend owns the title,
 * subtitle, fertility read-out and calls to action so the clients never re-implement
 * the Cycle Engine's rules — they just render this.
 */
final class DailyCard
{
    /**
     * @param  array<string>  $badges
     * @param  array{type: string, label: string}|null  $primaryAction
     * @param  array<array{type: string, label: string}>  $secondaryActions
     */
    public function __construct(
        public readonly string $title,
        public readonly string $subtitle,
        public readonly DataStatus $dataStatus,
        public readonly FertilityLevel $fertilityLevel,
        public readonly string $fertilityLabel,
        public readonly array $badges,
        public readonly ?array $primaryAction,
        public readonly array $secondaryActions,
    ) {}

    public function toApiArray(): array
    {
        return [
            'title' => $this->title,
            'subtitle' => $this->subtitle,
            'data_status' => $this->dataStatus->value,
            'fertility_level' => $this->fertilityLevel->value,
            'fertility_label' => $this->fertilityLabel,
            'badges' => $this->badges,
            'primary_action' => $this->primaryAction,
            'secondary_actions' => $this->secondaryActions,
        ];
    }
}
