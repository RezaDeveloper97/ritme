<?php

namespace App\Enums;

enum SleepQuality: string
{
    case GOOD = 'good';
    case MEDIUM = 'medium';
    case BAD = 'bad';

    public function label(): string
    {
        return match($this) {
            self::GOOD => 'خوب',
            self::MEDIUM => 'متوسط',
            self::BAD => 'بد',
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
