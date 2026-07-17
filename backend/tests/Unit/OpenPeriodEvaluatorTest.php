<?php

namespace Tests\Unit;

use App\Enums\DataQualityFlag;
use App\Services\HealthEngine\OpenPeriodEvaluator;
use Carbon\Carbon;
use PHPUnit\Framework\TestCase;

/**
 * Open-period handling (spec §4): while a bleed is within the expected length it is
 * shown as active; past that the user is asked to log the end; past the hard cap the
 * app stops presenting an active period. No actual end is ever fabricated.
 */
class OpenPeriodEvaluatorTest extends TestCase
{
    private OpenPeriodEvaluator $evaluator;

    protected function setUp(): void
    {
        parent::setUp();
        $this->evaluator = new OpenPeriodEvaluator;
    }

    public function test_no_open_period_returns_the_empty_state(): void
    {
        $state = $this->evaluator->evaluate(null, 5, 5, Carbon::parse('2026-07-10'));

        $this->assertFalse($state->hasOpenPeriod);
        $this->assertNull($state->menstrualDay);
        $this->assertFalse($state->displayAsActiveMenstrual);
    }

    public function test_within_expected_length_is_shown_as_active(): void
    {
        // Profile duration 5 → max_expected = max(8, 8) = 8. Day 3 is well inside it.
        $state = $this->evaluator->evaluate(Carbon::parse('2026-07-01'), 5, 5, Carbon::parse('2026-07-03'));

        $this->assertTrue($state->hasOpenPeriod);
        $this->assertSame(3, $state->menstrualDay);
        $this->assertTrue($state->displayAsActiveMenstrual);
        $this->assertFalse($state->endOverdue);
        $this->assertFalse($state->pastHardCap);
        $this->assertSame('2026-07-05', $state->assumedEndDate->toDateString()); // start + 5 − 1
        $this->assertSame([], $state->dataQualityFlags);
    }

    public function test_past_expected_length_flags_incomplete_and_asks_for_the_end(): void
    {
        // Day 10 > max_expected 8, but still under the 12-day hard cap.
        $state = $this->evaluator->evaluate(Carbon::parse('2026-07-01'), 5, 5, Carbon::parse('2026-07-10'));

        $this->assertSame(10, $state->menstrualDay);
        $this->assertFalse($state->displayAsActiveMenstrual);
        $this->assertTrue($state->endOverdue);
        $this->assertFalse($state->pastHardCap);
        $this->assertTrue($state->hasFlag(DataQualityFlag::INCOMPLETE_END_MISSING));
    }

    public function test_past_hard_cap_stops_presenting_an_active_period(): void
    {
        // Day 14 > hard cap 12.
        $state = $this->evaluator->evaluate(Carbon::parse('2026-07-01'), 5, 5, Carbon::parse('2026-07-14'));

        $this->assertSame(14, $state->menstrualDay);
        $this->assertFalse($state->displayAsActiveMenstrual);
        $this->assertTrue($state->endOverdue);
        $this->assertTrue($state->pastHardCap);
    }

    public function test_max_expected_follows_the_profile_period_duration(): void
    {
        // Profile duration 7 → max_expected = max(10, 8) = 10. Day 9 still active.
        $state = $this->evaluator->evaluate(Carbon::parse('2026-07-01'), 5, 7, Carbon::parse('2026-07-09'));

        $this->assertSame(9, $state->menstrualDay);
        $this->assertTrue($state->displayAsActiveMenstrual);
    }

    public function test_a_future_start_is_not_an_ongoing_bleed(): void
    {
        $state = $this->evaluator->evaluate(Carbon::parse('2026-07-20'), 5, 5, Carbon::parse('2026-07-10'));

        $this->assertFalse($state->hasOpenPeriod);
    }

    public function test_hard_cap_dominates_a_long_profile_period_duration(): void
    {
        // Profile duration 12 → max_expected = max(15, 8) = 15, but the 12-day hard cap
        // must win: day 13 is past the cap, so it is no longer shown as active.
        $state = $this->evaluator->evaluate(Carbon::parse('2026-07-01'), 5, 12, Carbon::parse('2026-07-13'));

        $this->assertSame(13, $state->menstrualDay);
        $this->assertFalse($state->displayAsActiveMenstrual);
        $this->assertTrue($state->endOverdue);
        $this->assertTrue($state->pastHardCap);
    }
}
