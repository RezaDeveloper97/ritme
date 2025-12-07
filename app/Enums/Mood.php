<?php

namespace App\Enums;

enum Mood: string
{
    case HAPPY = 'happy';
    case CALM = 'calm';
    case ANGRY = 'angry';
    case ANXIOUS = 'anxious';
    case SAD = 'sad';
    case FRUSTRATED = 'frustrated';
    case SENSITIVE = 'sensitive';
    case BORED = 'bored';

    public function label(): string
    {
        return match($this) {
            self::HAPPY => 'شاد',
            self::CALM => 'آرام',
            self::ANGRY => 'عصبی',
            self::ANXIOUS => 'نگران',
            self::SAD => 'ناراحت',
            self::FRUSTRATED => 'کلافه',
            self::SENSITIVE => 'حساس',
            self::BORED => 'بی‌حوصله',
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
