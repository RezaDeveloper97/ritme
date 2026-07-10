<?php

namespace App\Enums;

/**
 * Optional, self-reported chronic conditions collected at onboarding to make
 * health recommendations more accurate. General (not pregnancy-scoped); stored
 * as a JSON array on user_profiles. An empty array means "none".
 */
enum ChronicCondition: string
{
    case PCOS = 'pcos';
    case HYPOTHYROIDISM = 'hypothyroidism';
    case HYPERTHYROIDISM = 'hyperthyroidism';
    case HYPERTENSION = 'hypertension';
    case HEART_DISEASE = 'heart_disease';
    case DIABETES = 'diabetes';

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::PCOS => 'تخمدان پلی‌کیستیک (PCOS)',
                self::HYPOTHYROIDISM => 'کم‌کاری تیروئید',
                self::HYPERTHYROIDISM => 'پرکاری تیروئید',
                self::HYPERTENSION => 'فشار خون بالا',
                self::HEART_DISEASE => 'بیماری قلبی',
                self::DIABETES => 'دیابت (نوع ۱، ۲ یا بارداری)',
            },
            default => match ($this) {
                self::PCOS => 'Polycystic ovary syndrome (PCOS)',
                self::HYPOTHYROIDISM => 'Hypothyroidism',
                self::HYPERTHYROIDISM => 'Hyperthyroidism',
                self::HYPERTENSION => 'High blood pressure',
                self::HEART_DISEASE => 'Heart disease',
                self::DIABETES => 'Diabetes (type 1, 2 or gestational)',
            },
        };
    }
}
