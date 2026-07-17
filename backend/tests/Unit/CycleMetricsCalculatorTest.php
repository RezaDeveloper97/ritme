<?php

namespace Tests\Unit;

use App\Enums\EffectiveSource;
use App\Enums\RegularityStatus;
use App\Models\UserProfile;
use App\Services\HealthEngine\CycleMetricsCalculator;
use Illuminate\Support\Collection;
use PHPUnit\Framework\TestCase;

/**
 * Pure-logic tests for the three-layer effective-values engine (spec §6–8, §12).
 * Period records are lightweight objects so the calculator can be exercised without
 * booting Eloquent or a database — it only reads start/end/confirmed.
 */
class CycleMetricsCalculatorTest extends TestCase
{
    private CycleMetricsCalculator $calculator;

    protected function setUp(): void
    {
        parent::setUp();
        $this->calculator = new CycleMetricsCalculator;
    }

    /** A confirmed logged period, unless $confirmed is overridden. */
    private function period(string $start, ?string $end = null, bool $confirmed = true): object
    {
        return (object) [
            'period_start_date' => $start,
            'period_end_date' => $end,
            'is_confirmed' => $confirmed,
        ];
    }

    /** @param  array<object>  $periods */
    private function metricsFor(array $periods, ?UserProfile $profile): \App\Services\HealthEngine\CycleMetrics
    {
        return $this->calculator->calculate(new Collection($periods), $profile);
    }

    private function profile(?int $cycle, ?int $duration): UserProfile
    {
        return new UserProfile(['cycle_duration' => $cycle, 'period_duration' => $duration]);
    }

    public function test_falls_back_to_system_default_when_no_data(): void
    {
        $metrics = $this->metricsFor([], null);

        $this->assertSame(28, $metrics->effectiveCycleLength);
        $this->assertSame(5, $metrics->effectivePeriodDuration);
        $this->assertSame(EffectiveSource::DEFAULT, $metrics->cycleLengthSource);
        $this->assertSame(EffectiveSource::DEFAULT, $metrics->periodDurationSource);
        $this->assertNull($metrics->calculatedCycleLength);
        $this->assertSame(RegularityStatus::NOT_ENOUGH_DATA, $metrics->regularityStatus);
    }

    public function test_uses_profile_not_average_with_fewer_than_three_cycles(): void
    {
        // Two valid 30-day cycles, but the profile says 28. The spec forbids
        // averaging/weighting below 3 cycles — effective must be the profile's 28.
        $metrics = $this->metricsFor([
            $this->period('2026-01-01'),
            $this->period('2026-01-31'),
            $this->period('2026-03-02'),
        ], $this->profile(28, 5));

        $this->assertSame(28, $metrics->effectiveCycleLength);
        $this->assertSame(EffectiveSource::PROFILE, $metrics->cycleLengthSource);
        $this->assertNull($metrics->calculatedCycleLength);
        $this->assertSame(2, $metrics->validCyclesCount);
        $this->assertSame(RegularityStatus::NOT_ENOUGH_DATA, $metrics->regularityStatus);
    }

    public function test_uses_median_of_last_three_valid_cycles(): void
    {
        // Cycle lengths 28, 30, 35 → median 35? no: sorted [28,30,35] → median 30.
        $metrics = $this->metricsFor([
            $this->period('2026-01-01'),
            $this->period('2026-01-29'), // +28
            $this->period('2026-02-28'), // +30
            $this->period('2026-04-04'), // +35
        ], $this->profile(28, 5));

        $this->assertSame(30, $metrics->calculatedCycleLength);
        $this->assertSame(30, $metrics->effectiveCycleLength);
        $this->assertSame(EffectiveSource::RECENT_VALID_CYCLES, $metrics->cycleLengthSource);
        $this->assertSame(3, $metrics->validCyclesCount);
    }

