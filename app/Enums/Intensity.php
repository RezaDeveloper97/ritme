<?php

namespace App\Enums;

enum Intensity: string
{
    case LOW = 'low';
    case MEDIUM = 'medium';
    case HIGH = 'high';
    case VERY_HIGH = 'very_high';

    public function label(): string
    {
        return match($this) {
            self::LOW => 'کم',
            self::MEDIUM => 'متوسط',
            self::HIGH => 'زیاد',
            self::VERY_HIGH => 'خیلی زیاد',
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
