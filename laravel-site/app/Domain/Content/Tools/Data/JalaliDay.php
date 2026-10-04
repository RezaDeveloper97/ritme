<?php

declare(strict_types=1);

namespace App\Domain\Content\Tools\Data;

use App\Support\Jalali\JalaliCalendar;
use App\Support\Jalali\JalaliDate;
use App\Support\Text\PersianDigits;

/**
 * A calendar day for the calculators: Jalali parts + Julian Day Number, so «LMP + 280 days» is integer maths.
 * Display: «۱۷ بهمن ۱۴۰۵» (with year) or «۱۷ بهمن» — the same strings resources/js/lib/jalali.js `formatDay()` makes.
 */
final readonly class JalaliDay
{
    private function __construct(
        public int $year,
        public int $month,
        public int $day,
        public int $number,
    ) {}

    public static function of(int $year, int $month, int $day): self
    {
        return new self($year, $month, $day, JalaliCalendar::toDayNumber($year, $month, $day));
    }

    public static function fromNumber(int $number): self
    {
        [$year, $month, $day] = JalaliCalendar::fromDayNumber($number);

        return new self($year, $month, $day, $number);
    }

    public static function fromDate(JalaliDate $date): self
    {
        return self::of($date->year, $date->month, $date->day);
    }

    public function addDays(int $days): self
    {
        return self::fromNumber($this->number + $days);
    }

    public function format(bool $withYear = true): string
    {
        $text = $this->day.' '.JalaliDate::MONTHS[$this->month].($withYear ? ' '.$this->year : '');

        return PersianDigits::toPersian($text);
    }

    /** `Y-m-d` (Latin digits) — the value used in test vectors and `<time datetime>`-like attributes. */
    public function toIso(): string
    {
        return sprintf('%04d-%02d-%02d', $this->year, $this->month, $this->day);
    }

    /**
     * «۳ بهمن تا ۱ اسفند ۱۴۰۵»: the start drops its year when both days share one.
     */
    public static function formatRange(self $from, self $to, string $joiner = 'تا'): string
    {
        return $from->format($from->year !== $to->year).' '.$joiner.' '.$to->format();
    }
}
