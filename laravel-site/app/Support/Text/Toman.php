<?php

declare(strict_types=1);

namespace App\Support\Text;

use App\Support\Money\Money;

/**
 * Display shorthand for integer TOMAN amounts (Blade components, directory prices stored in tomans). Delegates to
 * App\Support\Money\Money so every price on the site is formatted the same way.
 *
 *   Toman::format(485000)       →  ۴۸۵ هزار       (whole thousands read «… هزار», as in the design)
 *   Toman::format(12500)        →  ۱۲٬۵۰۰
 *   Toman::withUnit(320000)     →  ۳۲۰ هزار تومان
 *   Toman::format(485000, false) →  ۴۸۵٬۰۰۰
 */
final class Toman
{
    public const UNIT = Money::UNIT;

    public static function format(int $amount, bool $thousands = true): string
    {
        return Money::fromToman($amount)->formatAmount($thousands);
    }

    public static function withUnit(int $amount, bool $thousands = true): string
    {
        return self::format($amount, $thousands).' '.self::UNIT;
    }
}
