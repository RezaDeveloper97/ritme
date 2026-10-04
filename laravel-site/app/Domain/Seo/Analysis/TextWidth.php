<?php

declare(strict_types=1);

namespace App\Domain\Seo\Analysis;

/**
 * Approximate rendered width of SERP text (Google: titles ≈ 20 px Arial on desktop, descriptions ≈ 14 px). Widths are
 * per-character averages in em; Persian letters are measured as their connected (medial) forms, diacritics and
 * joiners as zero width. Good enough to warn before Google cuts a title with «…» — not a font renderer.
 */
final class TextWidth
{
    public const TITLE_FONT_PX = 20;

    public const DESCRIPTION_FONT_PX = 14;

    /** Desktop title width Google shows before truncating. */
    public const TITLE_MAX_PX = 580;

    public const DESCRIPTION_MAX_PX = 920;

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
