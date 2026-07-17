<?php

namespace App\Enums;

/**
 * How a day's cycle information was derived — drives the daily card's framing and,
 * when more than one could apply to the same day, which one wins (see priority()).
 * Spec §18: actual > incomplete > needs_confirmation > predicted.
 */
enum DataStatus: string
{
    case ACTUAL = 'actual';                          // inside a real, logged period
    case INCOMPLETE = 'incomplete';                  // real data started but unfinished (End missing)
    case NEEDS_CONFIRMATION = 'needs_confirmation';  // a predicted event is due, waiting on the user
    case PREDICTED = 'predicted';                    // pure engine prediction

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::ACTUAL => 'ثبت‌شده',
                self::INCOMPLETE => 'اطلاعات ناقص',
                self::NEEDS_CONFIRMATION => 'نیاز به تأیید',
                self::PREDICTED => 'پیش‌بینی‌شده',
            },
            default => match ($this) {
                self::ACTUAL => 'Logged',
                self::INCOMPLETE => 'Incomplete',
                self::NEEDS_CONFIRMATION => 'Needs confirmation',
                self::PREDICTED => 'Predicted',
            },
        };
    }

    /**
     * Display priority when several statuses could describe one day; the highest wins
     * so a real logged period is never overridden by a prediction (spec §18).
     */
    public function priority(): int
    {
        return match ($this) {
            self::ACTUAL => 4,
            self::INCOMPLETE => 3,
            self::NEEDS_CONFIRMATION => 2,
            self::PREDICTED => 1,
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
