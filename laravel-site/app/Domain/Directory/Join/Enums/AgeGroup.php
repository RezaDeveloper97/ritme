<?php

declare(strict_types=1);

namespace App\Domain\Directory\Join\Enums;

use App\Domain\Directory\Support\AgeRange;

/**
 * The age chips of the join form (design/html/directory-join.html «رده سنی»). A place stores one month range
 * (AgeRange); `range()` turns the chosen chips into it. «بارداری» is kept as a chip only — it has no child age.
 */
enum AgeGroup: string
{
    case Pregnancy = 'pregnancy';
    case Months0To6 = '0-6m';
    case Months6To12 = '6-12m';
    case Years1To2 = '1-2y';
    case Years2To4 = '2-4y';
    case Years4To6 = '4-6y';

    public function label(): string
    {
        return match ($this) {
            self::Pregnancy => 'بارداری',
            self::Months0To6 => '۰ تا ۶ ماه',
            self::Months6To12 => '۶ تا ۱۲ ماه',
            self::Years1To2 => '۱ تا ۲ سال',
            self::Years2To4 => '۲ تا ۴ سال',
            self::Years4To6 => '۴ تا ۶ سال',
        };
    }

    /**
     * @return array{0: int, 1: int}|null child age in months, null for pregnancy
     */
    public function months(): ?array
    {
        return match ($this) {
            self::Pregnancy => null,
            self::Months0To6 => [0, 6],
            self::Months6To12 => [6, 12],
            self::Years1To2 => [12, 24],
            self::Years2To4 => [24, 48],
            self::Years4To6 => [48, 72],
        };
    }

    /**
     * @param  list<self>  $groups
     */
    public static function range(array $groups): AgeRange
    {
        $min = null;
        $max = null;
        foreach ($groups as $group) {
            $months = $group->months();
            if ($months === null) {
                continue;
            }
            $min = $min === null ? $months[0] : min($min, $months[0]);
            $max = $max === null ? $months[1] : max($max, $months[1]);
        }

        return new AgeRange($min, $max);
    }

    /**
     * @return array<string, string> value => label, in chip order
     */
    public static function options(): array
    {
        $options = [];
        foreach (self::cases() as $case) {
            $options[$case->value] = $case->label();
        }

        return $options;
    }
}
