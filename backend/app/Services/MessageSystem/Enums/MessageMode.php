<?php

namespace App\Services\MessageSystem\Enums;

enum MessageMode: string
{
    case CYCLE = 'cycle';
    case PREGNANCY = 'pregnancy';
    case POSTPARTUM = 'postpartum';

    public function label(string $locale = 'en'): string
    {
        return match ($this) {
            self::CYCLE => $locale === 'fa' ? 'سیکل قاعدگی' : 'Menstrual Cycle',
            self::PREGNANCY => $locale === 'fa' ? 'بارداری' : 'Pregnancy',
            self::POSTPARTUM => $locale === 'fa' ? 'پس از زایمان' : 'Postpartum',
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
