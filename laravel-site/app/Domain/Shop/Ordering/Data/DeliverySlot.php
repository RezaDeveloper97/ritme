<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Data;

use App\Domain\Shop\Ordering\Enums\DeliveryWindow;
use Carbon\CarbonImmutable;

/**
 * One preferred delivery slot offered at checkout: a Tehran day + a time window. `tomorrow` → labelled «فردا».
 */
final readonly class DeliverySlot
{
    public function __construct(public CarbonImmutable $date, public DeliveryWindow $window, public bool $tomorrow = false) {}

    /** Form value: `2026-10-08|morning`. */
    public function value(): string
    {
        return $this->date->format('Y-m-d').'|'.$this->window->value;
    }

    /** «فردا» or «شنبه ۱۹ مهر». */
    public function dayLabel(): string
    {
        return $this->tomorrow ? 'فردا' : jdate($this->date, 'l j F');
    }
}
