<?php

declare(strict_types=1);

namespace App\Domain\Content\Tools;

use App\Domain\Content\Tools\Data\JalaliDay;
use App\Support\Jalali\JalaliCalendar;
use App\Support\Jalali\JalaliDate;
use App\Support\Text\PersianDigits;

/**
 * Reads what people type into the calculator fields. Twin of `parseDate()` / `parseCycle()` in
 * resources/js/lib/jalali.js — change both together (tests/js/fixtures/calculator-vectors.json covers both).
 *
 * Dates: «۱۴۰۵/۰۲/۱۲», «1405-2-12», «1405.2.12» or «۱۲ اردیبهشت ۱۴۰۵» (Persian, Arabic-Indic or Latin digits; Arabic
 * «ي/ك» accepted), Jalali years 1300–1499, real calendar days only (no ۳۰ اسفند in a common year).
 * Cycle: the digits in the text («۲۸ روز» → 28); empty → 28.
 */
final class CalculatorInput
{
    public const DEFAULT_CYCLE = 28;

    public const MIN_CYCLE = 21;

    public const MAX_CYCLE = 45;

    public const MIN_YEAR = 1300;

    public const MAX_YEAR = 1499;

    public static function parseDate(?string $text): ?JalaliDay
    {
        $text = self::normalise($text);

        if (preg_match('~^(\d{4})[/.\-](\d{1,2})[/.\-](\d{1,2})$~', $text, $m) === 1) {
            return self::day((int) $m[1], (int) $m[2], (int) $m[3]);
        }

        if (preg_match('~^(\d{1,2}) (\S+) (\d{4})$~u', $text, $m) === 1) {
            $month = array_search($m[2], JalaliDate::MONTHS, true);

            return is_int($month) ? self::day((int) $m[3], $month, (int) $m[1]) : null;
        }

        return null;
    }

    /** The cycle length in days, or null when it is outside 21–45 (or absurdly long text). */
    public static function parseCycle(?string $text): ?int
    {
        $digits = preg_replace('~\D~', '', PersianDigits::toLatin($text)) ?? '';
        if ($digits === '') {
            return self::DEFAULT_CYCLE;
        }

        $days = strlen($digits) > 3 ? 0 : (int) $digits;

        return $days >= self::MIN_CYCLE && $days <= self::MAX_CYCLE ? $days : null;
    }

    private static function day(int $year, int $month, int $day): ?JalaliDay
    {
        if ($year < self::MIN_YEAR || $year > self::MAX_YEAR || ! JalaliCalendar::isValid($year, $month, $day)) {
            return null;
        }

        return JalaliDay::of($year, $month, $day);
    }

    private static function normalise(?string $text): string
    {
        $text = strtr(PersianDigits::toLatin($text), ['ي' => 'ی', 'ك' => 'ک', "\u{200C}" => '']);

        return trim(preg_replace('~\s+~u', ' ', $text) ?? '');
    }
}
