<?php

namespace App\Enums;

enum PreExistingCondition: string
{
    case CHRONIC_HYPERTENSION = 'chronic_hypertension';
    case DIABETES = 'diabetes';
    case HYPOTHYROIDISM = 'hypothyroidism';
    case HYPERTHYROIDISM = 'hyperthyroidism';
    case NONE = 'none';

    public function label(string $locale = 'en'): string
    {
        return match($locale) {
            'fa' => match($this) {
                self::CHRONIC_HYPERTENSION => 'فشار خون مزمن',
                self::DIABETES => 'دیابت',
                self::HYPOTHYROIDISM => 'کم‌کاری تیروئید',
                self::HYPERTHYROIDISM => 'پرکاری تیروئید',
                self::NONE => 'هیچ‌کدام',
            },
            default => match($this) {
                self::CHRONIC_HYPERTENSION => 'Chronic Hypertension',
                self::DIABETES => 'Diabetes',
                self::HYPOTHYROIDISM => 'Hypothyroidism',
                self::HYPERTHYROIDISM => 'Hyperthyroidism',
                self::NONE => 'None',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
