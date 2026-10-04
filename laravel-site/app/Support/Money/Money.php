<?php

declare(strict_types=1);

namespace App\Support\Money;

use App\Support\Text\PersianDigits;
use InvalidArgumentException;
use JsonSerializable;
use OverflowException;
use Stringable;

/**
 * An amount of Iranian money, stored as an INTEGER number of rials (ISO 4217 `IRR`) — never a float. The site shows
 * tomans (1 toman = 10 rials), schema.org offers use rials directly.
 *
 *   Money::fromToman(129000)->format()       →  ۱۲۹٬۰۰۰ تومان
 *   Money::fromToman(485000)->formatShort()  →  ۴۸۵ هزار تومان     (whole thousands read «… هزار», like the design)
 *   Money::fromRial(1290000)->toToman()      →  129000
 *   Money::fromToman(390000)->percentOff(Money::fromToman(460000))  →  15
 *
 * Arithmetic is integer-only and throws OverflowException instead of silently turning into a float.
 */
final readonly class Money implements JsonSerializable, Stringable
{
    public const RIALS_PER_TOMAN = 10;

    public const UNIT = 'تومان';

    public const CURRENCY = 'IRR';

    private function __construct(public int $rial) {}

    public static function fromRial(int $rial): self
    {
        return new self($rial);
    }

    public static function fromToman(int $toman): self
    {
        if ($toman > intdiv(PHP_INT_MAX, self::RIALS_PER_TOMAN) || $toman < intdiv(PHP_INT_MIN, self::RIALS_PER_TOMAN)) {
            throw new OverflowException("{$toman} toman does not fit into an integer number of rials.");
        }

        return new self($toman * self::RIALS_PER_TOMAN);
    }

    public static function zero(): self
    {
        return new self(0);
    }

    /**
     * @param  iterable<self>  $amounts
     */
    public static function sum(iterable $amounts): self
    {
        $total = self::zero();
        foreach ($amounts as $amount) {
            $total = $total->plus($amount);
        }

        return $total;
    }

    public static function min(self $first, self ...$others): self
    {
        foreach ($others as $other) {
            $first = $other->rial < $first->rial ? $other : $first;
        }

        return $first;
    }

    public static function max(self $first, self ...$others): self
    {
        foreach ($others as $other) {
            $first = $other->rial > $first->rial ? $other : $first;
        }

        return $first;
    }

    public function rial(): int
    {
        return $this->rial;
    }

    /**
     * Whole tomans, rounded half away from zero (a stray 5-rial remainder rounds up).
     */
    public function toToman(): int
    {
        $toman = intdiv($this->rial, self::RIALS_PER_TOMAN);
        $remainder = $this->rial % self::RIALS_PER_TOMAN;

        if (abs($remainder) * 2 >= self::RIALS_PER_TOMAN) {
            $toman += $remainder > 0 ? 1 : -1;
        }

        return $toman;
    }

    public function plus(self $other): self
    {
        return new self(self::checked($this->rial + $other->rial));
    }

    public function minus(self $other): self
    {
        return new self(self::checked($this->rial - $other->rial));
    }

    public function times(int $quantity): self
    {
        return new self(self::checked($this->rial * $quantity));
    }

    public function isZero(): bool
    {
        return $this->rial === 0;
    }

    public function isPositive(): bool
    {
        return $this->rial > 0;
    }

    public function isNegative(): bool
    {
        return $this->rial < 0;
    }

    public function equals(self $other): bool
    {
        return $this->rial === $other->rial;
    }

    /** -1, 0 or 1. */
    public function compareTo(self $other): int
    {
        return $this->rial <=> $other->rial;
    }

    public function greaterThan(self $other): bool
    {
        return $this->rial > $other->rial;
    }

    public function lessThan(self $other): bool
    {
        return $this->rial < $other->rial;
    }

    /**
     * Whole-percent discount of this (sale) price against a higher compare-at price, rounded down so the badge never
     * overstates the saving; null when there is no real discount.
     */
    public function percentOff(self $compareAt): ?int
    {
        if ($compareAt->rial <= 0 || $this->rial < 0 || $compareAt->rial <= $this->rial) {
            return null;
        }

        $percent = intdiv(($compareAt->rial - $this->rial) * 100, $compareAt->rial);

        return $percent > 0 ? $percent : null;
    }

    /**
     * The toman amount in Persian digits: grouped «۱۲۹٬۰۰۰», or «۴۸۵ هزار» for whole thousands when $thousands is
     * true (AUDIT §2.2 price style). No unit.
     */
    public function formatAmount(bool $thousands = false): string
    {
        $toman = $this->toToman();

        if ($thousands && $toman >= 1000 && $toman % 1000 === 0) {
            return PersianDigits::number(intdiv($toman, 1000)).' هزار';
        }

        return PersianDigits::number($toman);
    }

    /** «۱۲۹٬۰۰۰ تومان» */
    public function format(): string
    {
        return $this->formatAmount().' '.self::UNIT;
    }

    /** «۴۸۵ هزار تومان» (design cards), «۱۲٬۵۰۰ تومان» when not whole thousands. */
    public function formatShort(): string
    {
        return $this->formatAmount(true).' '.self::UNIT;
    }

    /** Rials, so a JSON round trip (cache, API) stays an exact integer. */
    public function jsonSerialize(): int
    {
        return $this->rial;
    }

    public function __toString(): string
    {
        return $this->format();
    }

    private static function checked(int|float $result): int
    {
        if (! is_int($result)) {
            throw new OverflowException('Money arithmetic overflowed the integer range.');
        }

        return $result;
    }

    /**
     * Parses a stored or typed value: int = rials; numeric strings (Latin or Persian digits, «٬»/, separators) = rials.
     */
    public static function parseRial(int|string $value): self
    {
        if (is_int($value)) {
            return new self($value);
        }

        $digits = str_replace([',', ' '], '', PersianDigits::toLatin(trim($value)));
        if (preg_match('/^-?\d+$/', $digits) !== 1 || filter_var($digits, FILTER_VALIDATE_INT) === false) {
            throw new InvalidArgumentException("«{$value}» is not an integer amount of rials.");
        }

        return new self((int) $digits);
    }
}
