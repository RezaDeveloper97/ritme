<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Actions;

use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use App\Support\Cache\NamespaceBumper;
use InvalidArgumentException;

/**
 * Changes the stock of a product (no variants) or of one active variant by $delta, atomically: a decrement only
 * happens when enough stock is left (single conditional UPDATE, safe against two concurrent orders), otherwise nothing
 * changes and false is returned. A product with variants must be adjusted through a variant. The product aggregate is
 * recalculated and `shop` (+ `pages`) bumped. For checkout (L6-05) and order cancellation (positive delta).
 */
final class AdjustStock
{
    public function __construct(private readonly RecalculateProductStock $recalculate, private readonly NamespaceBumper $bumper) {}

    public function handle(int $productId, ?int $variantId, int $delta): bool
    {
        if ($delta === 0) {
            return true;
        }

        if ($variantId !== null) {
            $query = ProductVariant::query()->whereKey($variantId)->where('product_id', $productId)->where('is_active', true);
        } else {
            if (ProductVariant::query()->where('product_id', $productId)->exists()) {
                throw new InvalidArgumentException('This product has variants: adjust the stock of a variant.');
            }
            $query = Product::query()->whereKey($productId);
        }

        if ($delta < 0) {
            $query->where('stock_qty', '>=', -$delta);
        }

        $changed = $delta > 0
            ? $query->toBase()->increment('stock_qty', $delta)
            : $query->toBase()->decrement('stock_qty', -$delta);

        if ($changed < 1) {
            return false;
        }

        $this->recalculate->handle($productId);
        $this->bumper->bumpFor(new Product, ['shop']);

        return true;
    }
}
