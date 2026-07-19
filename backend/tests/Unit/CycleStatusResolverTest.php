<?php

namespace Tests\Unit;

use App\Enums\ConfidenceLevel;
use App\Enums\CycleSubphase;
use App\Enums\CycleWarning;
use App\Enums\DataQualityLevel;
use App\Enums\FertilityLevel;
use App\Enums\MainPhase;
use App\Enums\PeriodStartSource;
use App\Enums\ResolutionSource;
use App\Models\UserProfile;
use App\Services\HealthEngine\CycleMetricsCalculator;
use App\Services\HealthEngine\CycleStatus;
use App\Services\HealthEngine\CycleStatusResolver;
use Carbon\Carbon;
use Illuminate\Support\Collection;
use PHPUnit\Framework\TestCase;

/**
 * Pure-logic tests for the v1.1 engine (task.md), covering the unit scenarios the
 * spec's final section demands: the deterministic 21-day walk, a period without an
 * end at every cap tier, period_expected at days_late 7/8/15, the empty fertile
 * window, onboarding-only and default-only resolves, and the §22.5 cancel rule.
 */
class CycleStatusResolverTest extends TestCase
{
    private CycleStatusResolver $resolver;

    private CycleMetricsCalculator $metrics;

    protected function setUp(): void
    {
        parent::setUp();
        $this->resolver = new CycleStatusResolver;
        $this->metrics = new CycleMetricsCalculator;
    }

    private function period(string $start, ?string $end = null, bool $confirmed = true, bool $estimated = false): object
    {
        return (object) [
            'period_start_date' => $start,
            'period_end_date' => $end,
            'is_confirmed' => $confirmed,
            'is_estimated' => $estimated,
        ];
    }

    /** @param  array<object>  $periods */
    private function resolve(array $periods, ?UserProfile $profile, string $target, ?string $today = null): CycleStatus
    {
        $histories = new Collection($periods);

        return $this->resolver->resolve(
            $histories,
            $profile,
            Carbon::parse($target),
            Carbon::parse($today ?? $target),
            $this->metrics->calculate($histories, $profile),
        );
    }

    private function profile(?int $cycle, ?int $duration, ?string $lmp = null): UserProfile
    {
        // Raw attributes: the date mutator would need a DB connection to
        // resolve the model's date format, which pure unit tests don't have.
        $profile = new UserProfile;
        $profile->setRawAttributes([
            'cycle_duration' => $cycle,
            'period_duration' => $duration,
            'last_period_start' => $lmp,
        ]);

        return $profile;
    }

    /**
     * Three 21-day cycles of history, each with a 5-day logged bleed, and the
     * current period started 2026-01-11 with its end logged on 2026-01-15.
     *
     * @return array<object>
     */
    private function history21(): array
    {
        return [
            $this->period('2025-11-09', '2025-11-13'),
            $this->period('2025-11-30', '2025-12-04'),
            $this->period('2025-12-21', '2025-12-25'),
            $this->period('2026-01-11', '2026-01-15'),
        ];
    }

    /** Day N of the current 21-day cycle (day 1 = 2026-01-11). */
    private function day21(int $day): string
    {
        return Carbon::parse('2026-01-11')->addDays($day - 1)->toDateString();
    }

