<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Data;

/**
 * Outcome of a cart mutation: the line's new quantity (0 = removed), an adjustment notice (e.g. clamped to the
 * stock) and the cart's new item count (for the header badge cookie).
 */
final readonly class CartChange
{
    public function __construct(
        public string $key,
        public int $quantity,
        public int $count,
        public ?string $title = null,
        public ?string $productSlug = null,
        public ?CartNotice $notice = null,
    ) {}
}
