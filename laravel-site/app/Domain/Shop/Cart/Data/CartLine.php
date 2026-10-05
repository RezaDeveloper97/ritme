<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Data;

/**
 * One stored cart line: product (+ variant) id, quantity and the unit price in rials seen when the line was last
 * checked (server-side snapshot, used only to tell the shopper a price changed — totals always use the live price).
 */
final readonly class CartLine
{
    public function __construct(
        public int $productId,
        public ?int $variantId,
        public int $quantity,
        public int $unitPriceRial,
    ) {}

    public static function keyFor(int $productId, ?int $variantId): string
    {
        return 'p'.$productId.($variantId === null ? '' : '-v'.$variantId);
    }

    public function key(): string
    {
        return self::keyFor($this->productId, $this->variantId);
    }

    public function withQuantity(int $quantity): self
    {
        return new self($this->productId, $this->variantId, $quantity, $this->unitPriceRial);
    }

    public function withPrice(int $unitPriceRial): self
    {
        return new self($this->productId, $this->variantId, $this->quantity, $unitPriceRial);
    }
}
