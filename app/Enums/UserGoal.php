<?php

namespace App\Enums;

enum UserGoal: string
{
    case TTC = 'ttc';           // Trying to Conceive
    case NON_TTC = 'non_ttc';   // Not Trying to Conceive

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::TTC => 'اقدام به بارداری',
                self::NON_TTC => 'بدون قصد بارداری',
            },
            default => match ($this) {
                self::TTC => 'Trying to Conceive',
                self::NON_TTC => 'Not Trying to Conceive',
            },
        };
    }
}
