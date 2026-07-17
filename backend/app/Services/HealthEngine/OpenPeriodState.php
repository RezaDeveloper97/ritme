<?php

namespace App\Services\HealthEngine;

use App\Enums\DataQualityFlag;
use Carbon\Carbon;

/**
 * The state of an ongoing (Start logged, End not yet logged) period relative to today
 * (spec §4). It never invents an actual end: while the bleed is within the expected
 * length the day is shown as "day X of period"; past that the card asks the user to
 * log the end, and past the hard cap it stops claiming an active period altogether.
 */
final class OpenPeriodState
{
    public function __construct(
        public readonly bool $hasOpenPeriod,
        public readonly ?Carbon $startDate,
        /** today − start + 1, or null when no open period. */
        public readonly ?int $menstrualDay,
        /** Within the expected bleed length → show "day X of period". */
        public readonly bool $displayAsActiveMenstrual,
        /** Past the expected length → prompt the user to log the end. */
        public readonly bool $endOverdue,
        /** Past the hard cap → stop showing an active period, keep predicting. */
        public readonly bool $pastHardCap,
        /** Internal-only assumed end for prediction continuity; never shown/stored as actual. */
        public readonly ?Carbon $assumedEndDate,
        /** @var array<string> data-quality flags derived at read time (e.g. incomplete_end_missing). */
        public readonly array $dataQualityFlags,
    ) {}

    public static function none(): self
    {
        return new self(false, null, null, false, false, false, null, []);
    }

    public function hasFlag(DataQualityFlag $flag): bool
    {
        return in_array($flag->value, $this->dataQualityFlags, true);
    }
}
