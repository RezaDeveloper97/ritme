<?php

declare(strict_types=1);

namespace App\Domain\Seo\Analysis;

use App\Domain\Search\Support\SearchTerms;

/**
 * Persian-aware text helpers for the analyser. Matching runs on SearchTerms::normalize() output on both sides
 * (Arabic ي/ك → ی/ک, ZWNJ / tatweel / harakat removed, digits Latin, lower case), and a keyword letter sequence may be
 * split by one space in the text — so «می‌شود», «میشود» and «می شود» are the same word, as are «درد پریود» and
 * «دردپریود». A keyword must start at a word boundary but may carry a suffix («پریود» matches «پریودی», «پریودها»).
 */
final class PersianText
{
    public static function normalize(string $text): string
    {
        return SearchTerms::normalize($text);
    }

    /**
     * Regex for a phrase in normalised text, or null when the phrase normalises to nothing.
     */
    public static function pattern(string $phrase, bool $wholeWord = false): ?string
    {
        $letters = mb_str_split(str_replace(' ', '', self::normalize($phrase)));
        if ($letters === []) {
            return null;
        }

        $body = implode(' ?', array_map(static fn (string $c): string => preg_quote($c, '/'), $letters));

        return '/(?<![\p{L}\p{N}])'.$body.($wholeWord ? '(?![\p{L}\p{N}])' : '').'/u';
    }

    public static function count(string $phrase, string $text, bool $wholeWord = false): int
    {
        $pattern = self::pattern($phrase, $wholeWord);
        if ($pattern === null || trim($text) === '') {
            return 0;
        }

        return (int) preg_match_all($pattern, self::normalize($text));
    }

    public static function contains(string $phrase, string $text, bool $wholeWord = false): bool
    {
        return self::count($phrase, $text, $wholeWord) > 0;
    }

    /**
     * Words of plain text: whitespace-separated tokens carrying a letter or digit (ZWNJ keeps «می‌شود» one word).
     *
     * @return list<string>
     */
    public static function words(string $text): array
    {
        $tokens = preg_split('/[\s\x{00A0}]+/u', trim($text), -1, PREG_SPLIT_NO_EMPTY) ?: [];

        return array_values(array_filter($tokens, static fn (string $t): bool => preg_match('/[\p{L}\p{N}]/u', $t) === 1));
    }

    public static function wordCount(string $text): int
    {
        return count(self::words($text));
    }

    /**
     * Sentences of plain text: split after . ! ? ؟ … ; ؛ followed by space, and at line breaks («۲.۵» stays whole).
     *
     * @return list<string>
     */
    public static function sentences(string $text): array
    {
        $parts = preg_split('/(?<=[.!?؟…;؛])\s+|[\r\n]+/u', trim($text), -1, PREG_SPLIT_NO_EMPTY) ?: [];

        return array_values(array_filter(array_map(trim(...), $parts), static fn (string $s): bool => self::wordCount($s) > 0));
    }

    public static function isLatin(string $text): bool
    {
        return preg_match('/\p{Arabic}/u', $text) !== 1;
    }
}
