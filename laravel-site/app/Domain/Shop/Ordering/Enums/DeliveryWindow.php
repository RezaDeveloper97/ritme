<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Enums;

/**
 * Preferred delivery time of day (design: «۹ تا ۱۳», «۱۷ تا ۲۱»). A preference the shop confirms on the phone.
 */
enum DeliveryWindow: string
{
    case Morning = 'morning';
    case Evening = 'evening';

    /** «۹ تا ۱۳» */
    public function hours(): string
    {
        return match ($this) {
            self::Morning => '۹ تا ۱۳',
            self::Evening => '۱۷ تا ۲۱',
        };
    }
}