    public function test_21_day_cycle_walk_matches_the_spec_example(): void
    {
        // task.md §36: O = day 8, display fertile = day 6–8, follicular removed.
        $expected = [
            1 => CycleSubphase::MENSTRUAL, 2 => CycleSubphase::MENSTRUAL,
            3 => CycleSubphase::MENSTRUAL, 4 => CycleSubphase::MENSTRUAL,
            5 => CycleSubphase::MENSTRUAL,
            6 => CycleSubphase::HIGH_FERTILITY, 7 => CycleSubphase::HIGH_FERTILITY,
            8 => CycleSubphase::OVULATION_LIKELY,
            9 => CycleSubphase::POST_OVULATION,
            10 => CycleSubphase::EARLY_LUTEAL, 11 => CycleSubphase::EARLY_LUTEAL,
            12 => CycleSubphase::EARLY_LUTEAL,
            13 => CycleSubphase::MID_LUTEAL, 14 => CycleSubphase::MID_LUTEAL,
            15 => CycleSubphase::MID_LUTEAL,
            16 => CycleSubphase::LATE_LUTEAL, 17 => CycleSubphase::LATE_LUTEAL,
            18 => CycleSubphase::LATE_LUTEAL,
            19 => CycleSubphase::PMS_POSSIBLE, 20 => CycleSubphase::PMS_POSSIBLE,
            21 => CycleSubphase::PMS_POSSIBLE,
        ];

        foreach ($expected as $day => $subphase) {
            $status = $this->resolve($this->history21(), $this->profile(28, 5), $this->day21($day));
            $this->assertSame($subphase, $status->subphase, "day {$day}");
            $this->assertSame($day, $status->cycleDay, "cycle_day on day {$day}");
        }

        // Day 22 = predicted next period start → period_expected with days_late 0.
        $status = $this->resolve($this->history21(), $this->profile(28, 5), $this->day21(22));
        $this->assertSame(MainPhase::PERIOD_EXPECTED, $status->mainPhase);
        $this->assertSame(0, $status->daysLate);
        $this->assertSame(0, $status->daysToPeriod);
    }

    public function test_menstrual_day_with_logged_end_is_user_logged_and_high_confidence(): void
    {
        $status = $this->resolve($this->history21(), $this->profile(28, 5), $this->day21(3));

        $this->assertSame(MainPhase::MENSTRUAL, $status->mainPhase);
        $this->assertSame(ResolutionSource::USER_LOGGED, $status->resolutionSource);
        $this->assertFalse($status->resolutionSource->isPredicted());
        $this->assertTrue($status->currentPeriodEndIsConfirmed);
        $this->assertSame(DataQualityLevel::GOOD, $status->dataQuality);
        $this->assertSame(ConfidenceLevel::HIGH, $status->confidence);
        $this->assertSame(FertilityLevel::LOW, $status->fertilityLevel);
    }

    public function test_luteal_day_resolves_via_prediction_even_with_full_logs(): void
    {
        // task.md §35 note: the phase leans on predicted anchors → prediction.
        $status = $this->resolve($this->history21(), $this->profile(28, 5), $this->day21(13));

        $this->assertSame(MainPhase::LUTEAL, $status->mainPhase);
        $this->assertSame(ResolutionSource::PREDICTION, $status->resolutionSource);
        $this->assertTrue($status->resolutionSource->isPredicted());
        $this->assertSame(ConfidenceLevel::HIGH, $status->confidence);
        $this->assertContains(CycleWarning::PREDICTION_BASED_OUTPUT->value, $status->warnings);
    }

    /**
     * An ongoing period without an end (effective length 5 → soft cap 5,
     * warning cap 8, hard cap 12), checked at every tier of §21.
     *
     * @return array<object>
     */
    private function openPeriodHistory(): array
    {
        return [
            $this->period('2025-11-09', '2025-11-13'),
            $this->period('2025-11-30', '2025-12-04'),
            $this->period('2025-12-21', '2025-12-25'),
            $this->period('2026-01-11', null),
        ];
    }

    public function test_open_period_within_soft_cap_is_menstrual_with_assumed_end(): void
    {
        $status = $this->resolve($this->openPeriodHistory(), $this->profile(28, 5), $this->day21(4));

        $this->assertSame(MainPhase::MENSTRUAL, $status->mainPhase);
        $this->assertSame(CycleSubphase::MENSTRUAL, $status->subphase);
        $this->assertSame(ResolutionSource::USER_LOGGED_WITH_ASSUMED_END, $status->resolutionSource);
        $this->assertFalse($status->currentPeriodEndIsConfirmed);
        $this->assertSame('assumed_from_effective_duration', $status->currentPeriodEndSource->value);
        $this->assertNotContains(CycleWarning::PERIOD_END_MISSING->value, $status->warnings);
    }

