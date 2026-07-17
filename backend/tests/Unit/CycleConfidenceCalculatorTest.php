<?php

namespace Tests\Unit;

use App\Enums\ConfidenceLevel;
use App\Enums\CycleVariability;
use App\Enums\EffectiveSource;
use App\Enums\RegularityStatus;
use App\Services\HealthEngine\CycleConfidenceCalculator;
use App\Services\HealthEngine\CycleMetrics;
use PHPUnit\Framework\TestCase;

/**
 * Prediction confidence grading (spec §18): more valid cycles and lower variation
 * raise it; a long-missing period end lowers it.
 */
class CycleConfidenceCalculatorTest extends TestCase
{
    private CycleConfidenceCalculator $calculator;

    protected function setUp(): void
    {
        parent::setUp();
        $this->calculator = new CycleConfidenceCalculator;
    }

    private function metrics(int $validCycles, CycleVariability $variability): CycleMetrics
    {
        return new CycleMetrics(
            profileCycleLength: 28,
            profilePeriodDuration: 5,
            calculatedCycleLength: $validCycles >= 3 ? 28 : null,
            calculatedPeriodDuration: null,
            effectiveCycleLength: 28,
            effectivePeriodDuration: 5,
            cycleLengthSource: EffectiveSource::PROFILE,
            periodDurationSource: EffectiveSource::PROFILE,
            validCyclesCount: $validCycles,
            validPeriodDurationsCount: 0,
            regularityStatus: RegularityStatus::NOT_ENOUGH_DATA,
            variability: $variability,
            recentValidCycleLengths: [],
        );
    }

    public function test_three_regular_cycles_give_high_confidence(): void
    {
        $confidence = $this->calculator->forPrediction($this->metrics(3, CycleVariability::REGULAR));

        $this->assertSame(ConfidenceLevel::HIGH, $confidence->level);
        $this->assertContains(CycleConfidenceCalculator::REASON_THREE_VALID_CYCLES, $confidence->reasons);
        $this->assertContains(CycleConfidenceCalculator::REASON_LOW_VARIATION, $confidence->reasons);
    }

    public function test_three_irregular_cycles_are_only_medium(): void
    {
        $confidence = $this->calculator->forPrediction($this->metrics(4, CycleVariability::IRREGULAR));

        $this->assertSame(ConfidenceLevel::MEDIUM, $confidence->level);
        $this->assertContains(CycleConfidenceCalculator::REASON_CYCLE_VARIATION, $confidence->reasons);
    }

    public function test_one_or_two_cycles_are_medium(): void
    {
        $confidence = $this->calculator->forPrediction($this->metrics(2, CycleVariability::REGULAR));

        $this->assertSame(ConfidenceLevel::MEDIUM, $confidence->level);
        $this->assertContains(CycleConfidenceCalculator::REASON_NOT_ENOUGH_CYCLES, $confidence->reasons);
    }

    public function test_profile_only_is_low(): void
    {
        $confidence = $this->calculator->forPrediction($this->metrics(0, CycleVariability::REGULAR));

        $this->assertSame(ConfidenceLevel::LOW, $confidence->level);
        $this->assertContains(CycleConfidenceCalculator::REASON_PROFILE_ONLY, $confidence->reasons);
    }

    public function test_a_long_missing_end_downgrades_and_annotates(): void
    {
        $confidence = $this->calculator->forPrediction($this->metrics(3, CycleVariability::REGULAR), endMissingLong: true);

        $this->assertSame(ConfidenceLevel::MEDIUM, $confidence->level); // downgraded from HIGH
        $this->assertContains(CycleConfidenceCalculator::REASON_END_MISSING_LONG, $confidence->reasons);
    }
}
