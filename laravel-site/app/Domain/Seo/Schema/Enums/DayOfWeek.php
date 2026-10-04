<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Enums;

/**
 * schema.org DayOfWeek (Iranian week order: Saturday first).
 */
enum DayOfWeek: string
{
    case Saturday = 'Saturday';
    case Sunday = 'Sunday';
    case Monday = 'Monday';
    case Tuesday = 'Tuesday';
    case Wednesday = 'Wednesday';
    case Thursday = 'Thursday';
    case Friday = 'Friday';

    public function uri(): string
    {
        return 'https://schema.org/'.$this->value;
    }
}
