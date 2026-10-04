<?php

declare(strict_types=1);

namespace App\Domain\Search\Support;

use App\Support\Text\PersianDigits;

/**
 * A visitor's query, normalised once for every provider (and for the cache key and the term log):
 *
 *   SearchTerms::normalize(' كيف  مي‌خواهم ۱۲ ')  →  'کیف میخواهم 12'
 *
 * Arabic yeh/alef maksura/kaf → Persian ی/ک, teh marbuta/heh-yeh → ه, ZWNJ/ZWJ, tatweel and harakat removed,
 * Persian/Arabic digits → Latin, lower case, whitespace collapsed. Stored content is normalised the same way at match
 * time (SQL REPLACE chain or normalize() in PHP), so «ي» in the database matches «ی» typed and vice versa.
 */
final readonly class SearchTerms
{
    public const MAX_LENGTH = 100;

    public const MAX_TOKENS = 5;

    public const MIN_TOKEN_LENGTH = 2;

    /** Character map applied before everything else (Arabic forms → Persian, invisible joiners removed). */
    public const CHARACTER_MAP = [
        'ي' => 'ی', 'ى' => 'ی', 'ئ' => 'ی', 'ك' => 'ک', 'ة' => 'ه', 'ۀ' => 'ه',
        "\u{200C}" => '', "\u{200D}" => '', "\u{200F}" => '', "\u{200E}" => '', 'ـ' => '',
    ];

    /**
     * @param  list<string>  $tokens
     */
    private function __construct(
        public string $query,
        public string $normalized,
        public array $tokens,
    ) {}

    public static function fromInput(?string $input): self
    {
        $query = trim(preg_replace('/\s+/u', ' ', (string) $input) ?? '');
        $query = mb_substr($query, 0, self::MAX_LENGTH);

        $tokens = [];
        foreach (explode(' ', self::normalize($query)) as $token) {
            if (mb_strlen($token) >= self::MIN_TOKEN_LENGTH && ! in_array($token, $tokens, true)) {
                $tokens[] = $token;
            }
            if (count($tokens) === self::MAX_TOKENS) {
                break;
            }
        }

        return new self($query, implode(' ', $tokens), $tokens);
    }

    public static function normalize(string $text): string
    {
        $text = strtr($text, self::CHARACTER_MAP);
        $text = preg_replace('/[\x{064B}-\x{065F}\x{0670}]/u', '', $text) ?? $text; // harakat, superscript alef
        $text = PersianDigits::toLatin($text);
        $text = mb_strtolower($text);
        $text = preg_replace('/[^\p{L}\p{N}]+/u', ' ', $text) ?? $text; // punctuation splits words

        return trim(preg_replace('/\s+/u', ' ', $text) ?? $text);
    }

    public function isEmpty(): bool
    {
        return $this->tokens === [];
    }

    /**
     * Whether every token occurs in the (raw) text after normalisation — for providers that filter in PHP.
     */
    public function matches(string $text): bool
    {
        $haystack = self::normalize($text);
        foreach ($this->tokens as $token) {
            if (! str_contains($haystack, $token)) {
                return false;
            }
        }

        return $this->tokens !== [];
    }

    /**
     * Whether any token occurs in the text (title boost).
     */
    public function touches(string $text): bool
    {
        $haystack = self::normalize($text);
        foreach ($this->tokens as $token) {
            if (str_contains($haystack, $token)) {
                return true;
            }
        }

        return false;
    }
}
