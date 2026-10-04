<?php

declare(strict_types=1);

namespace App\Domain\Seo\Support;

/**
 * Turns an excerpt / body fragment into a meta description: strips tags, decodes entities, collapses whitespace
 * and trims to a limit at a word boundary. Persian-aware: the zero-width non-joiner (U+200C) joins parts of one
 * word, so it is never a cut point; trailing Persian/Latin punctuation is dropped before the ellipsis.
 */
final class DescriptionText
{
    public const LIMIT = 155;

    public static function fromExcerpt(string $text, int $limit = self::LIMIT): string
    {
        $text = html_entity_decode(strip_tags(str_replace(['<br>', '<br/>', '<br />', '</p>'], ' ', $text)), ENT_QUOTES | ENT_HTML5, 'UTF-8');
        // Normalise whitespace (incl. NBSP and Unicode spaces) but keep ZWNJ, which belongs inside Persian words.
        $text = trim((string) preg_replace('/[\s\x{00A0}\x{2000}-\x{200B}\x{202F}\x{205F}\x{3000}]+/u', ' ', $text));

        if (mb_strlen($text) <= $limit) {
            return $text;
        }

        // Leave room for the ellipsis, then back up to the last real space.
        $cut = mb_substr($text, 0, $limit - 1);
        $space = mb_strrpos($cut, ' ');
        if ($space !== false && $space > (int) ($limit * 0.6)) {
            $cut = mb_substr($cut, 0, $space);
        }

        $cut = (string) preg_replace('/[\s\x{200C}،؛,.:;!?؟\-—–«(]+$/u', '', $cut);

        return $cut.'…';
    }
}
