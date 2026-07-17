<?php

namespace App\Services\HealthEngine;

use App\Enums\CycleVariability;
use App\Enums\EffectiveSource;
use App\Enums\RegularityStatus;
use App\Models\UserProfile;
use Carbon\Carbon;
use Illuminate\Support\Collection;

/**
 * Derives the spec's three-layer cycle metrics (§6–8, §12) from a user's logged
 * period history and profile: cycle length and period duration are computed
 * **independently**, each from its own set of valid records.
 *
 * MVP rules (§7): with three or more valid cycles the effective value is the median
 * of the last three; with fewer, it falls back to the profile value (no averaging,
 * no weighting) and finally a system default. A "valid" cycle is 21–45 days between
 * two user-confirmed starts; a valid period duration is 2–10 days (§8).
 */
class CycleMetricsCalculator
{
    private const VALID_CYCLE_MIN = 21;

    private const VALID_CYCLE_MAX = 45;

    private const VALID_DURATION_MIN = 2;

    private const VALID_DURATION_MAX = 10;

    /** MVP window: the effective value is the median of the last three valid records. */
    private const RECENT_WINDOW = 3;

    /** Sample size for the std-dev variability that feeds prediction uncertainty. */
    private const VARIABILITY_WINDOW = 6;

    private const DEFAULT_CYCLE_LENGTH = 28;

    private const DEFAULT_PERIOD_DURATION = 5;

    public function calculate(Collection $histories, ?UserProfile $profile): CycleMetrics
    {
        // Only user-confirmed starts count as valid cycles (§8); auto-detected rows
        // stay in history but never feed prediction. Sorted oldest→newest.
        $confirmed = $histories
            ->filter(fn ($history) => (bool) $history->is_confirmed)
            ->sortBy(fn ($history) => Carbon::parse($history->period_start_date)->timestamp)
            ->values();

        $validCycleLengths = $this->validCycleLengths($confirmed);
        $validDurations = $this->validPeriodDurations($confirmed);

        $recentCycles = array_slice($validCycleLengths, -self::RECENT_WINDOW);
        $recentDurations = array_slice($validDurations, -self::RECENT_WINDOW);

        $calculatedCycle = count($recentCycles) >= self::RECENT_WINDOW ? $this->median($recentCycles) : null;
        $calculatedDuration = count($recentDurations) >= self::RECENT_WINDOW ? $this->median($recentDurations) : null;

        // `?:` also rejects a stored 0, which would otherwise divide-by-zero downstream.
        $profileCycle = ($profile && $profile->cycle_duration) ? (int) $profile->cycle_duration : null;
        $profileDuration = ($profile && $profile->period_duration) ? (int) $profile->period_duration : null;

        [$effectiveCycle, $cycleSource] = $this->resolveEffective($calculatedCycle, $profileCycle, self::DEFAULT_CYCLE_LENGTH);
        [$effectiveDuration, $durationSource] = $this->resolveEffective($calculatedDuration, $profileDuration, self::DEFAULT_PERIOD_DURATION);

        return new CycleMetrics(
            profileCycleLength: $profileCycle,
            profilePeriodDuration: $profileDuration,
            calculatedCycleLength: $calculatedCycle,
            calculatedPeriodDuration: $calculatedDuration,
            effectiveCycleLength: $effectiveCycle,
            effectivePeriodDuration: $effectiveDuration,
            cycleLengthSource: $cycleSource,
            periodDurationSource: $durationSource,
            validCyclesCount: count($validCycleLengths),
            validPeriodDurationsCount: count($validDurations),
            regularityStatus: RegularityStatus::fromCycleLengths($recentCycles),
            variability: $this->variability(array_slice($validCycleLengths, -self::VARIABILITY_WINDOW)),
            recentValidCycleLengths: array_values($recentCycles),
        );
    }

    /**
     * Cycle lengths (start→next start) between consecutive confirmed periods that
     * fall in the valid 21–45 day range, oldest→newest. Deriving from the start
     * dates — rather than the stored `cycle_length` — keeps this a pure function of
     * the logged timeline and matches the spec definition (§6).
     *
     * @return array<int>
     */
    private function validCycleLengths(Collection $confirmed): array
    {
        $starts = $confirmed
            ->map(fn ($history) => Carbon::parse($history->period_start_date)->startOfDay())
            ->values();

        $lengths = [];
        for ($i = 1; $i < $starts->count(); $i++) {
            $length = (int) $starts[$i - 1]->diffInDays($starts[$i]);
            if ($length >= self::VALID_CYCLE_MIN && $length <= self::VALID_CYCLE_MAX) {
                $lengths[] = $length;
            }
        }

        return $lengths;
    }

    /**
     * Period durations (end−start+1) for confirmed, closed periods that fall in the
     * valid 2–10 day range, oldest→newest. Open periods (no end) are skipped — an
     * incomplete cycle never feeds the duration median (§8).
     *
     * @return array<int>
     */
    private function validPeriodDurations(Collection $confirmed): array
    {
        $durations = [];
        foreach ($confirmed as $history) {
            if ($history->period_end_date === null) {
                continue;
            }

            $start = Carbon::parse($history->period_start_date)->startOfDay();
            $end = Carbon::parse($history->period_end_date)->startOfDay();
            if ($end->lt($start)) {
                continue;
            }

            $duration = ((int) $start->diffInDays($end)) + 1;
            if ($duration >= self::VALID_DURATION_MIN && $duration <= self::VALID_DURATION_MAX) {
                $durations[] = $duration;
            }
        }

        return $durations;
    }

    /**
     * Pick the effective value and record which layer it came from: the learned
     * median when available, else the profile baseline, else the system default (§7).
     *
     * @return array{0: int, 1: EffectiveSource}
     */
    private function resolveEffective(?int $calculated, ?int $profile, int $default): array
    {
        if ($calculated !== null) {
            return [$calculated, EffectiveSource::RECENT_VALID_CYCLES];
        }

        if ($profile !== null) {
            return [max($profile, 1), EffectiveSource::PROFILE];
        }

        return [$default, EffectiveSource::DEFAULT];
    }

    /**
     * @param  array<int>  $values
     */
    private function median(array $values): int
    {
        sort($values);
        $count = count($values);
        $middle = (int) floor($count / 2);

        return $count % 2 === 0
            ? (int) round(($values[$middle - 1] + $values[$middle]) / 2)
            : (int) $values[$middle];
    }

    /**
     * @param  array<int>  $lengths
     */
    private function variability(array $lengths): CycleVariability
    {
        if (count($lengths) < 2) {
            return CycleVariability::REGULAR;
        }

        $mean = array_sum($lengths) / count($lengths);
        $squaredDiffs = array_map(fn ($x) => ($x - $mean) ** 2, $lengths);
        $stdDev = sqrt(array_sum($squaredDiffs) / count($lengths));

        return CycleVariability::fromStdDev($stdDev);
    }
}
