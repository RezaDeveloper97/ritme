<?php

namespace App\Enums;

enum Amount: string
{
    case LOW = 'low';
    case MEDIUM = 'medium';
    case HIGH = 'high';

    public function label(): string
    {
        return match($this) {
            self::LOW => 'کم',
            self::MEDIUM => 'متوسط',
            self::HIGH => 'زیاد',
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