    public function test_open_period_within_warning_cap_is_menstrual_possible(): void
    {
        // Day 7: past soft cap (5) but inside warning cap (8) → §21.3.
        $status = $this->resolve($this->openPeriodHistory(), $this->profile(28, 5), $this->day21(7));

        $this->assertSame(MainPhase::MENSTRUAL, $status->mainPhase);
        $this->assertSame(CycleSubphase::MENSTRUAL_POSSIBLE, $status->subphase);
        $this->assertContains(CycleWarning::PERIOD_END_MISSING->value, $status->warnings);
        $this->assertNotContains(CycleWarning::PERIOD_END_MISSING_WARNING_CAP_EXCEEDED->value, $status->warnings);
        $this->assertFalse($status->requiresUserInput);
    }

    public function test_open_period_past_warning_cap_degrades_quality_and_asks_for_input(): void
    {
        // Day 10: past warning cap (8), inside hard cap (12) → §21.4.
        $status = $this->resolve($this->openPeriodHistory(), $this->profile(28, 5), $this->day21(10));

        $this->assertSame(MainPhase::MENSTRUAL, $status->mainPhase);
        $this->assertSame(CycleSubphase::MENSTRUAL_POSSIBLE, $status->subphase);
        $this->assertContains(CycleWarning::PERIOD_END_MISSING_WARNING_CAP_EXCEEDED->value, $status->warnings);
        $this->assertSame(DataQualityLevel::POOR, $status->dataQuality);
        $this->assertSame(ConfidenceLevel::LOW, $status->confidence);
        $this->assertTrue($status->requiresUserInput);
    }

    public function test_open_period_past_hard_cap_resolves_the_cycle_onward(): void
    {
        // Day 13 (> hard cap 12): active menstrual is no longer assumed (§21.5)
        // but the engine keeps resolving — with O on day 8 the day lands in luteal.
        $status = $this->resolve($this->openPeriodHistory(), $this->profile(28, 5), $this->day21(13));

        $this->assertNotSame(MainPhase::MENSTRUAL, $status->mainPhase);
        $this->assertNotSame(MainPhase::UNKNOWN, $status->mainPhase);
        $this->assertSame(MainPhase::LUTEAL, $status->mainPhase);
        $this->assertContains(CycleWarning::PERIOD_END_MISSING_HARD_CAP_EXCEEDED->value, $status->warnings);
        $this->assertContains(CycleStatusResolver::REASON_MISSING_END_BEYOND_HARD_CAP, $status->confidenceReasons);
        $this->assertSame(DataQualityLevel::POOR, $status->dataQuality);
        $this->assertTrue($status->requiresUserInput);
    }

    public function test_period_expected_at_seven_days_late_is_calm(): void
    {
        $status = $this->resolve($this->history21(), $this->profile(28, 5), $this->day21(29));

        $this->assertSame(MainPhase::PERIOD_EXPECTED, $status->mainPhase);
        $this->assertSame(CycleSubphase::PERIOD_EXPECTED, $status->subphase);
        $this->assertSame(7, $status->daysLate);
        $this->assertSame(ResolutionSource::PREDICTION, $status->resolutionSource);
        $this->assertNotContains(CycleWarning::PREDICTED_PERIOD_OVERDUE->value, $status->warnings);
        $this->assertSame(FertilityLevel::UNKNOWN, $status->fertilityLevel);
    }

    public function test_period_expected_at_eight_days_late_warns_and_drops_confidence(): void
    {
        $status = $this->resolve($this->history21(), $this->profile(28, 5), $this->day21(30));

        $this->assertSame(MainPhase::PERIOD_EXPECTED, $status->mainPhase);
        $this->assertSame(8, $status->daysLate);
        $this->assertContains(CycleWarning::PREDICTED_PERIOD_OVERDUE->value, $status->warnings);
        $this->assertSame(ConfidenceLevel::LOW, $status->confidence);
        $this->assertContains(CycleStatusResolver::REASON_PERIOD_OVERDUE, $status->confidenceReasons);
    }

