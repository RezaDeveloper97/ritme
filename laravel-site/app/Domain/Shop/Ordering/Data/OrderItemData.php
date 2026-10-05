<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Data;

use App\Domain\Shop\Ordering\Models\OrderItem;
use App\Support\Money\Money;

/**
 * An order line for the order page (snapshot taken at checkout).
 */
final readonly class OrderItemData
{
    public function __construct(
        public string $title,
        public ?string $variantLabel,
        public Money $unitPrice,
        public int $quantity,
        public Money $lineTotal,
        public ?int $coverMediaId,
        public ?string $illustration,
    ) {}

    public static function fromModel(OrderItem $item): self
    {
        return new self(
            title: $item->title,
            variantLabel: $item->variant_label,
            unitPrice: $item->unit_price,
            quantity: $item->quantity,
            lineTotal: $item->line_total,
            coverMediaId: $item->cover_media_id,
            illustration: $item->illustration,
        );
    }
}
