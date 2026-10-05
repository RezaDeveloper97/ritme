<?php

declare(strict_types=1);

namespace App\Support\Jalali;

use App\Support\Text\PersianDigits;
use Carbon\CarbonImmutable;
use DateTimeImmutable;
use DateTimeInterface;
use DateTimeZone;
use InvalidArgumentException;

/**
 * An instant shown in the Jalali calendar, in Tehran time by default.
 *
 *   JalaliDate::fromDateTime($post->published_at)->format()    →  ۱۲ مهر ۱۴۰۵
 *   JalaliDate::fromDateTime($at)->format('Y/m/d H:i')          →  ۱۴۰۵/۰۷/۱۲ ۰۹:۳۰
 *   JalaliDate::fromDateTime($at)->format('l j F', false)       →  شنبه 12 مهر   (Latin digits)
 *   JalaliDate::fromJalali(1403, 12, 30)->toDateTime()          →  2025-03-20 00:00 Asia/Tehran
 *
 * Format tokens (PHP `date()` subset): Y y m n d j F (month) l (weekday) D (short weekday) H G h g i s A (ق.ظ/ب.ظ);
 * `\` escapes the next character. Everything else is copied as-is.
 */
final class JalaliDate
{
    public const TIMEZONE = 'Asia/Tehran';

    public const DEFAULT_FORMAT = 'j F Y';

    public const MONTHS = [1 => 'فروردین', 'اردیبهشت', 'خرداد', 'تیر', 'مرداد', 'شهریور', 'مهر', 'آبان', 'آذر', 'دی', 'بهمن', 'اسفند'];

    /** Indexed by ISO-8601 weekday (`N`: 1 = Monday … 7 = Sunday). */
    public const WEEKDAYS = [1 => 'دوشنبه', 'سه‌شنبه', 'چهارشنبه', 'پنج‌شنبه', 'جمعه', 'شنبه', 'یکشنبه'];

    private function __construct(
        public readonly int $year,
        public readonly int $month,
        public readonly int $day,
        private readonly DateTimeImmutable $moment,
    ) {}

    public static function fromDateTime(DateTimeInterface $dateTime, string|DateTimeZone|null $timezone = self::TIMEZONE): self
    {
        $moment = DateTimeImmutable::createFromInterface($dateTime);
        if ($timezone !== null) {
            $moment = $moment->setTimezone(is_string($timezone) ? new DateTimeZone($timezone) : $timezone);
        }

        [$year, $month, $day] = JalaliCalendar::toJalali((int) $moment->format('Y'), (int) $moment->format('n'), (int) $moment->format('j'));

        return new self($year, $month, $day, $moment);
    }

    /** Accepts a DateTime, a Unix timestamp or any `strtotime`-style string (interpreted in `$timezone`). */
    public static function parse(DateTimeInterface|int|string $value, string|DateTimeZone|null $timezone = self::TIMEZONE): self
    {
        if ($value instanceof DateTimeInterface) {
            return self::fromDateTime($value, $timezone);
        }

        if (is_int($value)) {
            return self::fromDateTime(new DateTimeImmutable('@'.$value), $timezone);
        }

        $zone = is_string($timezone) ? new DateTimeZone($timezone) : $timezone;

        try {
            return self::fromDateTime(new DateTimeImmutable($value, $zone), $zone);
        } catch (\Exception $e) {
            throw new InvalidArgumentException("Unparseable date «{$value}».", 0, $e);
        }
    }

    public static function now(string|DateTimeZone $timezone = self::TIMEZONE): self
    {
        // Carbon's clock honours test time travel (travelTo / setTestNow); plain `new DateTimeImmutable` would not.
        return self::fromDateTime(CarbonImmutable::now(), $timezone);
    }

    /** Midnight (or the given time) of a Jalali calendar day in `$timezone`. */
    public static function fromJalali(int $year, int $month, int $day, int $hour = 0, int $minute = 0, int $second = 0, string|DateTimeZone $timezone = self::TIMEZONE): self
    {
        [$gy, $gm, $gd] = JalaliCalendar::toGregorian($year, $month, $day);
        $zone = is_string($timezone) ? new DateTimeZone($timezone) : $timezone;
        $moment = (new DateTimeImmutable('now', $zone))->setDate($gy, $gm, $gd)->setTime($hour, $minute, $second);

        return new self($year, $month, $day, $moment);
    }

    public function toDateTime(): DateTimeImmutable
    {
        return $this->moment;
    }

    public function isLeapYear(): bool
    {
        return JalaliCalendar::isLeapYear($this->year);
    }

    public function monthName(): string
    {
        return self::MONTHS[$this->month];
    }

    public function weekdayName(): string
    {
        return self::WEEKDAYS[(int) $this->moment->format('N')];
    }

    /** `Y/m/d` with Latin digits — for `datetime`-like machine use; prefer `toDateTime()->format('c')` for `<time>`. */
    public function toDateString(): string
    {
        return sprintf('%04d/%02d/%02d', $this->year, $this->month, $this->day);
    }

    public function format(string $format = self::DEFAULT_FORMAT, bool $persianDigits = true): string
    {
        $out = '';
        $length = mb_strlen($format);
        for ($i = 0; $i < $length; $i++) {
            $char = mb_substr($format, $i, 1);
            if ($char === '\\') {
                $out .= $i + 1 < $length ? mb_substr($format, ++$i, 1) : '';

                continue;
            }
            $out .= $this->token($char);
        }

        return $persianDigits ? PersianDigits::toPersian($out) : $out;
    }

    private function token(string $char): string
    {
        return match ($char) {
            'Y' => (string) $this->year,
            'y' => sprintf('%02d', $this->year % 100),
            'm' => sprintf('%02d', $this->month),
            'n' => (string) $this->month,
            'd' => sprintf('%02d', $this->day),
            'j' => (string) $this->day,
            'F' => $this->monthName(),
            'l' => $this->weekdayName(),
            'D' => mb_substr($this->weekdayName(), 0, 1),
            'H', 'G', 'h', 'g', 'i', 's' => $this->moment->format($char),
            'A', 'a' => $this->moment->format('A') === 'AM' ? 'ق.ظ' : 'ب.ظ',
            default => $char,
        };
    }
}
