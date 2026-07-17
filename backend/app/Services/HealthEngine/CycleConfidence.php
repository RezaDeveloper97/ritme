<?php

namespace App\Services\HealthEngine;

use App\Enums\ConfidenceLevel;

/**
 * A prediction's confidence level plus the machine-readable reasons behind it
 * (spec §18). Reasons are stable string codes the client localises; keeping them
 * separate from the level lets the UI explain *why* a prediction is (un)certain.
 */
final class CycleConfidence
{
    /**
     * @param  array<string>  $reasons
     */
    public function __construct(
        public readonly ConfidenceLevel $level,
        public readonly array $reasons,
    ) {}

    public function toApiArray(): array
    {
        return [
            'confidence' => $this->level->value,
            'confidence_reasons' => $this->reasons,
        ];
    }
}
