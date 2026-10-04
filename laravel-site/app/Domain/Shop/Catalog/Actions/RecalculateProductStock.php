<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Actions;

use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductVariant;

/**
 * Stores a product's aggregate stock: with variants, `stock_qty` = the sum over its ACTIVE variants (inactive ones are
 * not for sale); without variants the product's own quantity stays. `stock_status` follows the quantity unless it is
 * a manual PreOrder/BackOrder. Written without model events or touching `updated_at` (callers bump the caches).
 */
final class RecalculateProductStock
{
    public function handle(int $productId): void
    {
        $product = Product::query()->whereKey($productId)->toBase()->first(['stock_qty', 'stock_status']);
        if ($product === null) {
            return;
        }

        $quantity = (int) $product->stock_qty;
        if (ProductVariant::query()->where('product_id', $productId)->exists()) {
            $quantity = (int) ProductVariant::query()->where('product_id', $productId)->where('is_active', true)->sum('stock_qty');
        }

        $status = StockStatus::forQuantity($quantity, StockStatus::tryFrom((string) $product->stock_status));

        Product::query()->whereKey($productId)->toBase()->update(['stock_qty' => $quantity, 'stock_status' => $status->value]);
    }
}
