<?php

declare(strict_types=1);

namespace App\Filament\Components\Seo;

use App\Domain\Seo\Analysis\TextWidth;
use App\Support\Text\PersianDigits;

/**
 * SERP counters and truncation for the SEO tab and the SERP preview. Widths come from the domain's TextWidth (the
 * same numbers the content analyser uses): Google titles ≈ 20 px Arial on desktop, descriptions ≈ 14 px.
 */
final class SerpMeasure
{
    public const TITLE_FONT_PX = TextWidth::TITLE_FONT_PX;

    public const DESCRIPTION_FONT_PX = TextWidth::DESCRIPTION_FONT_PX;

    /** Desktop title width Google shows before truncating. */
    public const TITLE_MAX_PX = TextWidth::TITLE_MAX_PX;

    public const DESCRIPTION_MAX_PX = TextWidth::DESCRIPTION_MAX_PX;

    public const MOBILE_DESCRIPTION_MAX_PX = 680;

    /** Recommended character ranges [min, max]. */
    public const TITLE_CHARS = TextWidth::TITLE_CHARS;

    public const DESCRIPTION_CHARS = TextWidth::DESCRIPTION_CHARS;

    public static function chars(?string $text): int
    {
        return TextWidth::chars($text);
    }

    public static function pixels(?string $text, int $fontPx): int
    {
        return TextWidth::pixels($text, $fontPx);
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
        return PersianDigits::toPersian((string) $value);
    }
}
