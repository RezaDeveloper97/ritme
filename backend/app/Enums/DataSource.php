<?php

namespace App\Enums;

/**
 * Origin of a cycle datum (spec §15). isActual() separates real, user-owned period
 * facts — which feed effective values and the "actual" data status — from estimates
 * and engine predictions, so the two are never conflated.
 */
enum DataSource: string
{
    case USER_LOGGED = 'user_logged';
    case USER_PROFILE_CONFIRMED = 'user_profile_confirmed';
    case ONBOARDING_ESTIMATE = 'onboarding_estimate';
    case ONBOARDING_INITIAL = 'onboarding_initial';
    case SYSTEM_DEFAULT = 'system_default';
    case ENGINE_PREDICTION = 'engine_prediction';

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::USER_LOGGED => 'ثبت‌شده توسط کاربر',
                self::USER_PROFILE_CONFIRMED => 'تأییدشده در پروفایل',
                self::ONBOARDING_ESTIMATE => 'تخمین اولیهٔ ثبت‌نام',
                self::ONBOARDING_INITIAL => 'مقدار اولیهٔ ثبت‌نام',
                self::SYSTEM_DEFAULT => 'پیش‌فرض سیستم',
                self::ENGINE_PREDICTION => 'پیش‌بینی موتور',
            },
            default => match ($this) {
                self::USER_LOGGED => 'User logged',
                self::USER_PROFILE_CONFIRMED => 'Profile confirmed',
                self::ONBOARDING_ESTIMATE => 'Onboarding estimate',
                self::ONBOARDING_INITIAL => 'Onboarding initial',
                self::SYSTEM_DEFAULT => 'System default',
                self::ENGINE_PREDICTION => 'Engine prediction',
            },
        };
    }

    /**
     * A real, user-owned period fact (counts toward effective values and an "actual"
     * data status) as opposed to an estimate or a prediction.
     */
    public function isActual(): bool
    {
        return $this === self::USER_LOGGED || $this === self::USER_PROFILE_CONFIRMED;
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
