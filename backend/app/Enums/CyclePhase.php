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
     * The sub-phases {@see App\Services\HealthEngine\CyclePhaseMapper} can emit
     * inside this phase, mirroring the boundaries it draws (§16–17): the fertile
     * ramp still sits before ovulation day O, so it belongs to the follicular
     * phase, and the ovulation phase spans only O and O+1.
     *
     * Two canonical sub-phases are deliberately absent, because content narrowed
     * to them would never be shown: the v1.1 aliases collapse onto the keys
     * listed here via {@see CycleSubphase::canonical()}, and `period_expected` is
     * never returned by `subphaseFor()` at all — the daily card decides it
     * separately when a predicted period is due with no Start logged.
     *
     * Used by the admin forms to offer, and accept, only pairings a real day can
     * actually produce.
     *
     * @return array<int, CycleSubphase>
     */
    public function subphases(): array
    {
        return match ($this) {
            self::MENSTRUATION => [CycleSubphase::MENSTRUATION],
            self::FOLLICULAR => [
                CycleSubphase::EARLY_FOLLICULAR,
                CycleSubphase::MID_FOLLICULAR,
                CycleSubphase::FERTILE_RISING,
                CycleSubphase::HIGH_FERTILITY,
            ],
            self::OVULATION => [
                CycleSubphase::OVULATION_LIKELY,
                CycleSubphase::POST_OVULATION,
            ],
            self::LUTEAL => [
                CycleSubphase::EARLY_LUTEAL,
                CycleSubphase::MID_LUTEAL,
                CycleSubphase::LATE_LUTEAL,
                CycleSubphase::PMS_POSSIBLE,
            ],
        };
    }

    /**
     * Every sub-phase a calculated day can report, across all phases.
     *
     * @return array<int, CycleSubphase>
     */
    public static function allSubphases(): array
    {
        return array_merge(...array_map(fn (self $phase): array => $phase->subphases(), self::cases()));
    }

    /**
     * Sub-phase keys valid for this phase, or every emittable key when no phase
     * is given (a phase-agnostic row may target any of them).
     *
     * @return array<int, string>
     */
    public static function subphaseValuesFor(?string $phase): array
    {
        $cases = $phase === null
            ? self::allSubphases()
            : (self::tryFrom($phase)?->subphases() ?? self::allSubphases());

        return array_map(fn (CycleSubphase $case): string => $case->value, $cases);
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
