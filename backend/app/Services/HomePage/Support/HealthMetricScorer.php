<?php

namespace App\Services\HomePage\Support;

use App\Enums\EnergyLevel;
use App\Models\DailyHealthLog;
use Illuminate\Support\Collection;

/**
 * Normalizes self-reported wellbeing fields (mood / sleep / energy) into a
 * 0-100 score so the "خلاصه هفته" (weekly summary) and "وضعیت امروز" trend
 * charts can show comparable percentages.
 *
 * Every method returns null when the underlying data is absent, so callers can
 * distinguish "no data" from "score of zero".
 */
final class HealthMetricScorer
{
    private const MOOD_SCORES = [
        'happy' => 100,
        'calm' => 85,
        'sensitive' => 55,
        'bored' => 45,
        'frustrated' => 35,
        'anxious' => 30,
        'angry' => 25,
        'sad' => 20,
    ];

    private const SLEEP_QUALITY_SCORES = [
        'good' => 100,
        'medium' => 60,
        'bad' => 25,
    ];

    private const SLEEP_DURATION_SCORES = [
        '9_plus' => 85,
        '6_9' => 100,
        '3_6' => 55,
        '0_3' => 25,
    ];

    /**
     * Average wellbeing of the reported moods for a single day.
     */
    public static function moodScore(?array $moods): ?int
    {
        if (empty($moods)) {
            return null;
        }

        $scores = array_filter(
            array_map(fn ($m) => self::MOOD_SCORES[$m] ?? null, $moods),
            fn ($s) => $s !== null
        );

        if (empty($scores)) {
            return null;
        }

        return (int) round(array_sum($scores) / count($scores));
    }

    /**
     * Combined sleep score from quality and/or duration.
     */
    public static function sleepScore(?string $quality, ?string $duration): ?int
    {
        $parts = [];

        if ($quality !== null && isset(self::SLEEP_QUALITY_SCORES[$quality])) {
            $parts[] = self::SLEEP_QUALITY_SCORES[$quality];
        }
        if ($duration !== null && isset(self::SLEEP_DURATION_SCORES[$duration])) {
            $parts[] = self::SLEEP_DURATION_SCORES[$duration];
        }

        if (empty($parts)) {
            return null;
        }

        return (int) round(array_sum($parts) / count($parts));
    }

    /**
     * Energy score. Falls back to the inverse of the fatigue flag when no
     * explicit energy level was logged.
     */
    public static function energyScore(?string $energyLevel, ?bool $fatigue): ?int
    {
        if ($energyLevel !== null) {
            $enum = EnergyLevel::tryFrom($energyLevel);
            if ($enum) {
                return $enum->score();
            }
        }

        if ($fatigue !== null) {
            return $fatigue ? 30 : 70;
        }

        return null;
    }

    /**
     * Per-day scores for one log.
     *
     * @return array{mood: ?int, sleep: ?int, energy: ?int}
     */
    public static function scoreLog(DailyHealthLog $log): array
    {
        return [
            'mood' => self::moodScore($log->moods),
            'sleep' => self::sleepScore($log->sleep_quality, $log->sleep_duration),
            'energy' => self::energyScore($log->energy_level, $log->fatigue),
        ];
    }

    /**
     * Average each metric across a collection of logs, ignoring missing values.
     *
     * @param Collection<int, DailyHealthLog> $logs
     * @return array{mood: ?int, sleep: ?int, energy: ?int}
     */
    public static function weeklyAverages(Collection $logs): array
    {
        $buckets = ['mood' => [], 'sleep' => [], 'energy' => []];

        foreach ($logs as $log) {
            foreach (self::scoreLog($log) as $metric => $value) {
                if ($value !== null) {
                    $buckets[$metric][] = $value;
                }
            }
        }

        return [
            'mood' => self::averageOrNull($buckets['mood']),
            'sleep' => self::averageOrNull($buckets['sleep']),
            'energy' => self::averageOrNull($buckets['energy']),
        ];
    }

    private static function averageOrNull(array $values): ?int
    {
        if (empty($values)) {
            return null;
        }

        return (int) round(array_sum($values) / count($values));
    }
}
