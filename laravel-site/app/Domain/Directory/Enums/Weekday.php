<?php

declare(strict_types=1);

namespace App\Domain\Directory\Enums;

use App\Domain\Seo\Schema\Enums\DayOfWeek;
use Carbon\CarbonInterface;

/**
 * Days of the Iranian week (Saturday first) — the keys of a place's `opening_hours` JSON.
 */
enum Weekday: string
{
    case Saturday = 'saturday';
    case Sunday = 'sunday';
    case Monday = 'monday';
    case Tuesday = 'tuesday';
    case Wednesday = 'wednesday';
    case Thursday = 'thursday';
    case Friday = 'friday';

    public function label(): string
    {
        return match ($this) {
            self::Saturday => 'شنبه',
            self::Sunday => 'یکشنبه',
            self::Monday => 'دوشنبه',
            self::Tuesday => 'سه‌شنبه',
            self::Wednesday => 'چهارشنبه',
            self::Thursday => 'پنجشنبه',
            self::Friday => 'جمعه',
        };
    }

    public static function fromDate(CarbonInterface $date): self
    {
        return match ($date->dayOfWeek) {
            CarbonInterface::SATURDAY => self::Saturday,
            CarbonInterface::SUNDAY => self::Sunday,
            CarbonInterface::MONDAY => self::Monday,
            CarbonInterface::TUESDAY => self::Tuesday,
            CarbonInterface::WEDNESDAY => self::Wednesday,
            CarbonInterface::THURSDAY => self::Thursday,
            default => self::Friday,
        };
    }

    public function previous(): self
    {
        $cases = self::cases();
        $index = array_search($this, $cases, true);

        return $cases[((int) $index + 6) % 7];
    }

    public function next(): self
    {
        $cases = self::cases();
        $index = array_search($this, $cases, true);

        return $cases[((int) $index + 1) % 7];
    }

    public function schemaDay(): DayOfWeek
    {
        return DayOfWeek::from(ucfirst($this->value));
    }
}
