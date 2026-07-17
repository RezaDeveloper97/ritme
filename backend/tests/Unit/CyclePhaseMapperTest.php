<?php

namespace Tests\Unit;

use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;
use App\Enums\FertilityLevel;
use App\Services\HealthEngine\CyclePhaseMapper;
use PHPUnit\Framework\TestCase;

/**
 * Phase / sub-phase mapping (spec §16–17) over a canonical 28-day cycle: ovulation
 * day O = 14, 5-day bleed. Boundaries are ovulation-relative, so this also guards the
 * fix for the old fixed day-12 / 13–15 phase cutoffs.
 */
class CyclePhaseMapperTest extends TestCase
{
    private CyclePhaseMapper $mapper;

    private const OVULATION = 14;

    private const CYCLE = 28;

    private const BLEED = 5;

    protected function setUp(): void
    {
        parent::setUp();
        $this->mapper = new CyclePhaseMapper;
    }

    private function phase(int $day): CyclePhase
    {
        return $this->mapper->phaseFor($day, self::OVULATION, self::BLEED);
    }

    private function subphase(int $day): CycleSubphase
    {
        return $this->mapper->subphaseFor($day, self::OVULATION, self::CYCLE, self::BLEED);
    }

    public function test_phase_boundaries_follow_the_ovulation_day(): void
    {
        $this->assertSame(CyclePhase::MENSTRUATION, $this->phase(3));
        $this->assertSame(CyclePhase::FOLLICULAR, $this->phase(6));
        $this->assertSame(CyclePhase::FOLLICULAR, $this->phase(12));
        $this->assertSame(CyclePhase::FOLLICULAR, $this->phase(13)); // O-1 stays follicular
        $this->assertSame(CyclePhase::OVULATION, $this->phase(14)); // O
        $this->assertSame(CyclePhase::OVULATION, $this->phase(15)); // O+1
        $this->assertSame(CyclePhase::LUTEAL, $this->phase(16));
        $this->assertSame(CyclePhase::LUTEAL, $this->phase(28));
    }

    public function test_subphase_walks_the_whole_cycle(): void
    {
        $expected = [
            1 => CycleSubphase::MENSTRUATION,
            5 => CycleSubphase::MENSTRUATION,
            6 => CycleSubphase::EARLY_FOLLICULAR,
            8 => CycleSubphase::MID_FOLLICULAR,
            9 => CycleSubphase::FERTILE_RISING,   // O-5
            11 => CycleSubphase::FERTILE_RISING,  // O-3
            12 => CycleSubphase::HIGH_FERTILITY,  // O-2
            13 => CycleSubphase::HIGH_FERTILITY,  // O-1
            14 => CycleSubphase::OVULATION_LIKELY, // O
            15 => CycleSubphase::POST_OVULATION,  // O+1
            16 => CycleSubphase::EARLY_LUTEAL,    // O+2
            19 => CycleSubphase::EARLY_LUTEAL,    // O+5
            20 => CycleSubphase::MID_LUTEAL,      // O+6
            23 => CycleSubphase::MID_LUTEAL,      // P-6
            24 => CycleSubphase::LATE_LUTEAL,     // P-5
            26 => CycleSubphase::LATE_LUTEAL,     // P-3
            27 => CycleSubphase::PMS_POSSIBLE,    // P-2
            28 => CycleSubphase::PMS_POSSIBLE,    // P-1
        ];

        foreach ($expected as $day => $subphase) {
            $this->assertSame($subphase, $this->subphase($day), "cycle day {$day}");
        }
    }

    public function test_every_day_maps_to_some_subphase_without_gaps(): void
    {
        for ($day = 1; $day <= self::CYCLE; $day++) {
            $this->assertInstanceOf(CycleSubphase::class, $this->subphase($day));
        }
    }

    public function test_subphase_maps_to_the_spec_fertility_level(): void
    {
        $this->assertSame(FertilityLevel::VERY_HIGH, CycleSubphase::OVULATION_LIKELY->fertilityLevel());
        $this->assertSame(FertilityLevel::HIGH, CycleSubphase::HIGH_FERTILITY->fertilityLevel());
        $this->assertSame(FertilityLevel::MEDIUM, CycleSubphase::FERTILE_RISING->fertilityLevel());
        $this->assertSame(FertilityLevel::MEDIUM, CycleSubphase::POST_OVULATION->fertilityLevel());
        $this->assertSame(FertilityLevel::LOW, CycleSubphase::MENSTRUATION->fertilityLevel());
        $this->assertSame(FertilityLevel::LOW, CycleSubphase::MID_LUTEAL->fertilityLevel());
        $this->assertSame(FertilityLevel::LOW, CycleSubphase::PERIOD_EXPECTED->fertilityLevel());
    }
}
