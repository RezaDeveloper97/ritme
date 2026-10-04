<?php

declare(strict_types=1);

namespace App\Domain\Directory\Booking\Enums;

/**
 * The part of the day the parent prefers. A preference, not a slot: the place answers with a real time by phone.
 */
enum TimeWindow: string
{
    case Morning = 'morning';
    case Noon = 'noon';
    case Evening = 'evening';
    case Any = 'any';

    public function label(): string
    {
        return match ($this) {
            self::Morning => 'صبح',
            self::Noon => 'ظهر',
            self::Evening => 'عصر',
            self::Any => 'فرقی نمی‌کند',
        };
    }

    /** The hours the window stands for (Persian digits), null for «any». */
    public function hours(): ?string
    {
        return match ($this) {
            self::Morning => '۸ تا ۱۲',
            self::Noon => '۱۲ تا ۱۶',
            self::Evening => '۱۶ تا ۲۰',
            self::Any => null,
        };
    }

    /** «صبح (۸ تا ۱۲)» */
    public function describe(): string
    {
        $hours = $this->hours();

        return $hours === null ? $this->label() : $this->label().' ('.$hours.')';
    }
}
