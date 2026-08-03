<?php

namespace App\Enums;

/**
 * Category of a "توصیه‌های امروز" daily recommendation.
 *
 * The single source of truth for the category list: the admin form builds its
 * dropdown from {@see options()}, the home-page section takes its icon and
 * default title from here, and the cycle-calculation payload sends the resolved
 * label to the clients — so adding a case below lights it up everywhere.
 */
enum RecommendationType: string
{
    case NUTRITION = 'nutrition';
    case HYDRATION = 'hydration';
    case WARMTH = 'warmth';
    case REST = 'rest';
    case ENERGY = 'energy';
    case EXERCISE = 'exercise';
    case FERTILITY = 'fertility';
    case PMS = 'pms';
    case MOOD = 'mood';
    case MENTAL_HEALTH = 'mental_health';
    case PAIN_RELIEF = 'pain_relief';
    case SLEEP = 'sleep';
    case DIGESTION = 'digestion';
    case GENERAL = 'general';

    public function label(string $locale = 'fa'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::NUTRITION => 'تغذیه',
                self::HYDRATION => 'آب‌رسانی',
                self::WARMTH => 'گرما درمانی',
                self::REST => 'استراحت',
                self::ENERGY => 'انرژی',
                self::EXERCISE => 'ورزش',
                self::FERTILITY => 'باروری',
                self::PMS => 'پی‌ام‌اس',
                self::MOOD => 'خلق‌وخو',
                self::MENTAL_HEALTH => 'سلامت روان',
                self::PAIN_RELIEF => 'تسکین درد',
                self::SLEEP => 'خواب',
                self::DIGESTION => 'گوارش',
                self::GENERAL => 'توصیه',
            },
            default => match ($this) {
                self::NUTRITION => 'Nutrition',
                self::HYDRATION => 'Hydration',
                self::WARMTH => 'Warmth',
                self::REST => 'Rest',
                self::ENERGY => 'Energy',
                self::EXERCISE => 'Exercise',
                self::FERTILITY => 'Fertility',
                self::PMS => 'PMS',
                self::MOOD => 'Mood',
                self::MENTAL_HEALTH => 'Mental health',
                self::PAIN_RELIEF => 'Pain relief',
                self::SLEEP => 'Sleep',
                self::DIGESTION => 'Digestion',
                self::GENERAL => 'Tip',
            },
        };
    }

    /** Icon key the home-page section sends to the clients. */
    public function icon(): string
    {
        return match ($this) {
            self::NUTRITION => 'apple',
            self::HYDRATION => 'water-glass',
            self::WARMTH => 'heat',
            self::REST => 'bed',
            self::ENERGY => 'bolt',
            self::EXERCISE => 'walking',
            self::FERTILITY => 'heart',
            self::PMS => 'flower',
            self::MOOD => 'smile',
            self::MENTAL_HEALTH => 'brain',
            self::PAIN_RELIEF => 'bandage',
            self::SLEEP => 'moon',
            self::DIGESTION => 'stomach',
            self::GENERAL => 'sparkle',
        };
    }

    /** @return array<int, string> */
    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }

    /**
     * Options for the admin category picker.
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

    /** Human label for a stored key, falling back to the generic "tip" label. */
    public static function labelFor(?string $value, string $locale = 'fa'): string
    {
        return (self::tryFrom((string) $value) ?? self::GENERAL)->label($locale);
    }

    /** Icon for a stored key, falling back to the generic sparkle. */
    public static function iconFor(?string $value): string
    {
        return (self::tryFrom((string) $value) ?? self::GENERAL)->icon();
    }
}
