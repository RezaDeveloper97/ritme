<?php

declare(strict_types=1);

namespace App\Support\Text;

/**
 * Minimal Toman display formatter for integer amounts (the Money value object arrives with L6-01).
 *
 *   Toman::format(485000)       →  ۴۸۵ هزار       (whole thousands read «… هزار», as in the design)
 *   Toman::format(12500)        →  ۱۲٬۵۰۰
 *   Toman::withUnit(320000)     →  ۳۲۰ هزار تومان
 *   Toman::format(485000, false) →  ۴۸۵٬۰۰۰
 */
final class Toman
{
    public const UNIT = 'تومان';

    public static function format(int $amount, bool $thousands = true): string
    {
        if ($thousands && $amount >= 1000 && $amount % 1000 === 0) {
            return PersianDigits::number(intdiv($amount, 1000)).' هزار';
        }

        return PersianDigits::number($amount);
    }

    public static function withUnit(int $amount, bool $thousands = true): string
    {
        return self::format($amount, $thousands).' '.self::UNIT;
    }
}