    public function test_period_expected_past_fourteen_days_late_goes_unknown_but_keeps_anchors(): void
    {
        $status = $this->resolve($this->history21(), $this->profile(28, 5), $this->day21(37));

        $this->assertSame(MainPhase::UNKNOWN, $status->mainPhase);
        $this->assertSame(15, $status->daysLate);
        $this->assertSame(0, $status->daysToPeriod);
        $this->assertNull($status->daysToOvulation); // §34
        $this->assertSame(37, $status->cycleDay); // start is still valid (§22.4)
        $this->assertNotNull($status->predictedNextPeriodStart);
        $this->assertNotNull($status->estimatedOvulationDate);
        $this->assertContains(CycleWarning::CYCLE_UNRESOLVED_AFTER_EXPECTED_PERIOD->value, $status->warnings);
        $this->assertTrue($status->requiresUserInput);
        $this->assertSame(DataQualityLevel::POOR, $status->dataQuality);
        $this->assertSame(ConfidenceLevel::LOW, $status->confidence);
    }

    public function test_a_newly_logged_period_cancels_period_expected(): void
    {
        // §22.5: the user logs a fresh start two days late → that day is
        // menstrual day 1 of a new confirmed cycle, not period_expected.
        $periods = [...$this->history21(), $this->period($this->day21(23), null)];
        $status = $this->resolve($periods, $this->profile(28, 5), $this->day21(23));

        $this->assertSame(MainPhase::MENSTRUAL, $status->mainPhase);
        $this->assertSame(1, $status->cycleDay);
        $this->assertSame(0, $status->daysLate);
        $this->assertSame(PeriodStartSource::USER_LOGGED, $status->currentPeriodStartSource);
    }

    public function test_long_bleed_empties_the_display_fertile_window(): void
    {
        // 21-day cycle with a logged 10-day bleed: O = day 8 ≤ period end day 10,
        // so display_fertile_end < display_fertile_start → no fertile day at all (§19).
        $periods = [
            $this->period('2025-11-09', '2025-11-18'),
            $this->period('2025-11-30', '2025-12-09'),
            $this->period('2025-12-21', '2025-12-30'),
            $this->period('2026-01-11', '2026-01-20'),
        ];

        for ($day = 1; $day <= 21; $day++) {
            $status = $this->resolve($periods, $this->profile(28, 5), $this->day21($day));
            $this->assertNotSame(MainPhase::FERTILE, $status->mainPhase, "day {$day}");
        }

        // Menstrual always overrides fertile: day 8 (the ovulation estimate) bleeds.
        $day8 = $this->resolve($periods, $this->profile(28, 5), $this->day21(8));
        $this->assertSame(MainPhase::MENSTRUAL, $day8->mainPhase);
    }

    public function test_28_day_cycle_follicular_split_and_fertile_tiers(): void
    {
        // §37: starts 28 apart, 5-day logged bleeds → O day 15, fertile 10–15,
        // follicular 6–9 splits 40/40/20 → early d6, mid d7, transition d8–9.
        $periods = [
            $this->period('2025-10-19', '2025-10-23'),
            $this->period('2025-11-16', '2025-11-20'),
            $this->period('2025-12-14', '2025-12-18'),
            $this->period('2026-01-11', '2026-01-15'),
        ];
        $expected = [
            6 => CycleSubphase::EARLY_FOLLICULAR,
            7 => CycleSubphase::MID_FOLLICULAR,
            8 => CycleSubphase::LATE_FOLLICULAR_TRANSITION,
            9 => CycleSubphase::LATE_FOLLICULAR_TRANSITION,
            10 => CycleSubphase::FERTILE_RISING,
            11 => CycleSubphase::FERTILE_RISING,
            12 => CycleSubphase::FERTILE_RISING,
            13 => CycleSubphase::HIGH_FERTILITY,
            14 => CycleSubphase::HIGH_FERTILITY,
            15 => CycleSubphase::OVULATION_LIKELY,
            16 => CycleSubphase::POST_OVULATION,
            17 => CycleSubphase::EARLY_LUTEAL,
            19 => CycleSubphase::EARLY_LUTEAL,
            20 => CycleSubphase::MID_LUTEAL,
            22 => CycleSubphase::MID_LUTEAL,
            23 => CycleSubphase::LATE_LUTEAL,
            25 => CycleSubphase::LATE_LUTEAL,
            26 => CycleSubphase::PMS_POSSIBLE,
            28 => CycleSubphase::PMS_POSSIBLE,
        ];

        foreach ($expected as $day => $subphase) {
            $status = $this->resolve($periods, $this->profile(28, 5), $this->day21($day));
            $this->assertSame($subphase, $status->subphase, "day {$day}");
        }

        $peak = $this->resolve($periods, $this->profile(28, 5), $this->day21(15));
        $this->assertSame(FertilityLevel::PEAK, $peak->fertilityLevel);
        $this->assertSame(0, $peak->daysToOvulation);
    }

