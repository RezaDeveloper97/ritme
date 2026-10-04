<?php

declare(strict_types=1);

namespace App\Domain\Directory\Booking\Support;

use App\Support\Text\PersianDigits;

/**
 * The child's age as entered (a number + «ماه»/«سال») ⇄ the months stored on the request, and its Persian label
 * («۱۴ ماه», «۳ سال», «۲ سال و ۵ ماه»). Up to MAX_MONTHS (the directory is for children up to 18).
 */
final class ChildAge
{
    public const MAX_MONTHS = 216;

    public static function toMonths(int $value, string $unit): int
    {
        return $unit === 'year' ? $value * 12 : $value;
    }

    public static function label(int $months): string
    {
        if ($months < 24) {
            return PersianDigits::toPersian($months).' ماه';
        }
        $years = intdiv($months, 12);
        $rest = $months % 12;

        return PersianDigits::toPersian($years).' سال'.($rest > 0 ? ' و '.PersianDigits::toPersian($rest).' ماه' : '');
    }
}
