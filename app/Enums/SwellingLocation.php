<?php

namespace App\Enums;

enum SwellingLocation: string
{
    case FEET = 'feet';
    case HANDS = 'hands';
    case FACE = 'face';

    public function label(string $locale = 'en'): string
    {
        return match($locale) {
            'fa' => match($this) {
                self::FEET => 'پا',
                self::HANDS => 'دست',
                self::FACE => 'صورت',
            },
            default => match($this) {
                self::FEET => 'Feet',
                self::HANDS => 'Hands',
                self::FACE => 'Face',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
