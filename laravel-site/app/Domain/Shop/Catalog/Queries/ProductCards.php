<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Queries;

use App\Domain\Shop\Catalog\Data\ProductCardData;
use App\Domain\Shop\Catalog\Models\Product;
use Illuminate\Database\Eloquent\Builder;

/**
 * Loads product cards for an ordered id list in two queries (products + brands), keeping the order and skipping ids
 * that no longer exist. Shared by the listing queries.
 */
final class ProductCards
{
    /**
     * @param  list<int>  $ids
     * @return list<ProductCardData>
     */
    public static function load(array $ids): array
    {
        if ($ids === []) {
            return [];
        }

        $products = Product::query()
            ->whereIn('id', $ids)
            ->with('brand')
            ->withCount(['variants' => static fn (Builder $q) => $q->where('is_active', true)])
            ->get()
            ->keyBy('id');

        $cards = [];
        foreach ($ids as $id) {
            $product = $products->get($id);
            if ($product instanceof Product) {
                $cards[] = ProductCardData::fromModel($product);
            }
        }

        return $cards;
    }

    /**
     * ORDER BY fragment that puts sold-out products last.
     */
    public static function outOfStockLast(string $table): string
    {
        return "CASE WHEN {$table}.stock_status = 'out_of_stock' THEN 1 ELSE 0 END";
    }
}
