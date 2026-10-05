<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Data;

use App\Support\Money\Money;

/**
 * A resolved cart line for the view: live title / price / stock. `available` false = sold out now (kept so the shopper
 * sees it, left out of the totals). `maxQuantity` = what the stock (and the per-line cap) allows.
 */
final readonly class CartItemData
{
    public function __construct(
        public string $key,
        public int $productId,
        public ?int $variantId,
        public string $title,
        public string $slug,
        public ?string $variantLabel,
        public ?string $brand,
        public Money $unitPrice,
        public ?Money $compareAtPrice,
        public int $quantity,
        public int $maxQuantity,
        public bool $available,
        public ?int $coverMediaId,
        public ?string $illustration,
        public bool $isDemo,
        public ?CartNotice $notice = null,
    ) {}

    public function lineTotal(): Money
    {
        return $this->unitPrice->times($this->quantity);
    }
}
