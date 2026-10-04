<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Data;

use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use App\Support\Money\Money;

/**
 * An active variant with its EFFECTIVE price (own price, else the product price) and compare-at price (own, else the
 * product's when the variant has no own price).
 */
final readonly class VariantData
{
    public function __construct(
        public int $id,
        public ?string $sku,
        public ?string $size,
        public ?string $color,
        public ?string $colorHex,
        public Money $price,
        public ?Money $compareAtPrice,
        public int $stockQty,
    ) {}

    public static function fromModel(ProductVariant $variant, Product $product): self
    {
        return new self(
            id: $variant->id,
            sku: $variant->sku,
            size: $variant->size,
            color: $variant->color,
            colorHex: $variant->color_hex,
            price: $variant->price ?? $product->price,
            compareAtPrice: $variant->price === null ? $product->compare_at_price : $variant->compare_at_price,
            stockQty: $variant->stock_qty,
        );
    }

    public function isInStock(): bool
    {
        return $this->stockQty > 0;
    }

    /** «۳-۶ ماه · شیری» — shown in the cart and next to reviews. */
    public function label(): string
    {
        return implode(' · ', array_filter([$this->size, $this->color], static fn (?string $part): bool => $part !== null && $part !== ''));
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'id' => $this->id, 'sku' => $this->sku, 'size' => $this->size, 'color' => $this->color, 'colorHex' => $this->colorHex,
            'price' => $this->price->rial, 'compareAtPrice' => $this->compareAtPrice?->rial, 'stockQty' => $this->stockQty,
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        $string = static fn (string $key): ?string => isset($data[$key]) ? (string) $data[$key] : null;

        return new self(
            id: (int) $data['id'],
            sku: $string('sku'),
            size: $string('size'),
            color: $string('color'),
            colorHex: $string('colorHex'),
            price: Money::fromRial((int) $data['price']),
            compareAtPrice: isset($data['compareAtPrice']) ? Money::fromRial((int) $data['compareAtPrice']) : null,
            stockQty: (int) $data['stockQty'],
        );
    }
}
