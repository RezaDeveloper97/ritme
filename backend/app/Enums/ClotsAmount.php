<?php

namespace App\Enums;

/**
 * How much clotting was seen (daily_health_logs.clots_amount).
 *
 * Replaces the older boolean `has_clots`, which only recorded presence. NONE is
 * an explicit choice so "checked and there were none" is distinguishable from
 * "not recorded" (null).
 */
enum ClotsAmount: string
{
    case NONE = 'none';
    case LOW = 'low';
    case MEDIUM = 'medium';
    case HIGH = 'high';

    public function label(string $locale = 'fa'): string
    {
        return $locale === 'fa'
            ? match ($this) {
                self::NONE => 'ندارد',
                self::LOW => 'کم',
                self::MEDIUM => 'متوسط',
                self::HIGH => 'زیاد',
            }
        : match ($this) {
            self::NONE => 'None',
            self::LOW => 'Light',
            self::MEDIUM => 'Moderate',
            self::HIGH => 'Heavy',
        };
    }

    /** True when clotting was actually recorded (NONE and null are not). */
    public function isPresent(): bool
    {
        return $this !== self::NONE;
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