    public function test_onboarding_only_resolve_is_onboarding_based_and_low_confidence(): void
    {
        // No logs at all — only the declared LMP and lengths from onboarding.
        $status = $this->resolve([], $this->profile(28, 5, '2026-01-11'), $this->day21(13));

        $this->assertSame(PeriodStartSource::ONBOARDING_DECLARED, $status->currentPeriodStartSource);
        $this->assertSame(ResolutionSource::ONBOARDING_BASED, $status->resolutionSource);
        $this->assertTrue($status->resolutionSource->isPredicted());
        $this->assertSame(ConfidenceLevel::LOW, $status->confidence);
        $this->assertSame(DataQualityLevel::PARTIAL, $status->dataQuality);
        $this->assertContains(CycleWarning::ONBOARDING_DATA_USED->value, $status->warnings);
        $this->assertContains(CycleWarning::LOW_HISTORY->value, $status->warnings);
    }

    public function test_defaults_only_resolve_is_default_based(): void
    {
        // A declared LMP but no declared lengths → system defaults (28/5).
        $status = $this->resolve([], $this->profile(null, null, '2026-01-11'), $this->day21(2));

        $this->assertSame(MainPhase::MENSTRUAL, $status->mainPhase);
        $this->assertSame(ResolutionSource::DEFAULT_BASED, $status->resolutionSource);
        $this->assertSame(28, $status->effectiveCycleLength);
        $this->assertSame(5, $status->effectivePeriodLength);
        $this->assertContains(CycleWarning::DEFAULT_VALUES_USED->value, $status->warnings);
        $this->assertSame(ConfidenceLevel::LOW, $status->confidence);
    }

    public function test_no_anchor_at_all_is_insufficient_and_unknown(): void
    {
        $status = $this->resolve([], null, '2026-01-15');

        $this->assertSame(MainPhase::UNKNOWN, $status->mainPhase);
        $this->assertNull($status->cycleDay);
        $this->assertSame(DataQualityLevel::INSUFFICIENT, $status->dataQuality);
        $this->assertSame(ConfidenceLevel::UNKNOWN, $status->confidence);
        $this->assertSame(ResolutionSource::UNKNOWN, $status->resolutionSource);
        $this->assertTrue($status->requiresUserInput);
        $this->assertContains(CycleWarning::INSUFFICIENT_ANCHOR_DATA->value, $status->warnings);
        $this->assertNull($status->currentPeriodStart);
    }

    public function test_future_dates_roll_the_anchor_forward_as_predicted_reference(): void
    {
        // Viewing day 23 of a 21-day cycle from day 13: the next (predicted)
        // cycle has started — its day 2 renders as a predicted menstrual day.
        $status = $this->resolve(
            $this->history21(),
            $this->profile(28, 5),
            $this->day21(23),
            $this->day21(13),
        );

        $this->assertSame(PeriodStartSource::PREDICTED_REFERENCE, $status->currentPeriodStartSource);
        $this->assertSame(2, $status->cycleDay);
        $this->assertSame(MainPhase::MENSTRUAL, $status->mainPhase);
        $this->assertSame(ResolutionSource::PREDICTION, $status->resolutionSource);
        $this->assertTrue($status->resolutionSource->isPredicted());
        $this->assertSame(0, $status->daysLate);
    }

    public function test_cycle_day_is_one_based(): void
    {
        $status = $this->resolve($this->history21(), $this->profile(28, 5), $this->day21(1));

        $this->assertSame(1, $status->cycleDay);
        $this->assertSame(MainPhase::MENSTRUAL, $status->mainPhase);
    }
}
