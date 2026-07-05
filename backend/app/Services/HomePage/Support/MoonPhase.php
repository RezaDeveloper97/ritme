<?php

namespace App\Services\HomePage\Support;

use Carbon\Carbon;

/**
 * Computes the lunar phase for a date (header "ماه کامل" indicator).
 *
 * Uses the mean synodic month from a known new-moon epoch; accurate to within a
 * day, which is plenty for a decorative header label.
 */
final class MoonPhase
{
    /** Reference new moon: 2000-01-06 18:14 UTC. */
    private const NEW_MOON_EPOCH = 947182440;

    private const SYNODIC_MONTH = 29.530588853;

    private const PHASES = [
        'new_moon' => ['fa' => 'ماه نو', 'en' => 'New moon'],
        'waxing_crescent' => ['fa' => 'هلال افزاینده', 'en' => 'Waxing crescent'],
        'first_quarter' => ['fa' => 'تربیع اول', 'en' => 'First quarter'],
        'waxing_gibbous' => ['fa' => 'محدب افزاینده', 'en' => 'Waxing gibbous'],
        'full_moon' => ['fa' => 'ماه کامل', 'en' => 'Full moon'],
        'waning_gibbous' => ['fa' => 'محدب کاهنده', 'en' => 'Waning gibbous'],
        'last_quarter' => ['fa' => 'تربیع آخر', 'en' => 'Last quarter'],
        'waning_crescent' => ['fa' => 'هلال کاهنده', 'en' => 'Waning crescent'],
    ];

    /**
     * @return array{key: string, label: string, illumination: float, age_days: float}
     */
    public static function forDate(Carbon $date, string $locale = 'fa'): array
    {
        $ageDays = self::ageInDays($date);
        $fraction = $ageDays / self::SYNODIC_MONTH;          // 0..1 through the cycle

        // 8 named phases; offset by half a segment so each label is centered.
        $index = (int) floor(($fraction * 8) + 0.5) % 8;
        $key = array_keys(self::PHASES)[$index];

        // Illuminated fraction (0 = new, 1 = full).
        $illumination = (1 - cos(2 * M_PI * $fraction)) / 2;

        return [
            'key' => $key,
            'label' => self::PHASES[$key][$locale] ?? self::PHASES[$key]['en'],
            'illumination' => round($illumination, 2),
            'age_days' => round($ageDays, 1),
        ];
    }

    private static function ageInDays(Carbon $date): float
    {
        $secondsSinceEpoch = $date->copy()->utc()->timestamp - self::NEW_MOON_EPOCH;
        $days = $secondsSinceEpoch / 86400;
        $age = fmod($days, self::SYNODIC_MONTH);

        return $age < 0 ? $age + self::SYNODIC_MONTH : $age;
    }
}
