<?php

namespace App\Enums;

/**
 * Cycle regularity as shown to the user (spec §11) — an MVP range test over recent
 * valid cycle lengths, deliberately simpler than the std-dev {@see CycleVariability}
 * used for prediction uncertainty. Never presented as a medical diagnosis.
 */
enum RegularityStatus: string
{
    case NOT_ENOUGH_DATA = 'not_enough_data';
    case RELATIVELY_REGULAR = 'relatively_regular';
    case IRREGULAR_POSSIBLE = 'irregular_possible';

    /** Spread (max-min) in days within which recent cycles read as relatively regular. */
    private const REGULAR_RANGE_DAYS = 7;

    /** Minimum valid cycles before regularity can be judged at all. */
    private const MIN_CYCLES = 3;

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::NOT_ENOUGH_DATA => 'دادهٔ کافی نیست',
                self::RELATIVELY_REGULAR => 'نسبتاً منظم',
                self::IRREGULAR_POSSIBLE => 'احتمال نامنظمی',
            },
            default => match ($this) {
                self::NOT_ENOUGH_DATA => 'Not enough data',
                self::RELATIVELY_REGULAR => 'Relatively regular',
                self::IRREGULAR_POSSIBLE => 'Possibly irregular',
            },
        };
    }

    /**
     * Classify from recent valid cycle lengths (spec §11): fewer than three valid
     * cycles is unknown; otherwise a spread of ≤7 days reads as relatively regular.
     *
     * @param  array<int>  $lengths
     */
    public static function fromCycleLengths(array $lengths): self
    {
        if (count($lengths) < self::MIN_CYCLES) {
            return self::NOT_ENOUGH_DATA;
        }

        $range = max($lengths) - min($lengths);

        return $range <= self::REGULAR_RANGE_DAYS
            ? self::RELATIVELY_REGULAR
            : self::IRREGULAR_POSSIBLE;
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
