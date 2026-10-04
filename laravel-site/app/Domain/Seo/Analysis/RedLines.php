<?php

declare(strict_types=1);

namespace App\Domain\Seo\Analysis;

/**
 * Content red lines from the brand brief: absolute-promise words and diagnosis / cure claims. Matched on normalised
 * text (PersianText), so «حتماً», «حتما» and «حتمــاً» are one word and «دقیق‌ترین» / «دقیق ترین» / «دقيقترين» too.
 */
final class RedLines
{
    /** Banned words (whole words), as written in the brief. */
    public const WORDS = ['حتماً', 'قطعاً', 'دقیق‌ترین', 'تضمینی', 'تضمین‌شده', 'صددرصد'];

    /**
     * Diagnosis / cure claim patterns on normalised text (no ZWNJ, Persian letters, single spaces) → label.
     * Kept narrow on purpose: «برای تشخیص به پزشک مراجعه کنید» is good advice, «ریتمی بیماری شما را تشخیص می‌دهد» is not.
     */
    private const CLAIMS = [
        '/(?<![\p{L}])(ریتمی|ریتم|ritme|اپ|اپلیکیشن|برنامه|ابزار|این تست|آزمون|محاسبه ?گر|ماشین ?حساب)( \S+){0,4} تشخیص (می ?دهد|می ?دهیم|دهد|بدهد)/u' => 'ادعای تشخیص توسط اپ یا ابزار',
        '/(?<![\p{L}])تشخیص (می ?دهیم|قطعی)/u' => 'ادعای تشخیص',
        '/(?<![\p{L}])(درمان|بهبود) (قطعی|کامل و قطعی|تضمینی)/u' => 'وعده درمان قطعی',
        '/(?<![\p{L}])(درمان می ?کنیم|درمان می ?کند|شفا می ?دهد)/u' => 'ادعای درمان',
        '/(?<![\p{L}])شما( \S+){0,3} مبتلا (هستید|هستی|شده اید)/u' => 'برچسب بیماری به خواننده',
        '/(?<![\p{L}])(بیماری|مشکل) (شما|تو) (را )?تشخیص/u' => 'ادعای تشخیص بیماری خواننده',
        '/(?<![\p{L}])(جایگزین (ویزیت |مشاوره )?پزشک|نیازی به (مراجعه به |ویزیت )?پزشک نیست|بدون نیاز به (مراجعه به )?پزشک)/u' => 'جایگزین کردن پزشک',
    ];

    /**
     * Banned words found in the text, in brief spelling, each once.
     *
     * @return list<string>
     */
    public static function words(string $text): array
    {
        $found = [];
        foreach (self::WORDS as $word) {
            if (PersianText::contains($word, $text, wholeWord: true)) {
                $found[] = $word;
            }
        }

        return $found;
    }

    /**
     * Labels of the diagnosis / cure claims found, each once.
     *
     * @return list<string>
     */
    public static function claims(string $text): array
    {
        $normalized = PersianText::normalize($text);
        $found = [];
        foreach (self::CLAIMS as $pattern => $label) {
            if (preg_match($pattern, $normalized) === 1 && ! in_array($label, $found, true)) {
                $found[] = $label;
            }
        }

        return $found;
    }
}
