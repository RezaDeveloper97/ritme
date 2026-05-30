<?php

namespace App\Enums;

/**
 * Category of a daily home-page task (وظایف امروز).
 * Each category carries a default icon key for the frontend.
 */
enum TaskCategory: string
{
    case TRACKING = 'tracking';
    case HYDRATION = 'hydration';
    case NUTRITION = 'nutrition';
    case SUPPLEMENT = 'supplement';
    case EXERCISE = 'exercise';
    case MINDFULNESS = 'mindfulness';
    case MEDICAL = 'medical';

    public function label(string $locale = 'fa'): string
    {
        return $locale === 'fa'
            ? match ($this) {
                self::TRACKING => 'ثبت و پایش',
                self::HYDRATION => 'آب‌رسانی',
                self::NUTRITION => 'تغذیه',
                self::SUPPLEMENT => 'مکمل',
                self::EXERCISE => 'ورزش',
                self::MINDFULNESS => 'آرامش ذهن',
                self::MEDICAL => 'پزشکی',
            }
            : match ($this) {
                self::TRACKING => 'Tracking',
                self::HYDRATION => 'Hydration',
                self::NUTRITION => 'Nutrition',
                self::SUPPLEMENT => 'Supplement',
                self::EXERCISE => 'Exercise',
                self::MINDFULNESS => 'Mindfulness',
                self::MEDICAL => 'Medical',
            };
    }

    /**
     * Default icon key (frontend maps it to an asset).
     */
    public function icon(): string
    {
        return match ($this) {
            self::TRACKING => 'clipboard-check',
            self::HYDRATION => 'water-glass',
            self::NUTRITION => 'apple',
            self::SUPPLEMENT => 'pill',
            self::EXERCISE => 'walking',
            self::MINDFULNESS => 'lotus',
            self::MEDICAL => 'stethoscope',
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
