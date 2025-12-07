<?php

namespace App\Enums;

enum BloodColor: string
{
    case BRIGHT_RED = 'bright_red';
    case RED = 'red';
    case DARK_RED = 'dark_red';
    case BROWN = 'brown';

    public function label(): string
    {
        return match($this) {
            self::BRIGHT_RED => 'قرمز روشن',
            self::RED => 'قرمز',
            self::DARK_RED => 'قرمز تیره',
            self::BROWN => 'قهوه‌ای',
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
