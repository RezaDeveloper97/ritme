<?php

declare(strict_types=1);

namespace App\Filament\Components\Seo;

/**
 * Approximate rendered width of SERP text (Google: titles ≈ 20 px Arial on desktop, descriptions ≈ 14 px), used by
 * the SEO tab counters and the SERP preview. Widths are per-character averages in em; Persian letters are measured
 * as their connected (medial) forms, diacritics and joiners as zero width. Good enough to warn before Google cuts a
 * title with «…» — not a font renderer.
 */
final class SerpMeasure
{
    public const TITLE_FONT_PX = 20;

    public const DESCRIPTION_FONT_PX = 14;

    /** Desktop title width Google shows before truncating. */
    public const TITLE_MAX_PX = 580;

    public const DESCRIPTION_MAX_PX = 920;

    public const MOBILE_DESCRIPTION_MAX_PX = 680;

    /** Recommended character ranges [min, max]. */
    public const TITLE_CHARS = [30, 60];

    public const DESCRIPTION_CHARS = [70, 160];

    private const NARROW = "ijlI.,:;'|!`()[]{}/\\ ";

    public static function chars(?string $text): int
    {
        return mb_strlen(trim((string) $text));
    }

    public static function pixels(?string $text, int $fontPx): int
    {
        $width = 0.0;
        foreach (mb_str_split(trim((string) $text)) as $char) {
            $width += self::em($char);
        }

        return (int) round($width * $fontPx);
    }

    /**
     * Cuts the text at a word boundary so it fits $maxPx, adding «…» like the result page does.
     */
    public static function truncate(string $text, int $fontPx, int $maxPx): string
    {
        $text = trim((string) preg_replace('/\s+/u', ' ', $text));
        if (self::pixels($text, $fontPx) <= $maxPx) {
            return $text;
        }

        $budget = $maxPx - self::pixels(' …', $fontPx);
        $out = '';
        foreach (explode(' ', $text) as $word) {
            $candidate = $out === '' ? $word : $out.' '.$word;
            if (self::pixels($candidate, $fontPx) > $budget) {
                break;
            }
            $out = $candidate;
        }

        return ($out === '' ? mb_substr($text, 0, 10) : $out).' …';
    }

    /**
     * success (within range) | warning (short / slightly long) | danger (will be cut) | gray (empty → inherited).
     *
     * @param  array{0: int, 1: int}  $range
     */
    public static function status(?string $text, array $range, int $fontPx, int $maxPx): string
    {
        $chars = self::chars($text);
        if ($chars === 0) {
            return 'gray';
        }
        if (self::pixels($text, $fontPx) > $maxPx) {
            return 'danger';
        }

        return $chars < $range[0] || $chars > $range[1] ? 'warning' : 'success';
    }

    /**
     * «۴۵ نویسه · ۴۲۰ از ۵۸۰ پیکسل».
     */
    public static function hint(?string $text, int $fontPx, int $maxPx): string
    {
        return self::digits(self::chars($text)).' نویسه · '.self::digits(self::pixels($text, $fontPx)).' از '.self::digits($maxPx).' پیکسل';
    }

    public static function digits(int|string $value): string
    {
        return strtr((string) $value, ['0' => '۰', '1' => '۱', '2' => '۲', '3' => '۳', '4' => '۴', '5' => '۵', '6' => '۶', '7' => '۷', '8' => '۸', '9' => '۹']);
    }

    private static function em(string $char): float
    {
        $code = mb_ord($char, 'UTF-8');

        return match (true) {
            // Arabic-script diacritics, tatweel-free marks, ZWNJ/ZWJ/RLM/LRM: no advance.
            ($code >= 0x064B && $code <= 0x065F) || $code === 0x0670 || ($code >= 0x200B && $code <= 0x200F) => 0.0,
            str_contains(self::NARROW, $char) => 0.28,
            // Persian/Arabic letters (connected forms average) and Persian digits.
            ($code >= 0x0600 && $code <= 0x06FF) || ($code >= 0xFB50 && $code <= 0xFEFF) => 0.45,
            ctype_digit($char) => 0.556,
            in_array($char, ['m', 'w', 'M', 'W'], true) => 0.83,
            ctype_upper($char) => 0.68,
            ctype_lower($char) => 0.5,
            default => 0.55,
        };
    }
}
