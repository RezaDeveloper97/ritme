<?php

declare(strict_types=1);

namespace App\Domain\Directory\Support;

use App\Support\Text\PersianDigits;

/**
 * Suitable child ages in months, labelled like the design: «۶ ماه تا ۴ سال», «۱ تا ۶ سال», «۱۲ تا ۳۶ ماه»,
 * «۱ تا ۱۲ ماه». Whole years are used only when the upper bound is above 36 months.
 */
final readonly class AgeRange
{
    public function __construct(
        public ?int $minMonths = null,
        public ?int $maxMonths = null,
    ) {}

    public function isEmpty(): bool
    {
        return $this->minMonths === null && $this->maxMonths === null;
    }

    public function contains(int $months): bool
    {
        return ($this->minMonths === null || $months >= $this->minMonths)
            && ($this->maxMonths === null || $months <= $this->maxMonths);
    }

    public function label(): ?string
    {
        if ($this->isEmpty()) {
            return null;
        }

        $years = ($this->maxMonths ?? $this->minMonths ?? 0) > 36;
        $min = $this->minMonths === null ? null : self::bound($this->minMonths, $years);
        $max = $this->maxMonths === null ? null : self::bound($this->maxMonths, $years);

        if ($min === null) {
            return $max === null ? null : 'تا '.$max[0].' '.$max[1];
        }
        if ($max === null) {
            return 'از '.$min[0].' '.$min[1];
        }

        return $min[1] === $max[1]
            ? $min[0].' تا '.$max[0].' '.$max[1]
            : $min[0].' '.$min[1].' تا '.$max[0].' '.$max[1];
    }

    /**
     * @return array{0: string, 1: string}
     */
    private static function bound(int $months, bool $years): array
    {
        return $years && $months >= 12 && $months % 12 === 0
            ? [PersianDigits::toPersian(intdiv($months, 12)), 'سال']
            : [PersianDigits::toPersian($months), 'ماه'];
    }
}
