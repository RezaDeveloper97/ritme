<?php

namespace App\Enums;

/**
 * How the user relates to pregnancy at onboarding time. Captured for women-only
 * users to branch the signup flow (pregnant → pregnancy mode, others → cycle
 * questions) and to seed the cycle engine's user_goal.
 */
enum PregnancyIntention: string
{
    case AVOIDING = 'avoiding';   // فعلاً قصد بارداری ندارم
    case PREGNANT = 'pregnant';   // باردار هستم
    case TRYING = 'trying';       // در حال تلاش برای بارداری
    case UNSURE = 'unsure';       // مطمئن نیستم / هنوز تصمیم نگرفته‌ام

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::AVOIDING => 'فعلاً قصد بارداری ندارم',
                self::PREGNANT => 'باردار هستم',
                self::TRYING => 'در حال تلاش برای بارداری هستم',
                self::UNSURE => 'مطمئن نیستم / هنوز تصمیم نگرفته‌ام',
            },
            default => match ($this) {
                self::AVOIDING => 'Not planning a pregnancy right now',
                self::PREGNANT => 'I am pregnant',
                self::TRYING => 'Trying to conceive',
                self::UNSURE => 'Not sure yet',
            },
        };
    }
}
