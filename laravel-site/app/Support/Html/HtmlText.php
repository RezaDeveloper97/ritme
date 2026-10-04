<?php

declare(strict_types=1);

namespace App\Support\Html;

/**
 * Plain text out of HTML and a Persian-aware word count (ZWNJ joins the parts of one word: «می‌شود» is one word).
 */
final class HtmlText
{
    public static function plain(string $html): string
    {
        $text = (string) preg_replace('/<(script|style)\b[^>]*>.*?<\/\1>/is', ' ', $html);
        $text = (string) preg_replace('/<[^>]+>/', ' ', $text); // a space per tag keeps words of adjacent blocks apart
        $text = html_entity_decode($text, ENT_QUOTES | ENT_HTML5, 'UTF-8');

        return trim((string) preg_replace('/[\s\x{00A0}\x{200F}\x{200E}]+/u', ' ', $text));
    }

    public static function wordCount(string $html): int
    {
        $count = 0;
        foreach (explode(' ', self::plain($html)) as $token) {
            if (preg_match('/[\p{L}\p{N}]/u', $token) === 1) {
                $count++;
            }
        }

        return $count;
    }

    /**
     * Minutes to read at $wordsPerMinute (Persian body text: ~200), never below 1.
     */
    public static function readingMinutes(int $words, int $wordsPerMinute = 200): int
    {
        return max(1, (int) ceil($words / max(1, $wordsPerMinute)));
    }
}