    public function test_outlier_cycles_are_excluded_from_the_median(): void
    {
        // A 50-day gap is outside 21–45 and must not enter the median.
        $metrics = $this->metricsFor([
            $this->period('2026-01-01'),
            $this->period('2026-02-20'), // +50 (outlier, dropped)
            $this->period('2026-03-20'), // +28
            $this->period('2026-04-19'), // +30
            $this->period('2026-05-21'), // +32
        ], $this->profile(28, 5));

        $this->assertSame(3, $metrics->validCyclesCount);
        $this->assertSame(30, $metrics->calculatedCycleLength); // median of [28,30,32]
    }

    public function test_period_duration_is_calculated_independently_of_cycle_length(): void
    {
        // Only two cycles (→ cycle falls back to profile) but three valid durations
        // (→ duration uses the learned median). Proves the two are independent (§7).
        $metrics = $this->metricsFor([
            $this->period('2026-01-01', '2026-01-04'), // duration 4
            $this->period('2026-01-31', '2026-02-04'), // duration 5
            $this->period('2026-03-02', '2026-03-07'), // duration 6
        ], $this->profile(28, 9));

        $this->assertSame(2, $metrics->validCyclesCount);
        $this->assertNull($metrics->calculatedCycleLength);
        $this->assertSame(EffectiveSource::PROFILE, $metrics->cycleLengthSource);

        $this->assertSame(3, $metrics->validPeriodDurationsCount);
        $this->assertSame(5, $metrics->calculatedPeriodDuration); // median of [4,5,6]
        $this->assertSame(5, $metrics->effectivePeriodDuration);
        $this->assertSame(EffectiveSource::RECENT_VALID_CYCLES, $metrics->periodDurationSource);
    }

    public function test_open_and_overlong_periods_do_not_feed_the_duration_median(): void
    {
        $metrics = $this->metricsFor([
            $this->period('2026-01-01', '2026-01-05'),  // duration 5 (valid)
            $this->period('2026-01-31', null),          // open → skipped
            $this->period('2026-03-02', '2026-03-20'),  // duration 19 → out of 2–10, dropped
        ], $this->profile(28, 5));

        $this->assertSame(1, $metrics->validPeriodDurationsCount);
        $this->assertNull($metrics->calculatedPeriodDuration); // fewer than 3 valid
        $this->assertSame(5, $metrics->effectivePeriodDuration); // profile fallback
        $this->assertSame(EffectiveSource::PROFILE, $metrics->periodDurationSource);
    }

    public function test_unconfirmed_periods_are_ignored(): void
    {
        $metrics = $this->metricsFor([
            $this->period('2026-01-01', confirmed: false),
            $this->period('2026-01-29', confirmed: false),
            $this->period('2026-02-28', confirmed: false),
            $this->period('2026-04-04', confirmed: false),
        ], $this->profile(28, 5));

        $this->assertSame(0, $metrics->validCyclesCount);
        $this->assertNull($metrics->calculatedCycleLength);
        $this->assertSame(EffectiveSource::PROFILE, $metrics->cycleLengthSource);
    }

    public function test_regularity_reads_the_range_of_the_last_three_cycles(): void
    {
        $regular = $this->metricsFor([
            $this->period('2026-01-01'),
            $this->period('2026-01-29'), // +28
            $this->period('2026-02-27'), // +29
            $this->period('2026-03-29'), // +30 → range 2
        ], $this->profile(28, 5));
        $this->assertSame(RegularityStatus::RELATIVELY_REGULAR, $regular->regularityStatus);

        $irregular = $this->metricsFor([
            $this->period('2026-01-01'),
            $this->period('2026-01-28'), // +27
            $this->period('2026-03-03'), // +34
            $this->period('2026-04-14'), // +42 → range 15
        ], $this->profile(28, 5));
        $this->assertSame(RegularityStatus::IRREGULAR_POSSIBLE, $irregular->regularityStatus);
    }
}
