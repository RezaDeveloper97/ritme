<?php

namespace App\Enums;

/**
 * Calendar-based fertility level shown on the daily card (spec §17). This is a
 * cycle-timing estimate only — never a diagnosis, and distinct from the TTC
 * conception-probability percentage the engine also computes.
 */
enum FertilityLevel: string
{
    case VERY_LOW = 'very_low';
    case LOW = 'low';
    case MEDIUM = 'medium';
    case HIGH = 'high';
    case VERY_HIGH = 'very_high';

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::VERY_LOW => 'خیلی پایین',
                self::LOW => 'پایین',
                self::MEDIUM => 'متوسط',
                self::HIGH => 'بالا',
                self::VERY_HIGH => 'خیلی بالا',
            },
            default => match ($this) {
                self::VERY_LOW => 'Very low',
                self::LOW => 'Low',
                self::MEDIUM => 'Medium',
                self::HIGH => 'High',
                self::VERY_HIGH => 'Very high',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
