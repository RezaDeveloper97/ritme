<?php

namespace App\Enums;

enum FetalMovementStatus: string
{
    case NOT_FELT_YET = 'not_felt_yet';
    case FELT = 'felt';
    case NORMAL = 'normal';
    case REDUCED = 'reduced';
    case INCREASED = 'increased';
    case NONE = 'none';

    public function label(string $locale = 'en'): string
    {
        return match($locale) {
            'fa' => match($this) {
                self::NOT_FELT_YET => 'هنوز حس نکرده‌ام',
                self::FELT => 'حس کرده‌ام',
                self::NORMAL => 'طبیعی',
                self::REDUCED => 'کاهش یافته',
                self::INCREASED => 'افزایش یافته',
                self::NONE => 'بدون حرکت',
            },
            default => match($this) {
                self::NOT_FELT_YET => 'Not felt yet',
                self::FELT => 'Felt',
                self::NORMAL => 'Normal',
                self::REDUCED => 'Reduced',
                self::INCREASED => 'Increased',
                self::NONE => 'No movement',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
