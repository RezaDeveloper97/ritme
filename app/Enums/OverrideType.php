<?php

namespace App\Enums;

enum OverrideType: string
{
    // General Overrides
    case NORMAL = 'normal';
    case PAIN = 'pain';
    case MOOD_SAD = 'mood_sad';
    case MOOD_ANGRY = 'mood_angry';
    case HEAVY_FLOW = 'heavy_flow';
    case SKIN_ACNE = 'skin_acne';
    case DISCHARGE = 'discharge';
    case HEADACHE = 'headache';
    case OVARIAN_PAIN = 'ovarian_pain';
    case STRESS = 'stress';
    case SPOTTING = 'spotting';
    case PMS = 'pms';

    // TTC Specific Overrides
    case TTC_GRIEF = 'ttc_grief';
    case TTC_PERFORMANCE_ANXIETY = 'ttc_performance_anxiety';
    case NEGATIVE_TEST = 'negative_test';
    case IMPLANTATION_SPOTTING = 'implantation_spotting';

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::NORMAL => 'حالت نرمال',
                self::PAIN => 'درد شدید',
                self::MOOD_SAD => 'غمگینی',
                self::MOOD_ANGRY => 'عصبانیت',
                self::HEAVY_FLOW => 'خونریزی شدید',
                self::SKIN_ACNE => 'جوش صورت',
                self::DISCHARGE => 'ترشحات',
                self::HEADACHE => 'سردرد',
                self::OVARIAN_PAIN => 'درد تخمدان',
                self::STRESS => 'استرس',
                self::SPOTTING => 'لکه‌بینی',
                self::PMS => 'سندروم پیش از قاعدگی',
                self::TTC_GRIEF => 'ناراحتی از پریود شدن',
                self::TTC_PERFORMANCE_ANXIETY => 'اضطراب عملکرد',
                self::NEGATIVE_TEST => 'تست منفی',
                self::IMPLANTATION_SPOTTING => 'لکه‌بینی لانه‌گزینی',
            },
            default => match ($this) {
                self::NORMAL => 'Normal',
                self::PAIN => 'Severe Pain',
                self::MOOD_SAD => 'Sad Mood',
                self::MOOD_ANGRY => 'Angry Mood',
                self::HEAVY_FLOW => 'Heavy Flow',
                self::SKIN_ACNE => 'Acne',
                self::DISCHARGE => 'Discharge',
                self::HEADACHE => 'Headache',
                self::OVARIAN_PAIN => 'Ovarian Pain',
                self::STRESS => 'Stress',
                self::SPOTTING => 'Spotting',
                self::PMS => 'PMS',
                self::TTC_GRIEF => 'TTC Grief',
                self::TTC_PERFORMANCE_ANXIETY => 'Performance Anxiety',
                self::NEGATIVE_TEST => 'Negative Test',
                self::IMPLANTATION_SPOTTING => 'Implantation Spotting',
            },
        };
    }
}
