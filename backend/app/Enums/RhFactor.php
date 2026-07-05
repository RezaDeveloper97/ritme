<?php

namespace App\Enums;

enum RhFactor: string
{
    case POSITIVE = 'positive';
    case NEGATIVE = 'negative';

    public function label(string $locale = 'en'): string
    {
        return match($locale) {
            'fa' => match($this) {
                self::POSITIVE => 'مثبت',
                self::NEGATIVE => 'منفی',
            },
            default => match($this) {
                self::POSITIVE => 'Positive',
                self::NEGATIVE => 'Negative',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
