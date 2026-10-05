<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Data;

use App\Domain\Shop\Catalog\Data\VariantData;
use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use App\Support\Money\Money;

/**
 * A published product as the cart sees it, read LIVE (never from the `shop` cache): price, stock and the ACTIVE
 * variants with their effective prices (catalog VariantData).
 */
final readonly class CartProduct
{
    /**
     * @param  list<VariantData>  $variants
     */
    public function __construct(
        public int $id,
        public string $title,
        public string $slug,
        public ?string $brand,
        public Money $price,
        public ?Money $compareAtPrice,
        public int $stockQty,
        public StockStatus $stockStatus,
        public ?int $coverMediaId,
        public ?string $illustration,
        public bool $isDemo,
        public array $variants,
    ) {}

    /**
     * Expects `brand` and `variants` loaded.
     */
    public static function fromModel(Product $product): self
    {
        return new self(
            id: $product->id,
            title: $product->title,
            slug: $product->slug,
            brand: $product->brand?->name,
            price: $product->price,
            compareAtPrice: $product->compare_at_price,
            stockQty: $product->stock_qty,
            stockStatus: $product->stock_status,
            coverMediaId: $product->cover_media_id,
            illustration: $product->illustration,
            isDemo: $product->is_demo,
            variants: array_values($product->variants
                ->filter(static fn (ProductVariant $v): bool => $v->is_active)
                ->map(static fn (ProductVariant $v): VariantData => VariantData::fromModel($v, $product))
                ->all()),
        );
    }

    public function hasVariants(): bool
    {
        return $this->variants !== [];
    }

    public function variantFor(?string $size, ?string $color): ?VariantData
    {
        foreach ($this->variants as $variant) {
            if ($variant->size === $size && $variant->color === $color) {
                return $variant;
            }
        }

        return null;
    }

    public function findVariant(int $id): ?VariantData
    {
        foreach ($this->variants as $variant) {
            if ($variant->id === $id) {
                return $variant;
            }
        }

        return null;
    }

    /** Pre-order / back-order products are sold whatever the stock quantity. */
    public function ignoresStock(): bool
    {
        return $this->stockStatus === StockStatus::PreOrder || $this->stockStatus === StockStatus::BackOrder;
    }

    /** Units that can be bought right now (of the variant, or of the product without variants). */
    public function available(?VariantData $variant): int
    {
        if ($this->stockStatus === StockStatus::OutOfStock && ! $this->hasVariants()) {
            return 0;
        }
        if ($this->ignoresStock()) {
            return PHP_INT_MAX;
        }

        return max(0, $variant === null ? $this->stockQty : $variant->stockQty);
    }

    public function unitPrice(?VariantData $variant): Money
    {
        return $variant === null ? $this->price : $variant->price;
    }

    public function compareAt(?VariantData $variant): ?Money
    {
        $compare = $variant === null ? $this->compareAtPrice : $variant->compareAtPrice;

        return $compare !== null && $compare->greaterThan($this->unitPrice($variant)) ? $compare : null;
    }
}
