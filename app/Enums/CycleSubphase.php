<?php

namespace App\Enums;

enum CycleSubphase: string
{
    case MENSTRUATION = 'menstruation';
    case EARLY_FOLLICULAR = 'early_follicular';
    case LATE_FOLLICULAR = 'late_follicular';
    case OVULATION_WINDOW = 'ovulation_window';
    case EARLY_LUTEAL = 'early_luteal';
    case MID_LUTEAL = 'mid_luteal';
    case LATE_LUTEAL = 'late_luteal';

    public function label(string $locale = 'en'): string
    {
        return match($locale) {
            'fa' => match($this) {
                self::MENSTRUATION => 'قاعدگی',
                self::EARLY_FOLLICULAR => 'فولیکولار اولیه',
                self::LATE_FOLLICULAR => 'فولیکولار پایانی',
                self::OVULATION_WINDOW => 'پنجره تخمک‌گذاری',
                self::EARLY_LUTEAL => 'لوتئال اولیه',
                self::MID_LUTEAL => 'لوتئال میانی',
                self::LATE_LUTEAL => 'لوتئال پایانی (PMS)',
            },
            default => match($this) {
                self::MENSTRUATION => 'Menstruation',
                self::EARLY_FOLLICULAR => 'Early Follicular',
                self::LATE_FOLLICULAR => 'Late Follicular',
                self::OVULATION_WINDOW => 'Ovulation Window',
                self::EARLY_LUTEAL => 'Early Luteal',
                self::MID_LUTEAL => 'Mid Luteal',
                self::LATE_LUTEAL => 'Late Luteal (PMS)',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
