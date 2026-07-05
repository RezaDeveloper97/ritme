<?php

namespace App\Enums;

enum CalculationStatus: string
{
    case PENDING = 'pending';
    case PROCESSING = 'processing';
    case COMPLETED = 'completed';
    case FAILED = 'failed';

    public function label(string $locale = 'en'): string
    {
        return match($locale) {
            'fa' => match($this) {
                self::PENDING => 'در انتظار',
                self::PROCESSING => 'در حال پردازش',
                self::COMPLETED => 'محاسبه شده',
                self::FAILED => 'خطا',
            },
            default => match($this) {
                self::PENDING => 'Pending',
                self::PROCESSING => 'Processing',
                self::COMPLETED => 'Completed',
                self::FAILED => 'Failed',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
