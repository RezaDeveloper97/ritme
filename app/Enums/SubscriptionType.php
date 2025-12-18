<?php

namespace App\Enums;

enum SubscriptionType: string
{
    case FREE = 'free';
    case PREMIUM = 'premium';

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::FREE => 'رایگان',
                self::PREMIUM => 'پرمیوم',
            },
            default => match ($this) {
                self::FREE => 'Free',
                self::PREMIUM => 'Premium',
            },
        };
    }
}
