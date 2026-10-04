<?php

declare(strict_types=1);

namespace App\Domain\Directory\Booking\Support;

use App\Support\Text\PersianDigits;

/**
 * A mobile number as the booked page shows it: only the first four and the last two digits («۰۹۱۲•••••۶۷»), so a
 * shared or leaked booked link does not reveal the parent's number.
 */
final class MobileMask
{
    public static function mask(string $mobile): string
    {
        $digits = preg_replace('/\D/', '', $mobile) ?? '';
        if (strlen($digits) < 7) {
            return PersianDigits::toPersian(str_repeat('•', strlen($digits)));
        }

        return PersianDigits::toPersian(substr($digits, 0, 4).str_repeat('•', strlen($digits) - 6).substr($digits, -2));
    }
}
