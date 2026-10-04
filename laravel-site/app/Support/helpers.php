<?php

declare(strict_types=1);

use App\Support\Jalali\JalaliDate;
use App\Support\Text\PersianDigits;

/*
 * Global view helpers (composer `autoload.files`). Thin wrappers — the logic lives in App\Support\Text / Jalali.
 */

if (! function_exists('fa_digits')) {
    /** Latin / Arabic-Indic digits → Persian digits («2026» → «۲۰۲۶»). Null renders as an empty string. */
    function fa_digits(string|int|float|null $value): string
    {
        return PersianDigits::toPersian($value);
    }
}

if (! function_exists('jdate')) {
    /**
     * Jalali display string in Tehran time with Persian digits: `jdate($post->published_at)` → «۱۲ مهر ۱۴۰۵».
     * `$date` may be a DateTime, a Unix timestamp, a date string or null (now). See JalaliDate::format() for tokens.
     */
    function jdate(DateTimeInterface|int|string|null $date = null, string $format = JalaliDate::DEFAULT_FORMAT, bool $persianDigits = true): string
    {
        $jalali = $date === null ? JalaliDate::now() : JalaliDate::parse($date);

        return $jalali->format($format, $persianDigits);
    }
}
