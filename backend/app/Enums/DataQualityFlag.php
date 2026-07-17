<?php

namespace App\Enums;

/**
 * Non-fatal data-quality markers on a logged period or cycle (spec §8, §16). A flagged
 * record stays in history but may be held out of the prediction medians; a flag never
 * blocks the user from logging.
 */
enum DataQualityFlag: string
{
    case INCOMPLETE_END_MISSING = 'incomplete_end_missing';
    case OUTLIER_LONG_CYCLE = 'outlier_long_cycle';
    case OUTLIER_SHORT_CYCLE = 'outlier_short_cycle';
    case LONG_PERIOD = 'long_period_flag';
    case SHORT_PERIOD = 'short_period_flag';
    case IRREGULAR_CYCLE = 'irregular_cycle_flag';

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::INCOMPLETE_END_MISSING => 'پایان پریود ثبت نشده',
                self::OUTLIER_LONG_CYCLE => 'چرخهٔ بلندِ پرت',
                self::OUTLIER_SHORT_CYCLE => 'چرخهٔ کوتاهِ پرت',
                self::LONG_PERIOD => 'پریود طولانی',
                self::SHORT_PERIOD => 'پریود کوتاه',
                self::IRREGULAR_CYCLE => 'چرخهٔ نامنظم',
            },
            default => match ($this) {
                self::INCOMPLETE_END_MISSING => 'Period end missing',
                self::OUTLIER_LONG_CYCLE => 'Outlier long cycle',
                self::OUTLIER_SHORT_CYCLE => 'Outlier short cycle',
                self::LONG_PERIOD => 'Long period',
                self::SHORT_PERIOD => 'Short period',
                self::IRREGULAR_CYCLE => 'Irregular cycle',
            },
        };
    }

    /**
     * Whether a record carrying this flag should be held out of the prediction
     * medians (spec §8: outliers and incomplete cycles stay in history but do not
     * feed effective values).
     */
    public function excludesFromPrediction(): bool
    {
        return match ($this) {
            self::OUTLIER_LONG_CYCLE,
            self::OUTLIER_SHORT_CYCLE,
            self::INCOMPLETE_END_MISSING => true,
            default => false,
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
