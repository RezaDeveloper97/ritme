<?php

namespace App\Enums;

enum AlertLevel: string
{
    case INFO = 'info';
    case WARNING = 'warning';
    case EMERGENCY = 'emergency';

    public function label(string $locale = 'en'): string
    {
        return match($locale) {
            'fa' => match($this) {
                self::INFO => 'اطلاع‌رسانی',
                self::WARNING => 'هشدار',
                self::EMERGENCY => 'اورژانسی',
            },
            default => match($this) {
                self::INFO => 'Information',
                self::WARNING => 'Warning',
                self::EMERGENCY => 'Emergency',
            },
        };
    }

    public function description(string $locale = 'en'): string
    {
        return match($locale) {
            'fa' => match($this) {
                self::INFO => 'اطلاعات عمومی و راهنمایی',
                self::WARNING => 'نیاز به توجه و احتیاط',
                self::EMERGENCY => 'مراجعه فوری به پزشک',
            },
            default => match($this) {
                self::INFO => 'General information and guidance',
                self::WARNING => 'Requires attention and caution',
                self::EMERGENCY => 'Immediate medical attention required',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
