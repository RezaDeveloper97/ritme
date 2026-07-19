<?php

namespace App\Enums;

enum ConfidenceLevel: string
{
    case HIGH = 'high';
    case MEDIUM = 'medium';
    case LOW = 'low';
    /** v1.1 (task.md §29.2): no resolvable anchor at all — below even "low". */
    case UNKNOWN = 'unknown';

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::HIGH => 'بالا',
                self::MEDIUM => 'متوسط',
                self::LOW => 'پایین',
                self::UNKNOWN => 'نامشخص',
            },
            default => match ($this) {
                self::HIGH => 'High',
                self::MEDIUM => 'Medium',
                self::LOW => 'Low',
                self::UNKNOWN => 'Unknown',
            },
        };
    }

    public function description(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::HIGH => 'مبتنی بر سونوگرافی',
                self::MEDIUM => 'مبتنی بر LMP دقیق',
                self::LOW => 'ورود نامطمئن',
                self::UNKNOWN => 'داده کافی وجود ندارد',
            },
            default => match ($this) {
                self::HIGH => 'Based on ultrasound',
                self::MEDIUM => 'Based on accurate LMP',
                self::LOW => 'Uncertain entry',
                self::UNKNOWN => 'Not enough data',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
