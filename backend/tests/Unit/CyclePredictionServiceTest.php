<?php

namespace Tests\Unit;

use App\Enums\EffectiveSource;
use App\Services\HealthEngine\CyclePredictionService;
use Carbon\Carbon;
use PHPUnit\Framework\TestCase;

/**
 * Date-math tests for the Start-anchored prediction (spec §2, §10). Ovulation is
 * next-period-start minus 14; the fertile window is −5/+1 around it.
 */
class CyclePredictionServiceTest extends TestCase
{
    private CyclePredictionService $service;

    protected function setUp(): void
    {
        parent::setUp();
        $this->service = new CyclePredictionService;
    }

    public function test_predicts_next_period_ovulation_and_fertile_window(): void
    {
        $prediction = $this->service->predict(
            Carbon::parse('2026-07-01'),
            28,
            5,
            Carbon::parse('2026-07-15'),
            EffectiveSource::RECENT_VALID_CYCLES,
        );

        $this->assertSame('2026-07-01', $prediction->currentCycleStart->toDateString());
        $this->assertSame('2026-07-29', $prediction->nextPeriodStart->toDateString());
        $this->assertSame('2026-08-02', $prediction->nextPeriodEnd->toDateString());
        $this->assertSame('2026-07-15', $prediction->estimatedOvulationDate->toDateString());
        $this->assertSame('2026-07-10', $prediction->fertileWindowStart->toDateString());
        $this->assertSame('2026-07-16', $prediction->fertileWindowEnd->toDateString());
        $this->assertSame(EffectiveSource::RECENT_VALID_CYCLES, $prediction->source);
    }

    public function test_projects_the_anchor_forward_when_several_cycles_have_passed(): void
    {
        // Anchor two-plus cycles in the past with no new log: the prediction rolls
        // forward to the cycle that actually contains the reference date.
        $prediction = $this->service->predict(
            Carbon::parse('2026-07-01'),
            28,
            5,
            Carbon::parse('2026-09-10'),
            EffectiveSource::PROFILE,
        );

        $this->assertSame('2026-08-26', $prediction->currentCycleStart->toDateString());
        $this->assertSame('2026-09-23', $prediction->nextPeriodStart->toDateString());
    }

    public function test_uses_the_period_duration_for_the_predicted_end(): void
    {
        $prediction = $this->service->predict(
            Carbon::parse('2026-07-01'),
            30,
            7,
            Carbon::parse('2026-07-01'),
            EffectiveSource::RECENT_VALID_CYCLES,
        );

        $this->assertSame('2026-07-31', $prediction->nextPeriodStart->toDateString());
        $this->assertSame('2026-08-06', $prediction->nextPeriodEnd->toDateString()); // 7-day period
        $this->assertSame('2026-07-17', $prediction->estimatedOvulationDate->toDateString()); // 07-31 − 14
    }
}
