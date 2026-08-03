<?php

namespace App\Enums;

/**
 * Self-reported libido relative to the user's own baseline
 * (daily_health_logs.sexual_desire).
 */
enum SexualDesire: string
{
    case LOWER = 'lower';
    case NORMAL = 'normal';
    case HIGHER = 'higher';

    public function label(string $locale = 'fa'): string
    {
        return $locale === 'fa'
            ? match ($this) {
                self::LOWER => 'کمتر از معمول',
                self::NORMAL => 'معمولی',
                self::HIGHER => 'بیشتر از معمول',
            }
        : match ($this) {
            self::LOWER => 'Lower than usual',
            self::NORMAL => 'Usual',
            self::HIGHER => 'Higher than usual',
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
