<?php

namespace App\Enums;

use App\Models\DailyHealthLog;

/**
 * A symptom condition that switches an extra recommendation on for the day.
 *
 * The *text* of a recommendation is admin-editable content; deciding whether a
 * symptom is present is behaviour, so it stays in code here. A recommendation
 * with no trigger is purely phase-driven and always applies.
 */
enum RecommendationTrigger: string
{
    case HEADACHE = 'headache';
    case CRAMPS = 'cramps';
    case POOR_SLEEP = 'poor_sleep';
    case LOW_MOOD = 'low_mood';
    case BLOATING = 'bloating';
    case FATIGUE = 'fatigue';

    public function label(string $locale = 'fa'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::HEADACHE => 'سردرد',
                self::CRAMPS => 'درد و کرامپ',
                self::POOR_SLEEP => 'کیفیت خواب بد',
                self::LOW_MOOD => 'اضطراب یا غم',
                self::BLOATING => 'نفخ',
                self::FATIGUE => 'خستگی',
            },
            default => match ($this) {
                self::HEADACHE => 'Headache',
                self::CRAMPS => 'Cramps',
                self::POOR_SLEEP => 'Poor sleep',
                self::LOW_MOOD => 'Anxious or sad mood',
                self::BLOATING => 'Bloating',
                self::FATIGUE => 'Fatigue',
            },
        };
    }

    /** Whether today's log reports this symptom. No log means no symptom. */
    public function matches(?DailyHealthLog $log): bool
    {
        if (! $log) {
            return false;
        }

        return match ($this) {
            self::HEADACHE => $log->headache_intensity !== null,
            self::CRAMPS => $log->pelvic_pain_intensity !== null || $log->stomach_ache_intensity !== null,
            self::POOR_SLEEP => $log->sleep_quality === SleepQuality::BAD->value,
            self::LOW_MOOD => is_array($log->moods) && array_intersect(
                [Mood::ANXIOUS->value, Mood::SAD->value],
                $log->moods,
            ) !== [],
            self::BLOATING => $log->bloating_intensity !== null,
            self::FATIGUE => $log->fatigue === true,
        };
    }

    /** @return array<int, string> */
    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }

    /**
     * Options for the admin trigger picker.
     *
     * @return array<int, array{value: string, label: string}>
     */
    public static function options(string $locale = 'fa'): array
    {
        return array_map(
            fn (self $case): array => ['value' => $case->value, 'label' => $case->label($locale)],
            self::cases(),
        );
    }

    /**
     * The triggers today's log satisfies — the set a day's recommendations may
     * be keyed to.
     *
     * @return array<int, string>
     */
    public static function activeFor(?DailyHealthLog $log): array
    {
        return array_values(array_map(
            fn (self $case): string => $case->value,
            array_filter(self::cases(), fn (self $case): bool => $case->matches($log)),
        ));
    }

    public static function labelFor(?string $value, string $locale = 'fa'): ?string
    {
        return $value === null ? null : self::tryFrom($value)?->label($locale);
    }
}
