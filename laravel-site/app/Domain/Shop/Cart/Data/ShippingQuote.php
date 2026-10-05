<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Data;

use App\Support\Money\Money;

/**
 * Shipping estimate for a subtotal. `fee` null = not known yet («محاسبه در مرحله بعد»); `threshold` / `remaining` /
 * `progress` (0–100) drive the free-shipping bar when a free-shipping threshold is set.
 */
final readonly class ShippingQuote
{
    public function __construct(
        public ?Money $fee,
        public bool $free,
        public ?Money $threshold,
        public ?Money $remaining,
        public int $progress,
    ) {}

    public function isKnown(): bool
    {
        return $this->fee !== null;
    }
}
