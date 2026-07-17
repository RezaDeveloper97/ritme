<?php

namespace App\Services\HealthEngine;

use App\Enums\DataQualityFlag;
use Carbon\Carbon;

/**
 * Decides how an ongoing period (Start logged, no End) should be treated today
 * (spec §4). Two thresholds bound it: the expected bleed length, after which the app
 * asks the user to confirm the end, and a hard cap, after which it stops presenting
 * an active period. The engine never writes an actual end on the user's behalf.
 */
class OpenPeriodEvaluator
{
    /** After this many days an unclosed period is no longer shown as active bleeding. */
    private const HARD_CAP_DAYS = 12;

    /** The expected bleed length is never treated as shorter than this. */
    private const MIN_MAX_EXPECTED_DAYS = 8;

    /** Grace added to the profile's period duration to get the "still expected" bound. */
    private const MAX_EXPECTED_EXTRA_DAYS = 3;

    public function evaluate(
        ?Carbon $openPeriodStart,
        int $effectivePeriodDuration,
        ?int $profilePeriodDuration,
        Carbon $today,
    ): OpenPeriodState {
        if ($openPeriodStart === null) {
            return OpenPeriodState::none();
        }

        $start = $openPeriodStart->copy()->startOfDay();
        $reference = $today->copy()->startOfDay();

        // A start dated in the future isn't an "ongoing" bleed yet.
        if ($reference->lt($start)) {
            return OpenPeriodState::none();
        }

        $menstrualDay = (int) $start->diffInDays($reference) + 1;

        // max_expected_period_days = max(profile_period_duration + 3, 8) (§4).
        $baseDuration = $profilePeriodDuration ?? $effectivePeriodDuration;
        $maxExpected = max($baseDuration + self::MAX_EXPECTED_EXTRA_DAYS, self::MIN_MAX_EXPECTED_DAYS);

        // The hard cap always dominates: a long profile period duration can push
        // max_expected past it, but a period is never shown active beyond the cap (§4).
        $displayActive = $menstrualDay <= min($maxExpected, self::HARD_CAP_DAYS);
        $endOverdue = ! $displayActive;
        $pastHardCap = $menstrualDay > self::HARD_CAP_DAYS;

        // assumed_period_end_date keeps internal prediction going without fabricating
        // an actual end (§4) — start + effective duration − 1.
        $assumedEnd = $start->copy()->addDays(max(1, $effectivePeriodDuration) - 1);

        $flags = $endOverdue ? [DataQualityFlag::INCOMPLETE_END_MISSING->value] : [];

        return new OpenPeriodState(
            hasOpenPeriod: true,
            startDate: $start,
            menstrualDay: $menstrualDay,
            displayAsActiveMenstrual: $displayActive,
            endOverdue: $endOverdue,
            pastHardCap: $pastHardCap,
            assumedEndDate: $assumedEnd,
            dataQualityFlags: $flags,
        );
    }
}
