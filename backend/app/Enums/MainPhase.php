<?php

namespace App\Enums;

/**
 * Final v1.1 main phase (task.md §13). Unlike the display-oriented {@see CyclePhase},
 * this is the engine's authoritative classification: ovulation is only ever a
 * subphase, and two non-biological states exist — period_expected (a predicted
 * period that has not been logged yet) and unknown (not enough anchor data).
 */
enum MainPhase: string
{
    case MENSTRUAL = 'menstrual';
    case FOLLICULAR = 'follicular';
    case FERTILE = 'fertile';
    case LUTEAL = 'luteal';
    case PERIOD_EXPECTED = 'period_expected';
    case UNKNOWN = 'unknown';

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::MENSTRUAL => 'قاعدگی',
                self::FOLLICULAR => 'فولیکولار',
                self::FERTILE => 'باروری',
                self::LUTEAL => 'لوتئال',
                self::PERIOD_EXPECTED => 'انتظار پریود',
                self::UNKNOWN => 'نامشخص',
            },
            default => match ($this) {
                self::MENSTRUAL => 'Menstrual',
                self::FOLLICULAR => 'Follicular',
                self::FERTILE => 'Fertile',
                self::LUTEAL => 'Luteal',
                self::PERIOD_EXPECTED => 'Period Expected',
                self::UNKNOWN => 'Unknown',
            },
        };
    }

    /**
     * The legacy four-phase value the pre-v1.1 clients still read from
     * `cycle_view.phase`. period_expected renders as late luteal; unknown has no
     * legacy equivalent and maps to null.
     */
    public function legacyPhase(): ?CyclePhase
    {
        return match ($this) {
            self::MENSTRUAL => CyclePhase::MENSTRUATION,
            self::FOLLICULAR => CyclePhase::FOLLICULAR,
            self::FERTILE => CyclePhase::OVULATION,
            self::LUTEAL, self::PERIOD_EXPECTED => CyclePhase::LUTEAL,
            self::UNKNOWN => null,
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
