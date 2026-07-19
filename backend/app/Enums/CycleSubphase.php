<?php

namespace App\Enums;

/**
 * Fine-grained cycle sub-phase (spec §16–17). Twelve states walk the cycle from
 * menstruation through the fertile ramp, ovulation and the luteal wind-down to a
 * possible-PMS and finally an expected-period marker. Positioned relative to the
 * estimated ovulation day, so they stay correct for non-28-day cycles.
 */
enum CycleSubphase: string
{
    case MENSTRUATION = 'menstruation';
    /** v1.1 (task.md §14) spelling of the menstrual sub-phase; MENSTRUATION stays for legacy payloads. */
    case MENSTRUAL = 'menstrual';
    /** Bleed may still be ongoing but the logged end is missing past its expected length (task.md §21.3–21.4). */
    case MENSTRUAL_POSSIBLE = 'menstrual_possible';
    case EARLY_FOLLICULAR = 'early_follicular';
    case MID_FOLLICULAR = 'mid_follicular';
    case LATE_FOLLICULAR_TRANSITION = 'late_follicular_transition';
    case FERTILE_RISING = 'fertile_rising';
    case HIGH_FERTILITY = 'high_fertility';
    case OVULATION_LIKELY = 'ovulation_likely';
    case POST_OVULATION = 'post_ovulation';
    case EARLY_LUTEAL = 'early_luteal';
    case MID_LUTEAL = 'mid_luteal';
    case LATE_LUTEAL = 'late_luteal';
    case PMS_POSSIBLE = 'pms_possible';
    case PERIOD_EXPECTED = 'period_expected';
    case UNKNOWN = 'unknown';

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::MENSTRUATION, self::MENSTRUAL => 'قاعدگی',
                self::MENSTRUAL_POSSIBLE => 'احتمال ادامه قاعدگی',
                self::EARLY_FOLLICULAR => 'فولیکولار اولیه',
                self::MID_FOLLICULAR => 'فولیکولار میانی',
                self::LATE_FOLLICULAR_TRANSITION => 'گذار فولیکولار پایانی',
                self::FERTILE_RISING => 'افزایش باروری',
                self::HIGH_FERTILITY => 'باروری بالا',
                self::OVULATION_LIKELY => 'احتمال تخمک‌گذاری',
                self::POST_OVULATION => 'پس از تخمک‌گذاری',
                self::EARLY_LUTEAL => 'لوتئال اولیه',
                self::MID_LUTEAL => 'لوتئال میانی',
                self::LATE_LUTEAL => 'لوتئال پایانی',
                self::PMS_POSSIBLE => 'احتمال پی‌ام‌اس',
                self::PERIOD_EXPECTED => 'انتظار پریود',
                self::UNKNOWN => 'نامشخص',
            },
            default => match ($this) {
                self::MENSTRUATION, self::MENSTRUAL => 'Menstruation',
                self::MENSTRUAL_POSSIBLE => 'Possibly Still Menstruating',
                self::EARLY_FOLLICULAR => 'Early Follicular',
                self::MID_FOLLICULAR => 'Mid Follicular',
                self::LATE_FOLLICULAR_TRANSITION => 'Late Follicular Transition',
                self::FERTILE_RISING => 'Rising Fertility',
                self::HIGH_FERTILITY => 'High Fertility',
                self::OVULATION_LIKELY => 'Ovulation Likely',
                self::POST_OVULATION => 'Post-Ovulation',
                self::EARLY_LUTEAL => 'Early Luteal',
                self::MID_LUTEAL => 'Mid Luteal',
                self::LATE_LUTEAL => 'Late Luteal',
                self::PMS_POSSIBLE => 'Possible PMS',
                self::PERIOD_EXPECTED => 'Period Expected',
                self::UNKNOWN => 'Unknown',
            },
        };
    }

    /**
     * v1.1 fertility level for this sub-phase (task.md §26). Distinct from the
     * legacy {@see fertilityLevel()} scale: peaks at ovulation, medium only on
     * the fertile ramp-up and late-follicular transition, and unknown whenever
     * the cycle itself is unresolved.
     */
    public function fertilityLevelV11(): FertilityLevel
    {
        return match ($this) {
            self::OVULATION_LIKELY => FertilityLevel::PEAK,
            self::HIGH_FERTILITY => FertilityLevel::HIGH,
            self::FERTILE_RISING, self::LATE_FOLLICULAR_TRANSITION => FertilityLevel::MEDIUM,
            self::PERIOD_EXPECTED, self::UNKNOWN => FertilityLevel::UNKNOWN,
            default => FertilityLevel::LOW,
        };
    }

    /**
     * Calendar-based fertility level for this sub-phase (spec §17 mapping table).
     * A cycle-timing estimate only — never a diagnosis.
     */
    public function fertilityLevel(): FertilityLevel
    {
        return match ($this) {
            self::OVULATION_LIKELY => FertilityLevel::VERY_HIGH,
            self::HIGH_FERTILITY => FertilityLevel::HIGH,
            self::FERTILE_RISING, self::POST_OVULATION => FertilityLevel::MEDIUM,
            default => FertilityLevel::LOW,
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }

    /**
     * The sub-phases that have their own seeded educational content. The v1.1
     * additions (menstrual, menstrual_possible, late_follicular_transition,
     * unknown) are aliased onto these by the phase-content endpoint.
     *
     * @return array<self>
     */
    public static function contentBacked(): array
    {
        return array_values(array_filter(
            self::cases(),
            fn (self $case) => ! in_array($case, [
                self::MENSTRUAL,
                self::MENSTRUAL_POSSIBLE,
                self::LATE_FOLLICULAR_TRANSITION,
                self::UNKNOWN,
            ], true),
        ));
    }
}
