<?php

namespace App\Enums;

enum ConfidenceLevel: string
{
    case HIGH = 'high';
    case MEDIUM = 'medium';
    case LOW = 'low';

    public function label(string $locale = 'en'): string
    {
        return match($locale) {
            'fa' => match($this) {
                self::HIGH => 'بالا',
                self::MEDIUM => 'متوسط',
                self::LOW => 'پایین',
            },
            default => match($this) {
                self::HIGH => 'High',
                self::MEDIUM => 'Medium',
                self::LOW => 'Low',
            },
        };
    }

    public function description(string $locale = 'en'): string
    {
        return match($locale) {
            'fa' => match($this) {
                self::HIGH => 'مبتنی بر سونوگرافی',
                self::MEDIUM => 'مبتنی بر LMP دقیق',
                self::LOW => 'ورود نامطمئن',
            },
            default => match($this) {
                self::HIGH => 'Based on ultrasound',
                self::MEDIUM => 'Based on accurate LMP',
                self::LOW => 'Uncertain entry',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
