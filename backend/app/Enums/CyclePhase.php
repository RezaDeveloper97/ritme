<?php

namespace App\Enums;

enum CyclePhase: string
{
    case MENSTRUATION = 'menstruation';
    case FOLLICULAR = 'follicular';
    case OVULATION = 'ovulation';
    case LUTEAL = 'luteal';

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::MENSTRUATION => 'قاعدگی',
                self::FOLLICULAR => 'فولیکولار',
                self::OVULATION => 'تخمک‌گذاری',
                self::LUTEAL => 'لوتئال',
            },
            default => match ($this) {
                self::MENSTRUATION => 'Menstruation',
                self::FOLLICULAR => 'Follicular',
                self::OVULATION => 'Ovulation',
                self::LUTEAL => 'Luteal',
            },
        };
    }

    public function description(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::MENSTRUATION => 'دوران خون‌ریزی ماهانه',
                self::FOLLICULAR => 'دوران رشد فولیکول و آماده‌سازی تخمک',
                self::OVULATION => 'دوران آزادسازی تخمک',
                self::LUTEAL => 'دوران پس از تخمک‌گذاری',
            },
            default => match ($this) {
                self::MENSTRUATION => 'Monthly bleeding period',
                self::FOLLICULAR => 'Follicle growth and egg preparation phase',
                self::OVULATION => 'Egg release period',
                self::LUTEAL => 'Post-ovulation phase',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }

    /**
     * Options for the admin phase picker (see admin.partials.phase-select).
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
     * Human label for a stored phase key, or null when the key is unknown.
     */
    public static function labelFor(?string $value, string $locale = 'fa'): ?string
    {
        return $value === null ? null : self::tryFrom($value)?->label($locale);
    }
}
