<?php

namespace App\Enums;

/**
 * How hard the recorded activity felt (daily_health_logs.exercise_intensity).
 *
 * Kept separate from PainIntensity even though the values match today — the two
 * scales are unrelated domains and are free to diverge.
 */
enum ExerciseIntensity: string
{
    case LOW = 'low';
    case MEDIUM = 'medium';
    case HIGH = 'high';

    public function label(string $locale = 'fa'): string
    {
        return $locale === 'fa'
            ? match ($this) {
                self::LOW => 'کم',
                self::MEDIUM => 'متوسط',
                self::HIGH => 'زیاد',
            }
        : match ($this) {
            self::LOW => 'Low',
            self::MEDIUM => 'Medium',
            self::HIGH => 'High',
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
