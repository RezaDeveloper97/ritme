<?php

namespace App\Enums;

/**
 * Whether the day's intercourse was protected (daily_health_logs.intercourse_type).
 *
 * Split out of the old multi-select `sexual_activities` so the answer is a
 * single, unambiguous value — the fertility engines key on it.
 */
enum IntercourseType: string
{
    case PROTECTED = 'protected';
    case UNPROTECTED = 'unprotected';

    public function label(string $locale = 'fa'): string
    {
        return $locale === 'fa'
            ? match ($this) {
                self::PROTECTED => 'رابطهٔ محافظت‌شده',
                self::UNPROTECTED => 'رابطهٔ محافظت‌نشده',
            }
        : match ($this) {
            self::PROTECTED => 'Protected intercourse',
            self::UNPROTECTED => 'Unprotected intercourse',
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
