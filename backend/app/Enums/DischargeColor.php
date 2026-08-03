<?php

namespace App\Enums;

/**
 * Vaginal discharge color (daily_health_logs.discharge_color).
 *
 * The column used to hold free text; the daily-log form now offers this closed
 * list so the values are comparable over time.
 */
enum DischargeColor: string
{
    case CLEAR = 'clear';
    case WHITE = 'white';
    case YELLOW = 'yellow';
    case GREEN = 'green';
    case GRAY = 'gray';
    case PINK_BLOODY = 'pink_bloody';

    public function label(string $locale = 'fa'): string
    {
        return $locale === 'fa'
            ? match ($this) {
                self::CLEAR => 'شفاف',
                self::WHITE => 'سفید',
                self::YELLOW => 'زرد',
                self::GREEN => 'سبز',
                self::GRAY => 'خاکستری',
                self::PINK_BLOODY => 'صورتی / خونی',
            }
        : match ($this) {
            self::CLEAR => 'Clear',
            self::WHITE => 'White',
            self::YELLOW => 'Yellow',
            self::GREEN => 'Green',
            self::GRAY => 'Gray',
            self::PINK_BLOODY => 'Pink / bloody',
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
