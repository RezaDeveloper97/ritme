<?php

namespace App\Enums;

enum MentalHealthStatus: string
{
    case GOOD = 'good';
    case MODERATE = 'moderate';
    case POOR = 'poor';

    public function label(string $locale = 'en'): string
    {
        return match($locale) {
            'fa' => match($this) {
                self::GOOD => 'خوب',
                self::MODERATE => 'متوسط',
                self::POOR => 'ضعیف',
            },
            default => match($this) {
                self::GOOD => 'Good',
                self::MODERATE => 'Moderate',
                self::POOR => 'Poor',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
