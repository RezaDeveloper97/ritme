<?php

namespace App\Enums;

enum PregnancyAgeSource: string
{
    case LMP = 'lmp';
    case ULTRASOUND = 'ultrasound';
    case MANUAL = 'manual';

    public function label(string $locale = 'en'): string
    {
        return match($locale) {
            'fa' => match($this) {
                self::LMP => 'تاریخ آخرین پریود',
                self::ULTRASOUND => 'سونوگرافی',
                self::MANUAL => 'ورود دستی',
            },
            default => match($this) {
                self::LMP => 'Last Menstrual Period',
                self::ULTRASOUND => 'Ultrasound',
                self::MANUAL => 'Manual Entry',
            },
        };
    }

    public function description(string $locale = 'en'): string
    {
        return match($locale) {
            'fa' => match($this) {
                self::LMP => 'محاسبه بر اساس اولین روز آخرین پریود',
                self::ULTRASOUND => 'محاسبه بر اساس اطلاعات سونوگرافی',
                self::MANUAL => 'ورود مستقیم هفته و روز بارداری',
            },
            default => match($this) {
                self::LMP => 'Calculation based on first day of last menstrual period',
                self::ULTRASOUND => 'Calculation based on ultrasound information',
                self::MANUAL => 'Direct entry of pregnancy week and day',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
