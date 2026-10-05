<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Data;

use App\Support\Money\Money;

/**
 * The cart page model: resolved lines, item count + subtotal of the available lines, the shipping estimate, the total
 * (subtotal + known fee) and notices about lines that were dropped while re-validating.
 */
final readonly class CartSummary
{
    /**
     * @param  list<CartItemData>  $items
     * @param  list<CartNotice>  $notices
     */
    public function __construct(
        public array $items,
        public int $count,
        public Money $subtotal,
        public ShippingQuote $shipping,
        public Money $total,
        public array $notices = [],
    ) {}

    public function isEmpty(): bool
    {
        return $this->items === [];
    }

    /** Whether there is anything to check out. */
    public function canCheckout(): bool
    {
        return $this->count > 0;
    }

    public function hasDemo(): bool
    {
        foreach ($this->items as $item) {
            if ($item->isDemo) {
                return true;
            }
        }

        return false;
    }

    /**
     * @return list<int>
     */
    public function productIds(): array
    {
        return array_values(array_unique(array_map(static fn (CartItemData $i): int => $i->productId, $this->items)));
    }
}
