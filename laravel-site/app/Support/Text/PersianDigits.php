<?php

declare(strict_types=1);

namespace App\Support\Text;

/**
 * Persian digits and number separators (AUDIT §2.2): «۰–۹», thousands «٬» (U+066C), decimal «٫» (U+066B).
 *
 *   PersianDigits::toPersian('1403/01/01')  →  ۱۴۰۳/۰۱/۰۱      (Arabic-Indic «٤٥٦» are normalised too)
 *   PersianDigits::toLatin('۰۹۱۲ ٣٤٥')      →  0912 345        (for parsing user input)
 *   PersianDigits::number(1250)             →  ۱٬۲۵۰
 *   PersianDigits::number(4.80, 1)          →  ۴٫۸             (trailing fractional zeros are dropped)
 */
final class PersianDigits
{
    public const THOUSANDS_SEPARATOR = '٬';

    public const DECIMAL_SEPARATOR = '٫';

    private const TO_PERSIAN = [
        '0' => '۰', '1' => '۱', '2' => '۲', '3' => '۳', '4' => '۴', '5' => '۵', '6' => '۶', '7' => '۷', '8' => '۸', '9' => '۹',
        '٠' => '۰', '١' => '۱', '٢' => '۲', '٣' => '۳', '٤' => '۴', '٥' => '۵', '٦' => '۶', '٧' => '۷', '٨' => '۸', '٩' => '۹',
    ];

    private const TO_LATIN = [
        '۰' => '0', '۱' => '1', '۲' => '2', '۳' => '3', '۴' => '4', '۵' => '5', '۶' => '6', '۷' => '7', '۸' => '8', '۹' => '9',
        '٠' => '0', '١' => '1', '٢' => '2', '٣' => '3', '٤' => '4', '٥' => '5', '٦' => '6', '٧' => '7', '٨' => '8', '٩' => '9',
    ];

    /** Replaces every Latin / Arabic-Indic digit with its Persian form; other characters are kept as they are. */
    public static function toPersian(string|int|float|null $value): string
    {
        return strtr((string) $value, self::TO_PERSIAN);
    }

    /** Replaces Persian / Arabic-Indic digits with Latin ones and the Persian separators with `,` / `.`. */
    public static function toLatin(?string $value): string
    {
        return strtr((string) $value, self::TO_LATIN + [self::THOUSANDS_SEPARATOR => ',', self::DECIMAL_SEPARATOR => '.']);
    }

    /** Grouped with «٬», `$decimals` fraction digits with «٫» (trailing zeros dropped), Persian digits. */
    public static function number(int|float $value, int $decimals = 0): string
    {
        $text = number_format($value, max(0, $decimals), '.', ',');
        if (str_contains($text, '.')) {
            $text = rtrim(rtrim($text, '0'), '.');
        }
        if ($text === '-0') {
            $text = '0';
        }

        return self::toPersian(strtr($text, [',' => self::THOUSANDS_SEPARATOR, '.' => self::DECIMAL_SEPARATOR]));
    }
}
