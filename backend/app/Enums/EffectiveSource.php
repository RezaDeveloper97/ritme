<?php

namespace App\Enums;

/**
 * Which layer a resolved "effective" value came from (spec §12/§14): learned from
 * recent valid cycles, the user's profile baseline, or the system default.
 */
enum EffectiveSource: string
{
    case RECENT_VALID_CYCLES = 'recent_valid_cycles';
    case PROFILE = 'profile';
    case DEFAULT = 'default';

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::RECENT_VALID_CYCLES => 'ثبت‌های اخیر',
                self::PROFILE => 'مقدار پروفایل',
                self::DEFAULT => 'پیش‌فرض',
            },
            default => match ($this) {
                self::RECENT_VALID_CYCLES => 'Recent logs',
                self::PROFILE => 'Profile value',
                self::DEFAULT => 'Default',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
