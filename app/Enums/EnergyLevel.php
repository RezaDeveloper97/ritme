<?php

namespace App\Enums;

/**
 * Self-reported daily energy level.
 *
 * Stored on daily_health_logs.energy_level. Also consumed by the unified
 * MessageSystem (low / very_low map to the "low_energy" symptom signal).
 */
enum EnergyLevel: string
{
    case VERY_LOW = 'very_low';
    case LOW = 'low';
    case MEDIUM = 'medium';
    case HIGH = 'high';
    case VERY_HIGH = 'very_high';

    public function label(string $locale = 'fa'): string
    {
        return $locale === 'fa'
            ? match ($this) {
                self::VERY_LOW => 'خیلی کم',
                self::LOW => 'کم',
                self::MEDIUM => 'متوسط',
                self::HIGH => 'زیاد',
                self::VERY_HIGH => 'خیلی زیاد',
            }
            : match ($this) {
                self::VERY_LOW => 'Very low',
                self::LOW => 'Low',
                self::MEDIUM => 'Medium',
                self::HIGH => 'High',
                self::VERY_HIGH => 'Very high',
            };
    }

    /**
     * Normalized 0-100 score used for weekly aggregation / charts.
     */
    public function score(): int
    {
        return match ($this) {
            self::VERY_LOW => 15,
            self::LOW => 35,
            self::MEDIUM => 60,
            self::HIGH => 80,
            self::VERY_HIGH => 100,
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
