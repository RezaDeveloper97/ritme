<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Data;

use App\Domain\Seo\Schema\Enums\DayOfWeek;
use InvalidArgumentException;

/**
 * One OpeningHoursSpecification: the same hours on one or more days. `00:00`–`23:59` = open all day;
 * `00:00`–`00:00` = closed that day.
 */
final readonly class OpeningHoursData
{
    /**
     * @param  list<DayOfWeek>  $days
     */
    public function __construct(
        public array $days,
        public string $opens,
        public string $closes,
    ) {
        if ($days === []) {
            throw new InvalidArgumentException('Opening hours need at least one day.');
        }
        foreach ([$opens, $closes] as $time) {
            if (preg_match('/^([01]\d|2[0-3]):[0-5]\d(:[0-5]\d)?$/', $time) !== 1) {
                throw new InvalidArgumentException("Invalid time [{$time}], expected HH:MM.");
            }
        }
    }

    /**
     * @return array<string, mixed>
     */
    public function toNode(): array
    {
        return [
            '@type' => 'OpeningHoursSpecification',
            'dayOfWeek' => array_map(static fn (DayOfWeek $day): string => $day->uri(), $this->days),
            'opens' => $this->opens,
            'closes' => $this->closes,
        ];
    }
}
