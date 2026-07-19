<?php

namespace App\Enums;

/**
 * Calendar-based fertility level shown on the daily card (spec §17). This is a
 * cycle-timing estimate only — never a diagnosis, and distinct from the TTC
 * conception-probability percentage the engine also computes.
 */
enum FertilityLevel: string
{
    case NONE = 'none';
    case VERY_LOW = 'very_low';
    case LOW = 'low';
    case MEDIUM = 'medium';
    case HIGH = 'high';
    case VERY_HIGH = 'very_high';
    /** v1.1 (task.md §26): the estimated ovulation day itself. */
    case PEAK = 'peak';
    /** v1.1 (task.md §26): cycle unresolved (period_expected / unknown states). */
    case UNKNOWN = 'unknown';

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::NONE => 'هیچ',
                self::VERY_LOW => 'خیلی پایین',
                self::LOW => 'پایین',
                self::MEDIUM => 'متوسط',
                self::HIGH => 'بالا',
                self::VERY_HIGH => 'خیلی بالا',
                self::PEAK => 'اوج',
                self::UNKNOWN => 'نامشخص',
            },
            default => match ($this) {
                self::NONE => 'None',
                self::VERY_LOW => 'Very low',
                self::LOW => 'Low',
                self::MEDIUM => 'Medium',
                self::HIGH => 'High',
                self::VERY_HIGH => 'Very high',
                self::PEAK => 'Peak',
                self::UNKNOWN => 'Unknown',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
