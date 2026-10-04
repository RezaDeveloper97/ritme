<?php

declare(strict_types=1);

namespace App\Support\Html;

/**
 * URL/fragment-safe slug that keeps Persian letters (content slugs may be Persian — tasks/README decision) and is used
 * for heading ids too.
 *
 *  - Arabic ي/ك/ة/ۀ → Persian ی/ک/ه, Arabic-Indic and Persian digits → ASCII, tatweel and diacritics removed.
 *  - ZWNJ (نیم‌فاصله) and every run of non letter/digit characters become one separator.
 *  - Latin letters are lower-cased; the result is trimmed to $maxLength characters on a word boundary when possible.
 */
final class TextSlug
{
    private const MAP = [
        'ي' => 'ی', 'ى' => 'ی', 'ك' => 'ک', 'ة' => 'ه', 'ۀ' => 'ه', 'ؤ' => 'و', 'إ' => 'ا', 'أ' => 'ا', 'ٱ' => 'ا',
        '۰' => '0', '۱' => '1', '۲' => '2', '۳' => '3', '۴' => '4', '۵' => '5', '۶' => '6', '۷' => '7', '۸' => '8', '۹' => '9',
        '٠' => '0', '١' => '1', '٢' => '2', '٣' => '3', '٤' => '4', '٥' => '5', '٦' => '6', '٧' => '7', '٨' => '8', '٩' => '9',
    ];

    public static function make(string $text, int $maxLength = 120, string $separator = '-'): string
    {
        $text = strtr($text, self::MAP);
        $text = (string) preg_replace('/[\x{064B}-\x{065F}\x{0670}\x{0640}]/u', '', $text); // harakat, tatweel
        $text = mb_strtolower($text, 'UTF-8');
        $text = (string) preg_replace('/[^\p{L}\p{N}]+/u', $separator, $text);
        $text = trim($text, $separator);

        if (mb_strlen($text, 'UTF-8') > $maxLength) {
            $cut = mb_substr($text, 0, $maxLength, 'UTF-8');
            $boundary = mb_strrpos($cut, $separator, 0, 'UTF-8');
            $text = $boundary !== false && $boundary > (int) ($maxLength / 2) ? mb_substr($cut, 0, $boundary, 'UTF-8') : $cut;
            $text = trim($text, $separator);
        }

        return $text;
    }
}
