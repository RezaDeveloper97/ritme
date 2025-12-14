<?php

namespace App\Enums;

enum SymptomSeverity: string
{
    case MILD = 'mild';
    case MODERATE = 'moderate';
    case SEVERE = 'severe';

    public function label(string $locale = 'en'): string
    {
        return match($locale) {
            'fa' => match($this) {
                self::MILD => 'خفیف',
                self::MODERATE => 'متوسط',
                self::SEVERE => 'شدید',
            },
            default => match($this) {
                self::MILD => 'Mild',
                self::MODERATE => 'Moderate',
                self::SEVERE => 'Severe',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
