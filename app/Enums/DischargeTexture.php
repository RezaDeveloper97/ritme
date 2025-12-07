<?php

namespace App\Enums;

enum DischargeTexture: string
{
    case WATERY = 'watery';
    case CREAMY = 'creamy';
    case EGG_WHITE = 'egg_white';
    case THICK = 'thick';

    public function label(): string
    {
        return match($this) {
            self::WATERY => 'آبکی',
            self::CREAMY => 'کرمی',
            self::EGG_WHITE => 'سفیده تخم‌مرغی',
            self::THICK => 'غلیظ',
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
