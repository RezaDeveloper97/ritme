<?php

declare(strict_types=1);

namespace App\Domain\Content\Tools;

use App\Domain\Content\Tools\Data\DueDateResult;
use App\Domain\Content\Tools\Data\FertilityWindowResult;
use App\Domain\Content\Tools\Data\JalaliDay;
use App\Support\Text\PersianDigits;

/**
 * Turns a calculator result into the two lines of its result box (big value + detail line) from the `tools.text`
 * templates (lang/fa/tools.php). Twin of `dueText()` / `fertilityText()` in resources/js/lib/jalali.js, which gets
 * the same templates through the form's `data-text` attribute — so JS and no-JS show identical words.
 *
 * Templates: separator, range (:from :to), due_progress (:weeks :days), due_progress_whole (:weeks),
 * due_progress_days (:days, first week), due_range (:range), fert_detail (:ovulation :next). On the LMP day itself
 * no progress line is shown.
 */
final class ResultText
{
    public const KEYS = ['separator', 'range', 'due_progress', 'due_progress_whole', 'due_progress_days', 'due_range', 'fert_detail'];

    /** @var array<string, string> */
    private readonly array $templates;

    /**
     * @param  array<array-key, mixed>  $templates
     */
    public function __construct(array $templates)
    {
        $clean = [];
        foreach (self::KEYS as $key) {
            $clean[$key] = is_string($templates[$key] ?? null) ? $templates[$key] : '';
        }
        $this->templates = $clean;
    }

    /** @return array{value: string, detail: string} */
    public function due(DueDateResult $result): array
    {
        $parts = [];
        $weeks = $result->weeks ?? 0;
        $days = $result->days ?? 0;
        if ($weeks > 0) {
            $parts[] = $days > 0
                ? $this->fill('due_progress', ['weeks' => $weeks, 'days' => $days])
                : $this->fill('due_progress_whole', ['weeks' => $weeks]);
        } elseif ($days > 0) {
            $parts[] = $this->fill('due_progress_days', ['days' => $days]);
        }
        $parts[] = $this->fill('due_range', ['range' => $this->range($result->rangeFrom, $result->rangeTo)]);

        return ['value' => $result->due->format(), 'detail' => implode($this->templates['separator'], $parts)];
    }

    /** @return array{value: string, detail: string} */
    public function fertility(FertilityWindowResult $result): array
    {
        return [
            'value' => $this->range($result->windowFrom, $result->windowTo),
            'detail' => $this->fill('fert_detail', [
                'ovulation' => $result->ovulation->format(withYear: false),
                'next' => $result->nextPeriod->format(),
            ]),
        ];
    }

    /**
     * «۱۲ تا ۱۷ مهر ۱۴۰۵» (same month) · «۳ بهمن تا ۱ اسفند ۱۴۰۵» (same year) · «۲۵ اسفند ۱۴۰۵ تا ۸ فروردین ۱۴۰۶».
     */
    public function range(JalaliDay $from, JalaliDay $to): string
    {
        $sameYear = $from->year === $to->year;
        $start = $sameYear && $from->month === $to->month
            ? PersianDigits::toPersian($from->day)
            : $from->format(withYear: ! $sameYear);

        return $this->fill('range', ['from' => $start, 'to' => $to->format()]);
    }

    /**
     * @param  array<string, int|string>  $values
     */
    private function fill(string $key, array $values): string
    {
        $pairs = [];
        foreach ($values as $name => $value) {
            $pairs[':'.$name] = PersianDigits::toPersian($value);
        }

        return strtr($this->templates[$key], $pairs);
    }
}
