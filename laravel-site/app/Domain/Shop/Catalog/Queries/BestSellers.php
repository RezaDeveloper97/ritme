<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Queries;

use App\Domain\Shop\Catalog\Data\ProductCardData;
use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Domain\Shop\Catalog\Models\Product;
use Illuminate\Database\Query\Builder as QueryBuilder;

/**
 * Purchasable published products by units sold (`sales_count`, fed by orders in L6-05), then real review count,
 * featured flag and the editor's sort order — so before the first order the editor's picks lead. Optionally limited
 * to a set of categories (pass a subtree from CategoryTree::descendantIds()) and excluding some products.
 */
final class BestSellers
{
    /**
     * @param  list<int>  $categoryIds
     * @param  list<int>  $excludeIds
     */
    public function __construct(
        private readonly int $limit = 10,
        private readonly array $categoryIds = [],
        private readonly array $excludeIds = [],
    ) {}

    /**
     * @return list<ProductCardData>
     */
    public function get(): array
    {
        return ProductCards::load($this->ids());
    }

    /**
     * @return list<int>
     */
    public function ids(): array
    {
        if ($this->limit < 1) {
            return [];
        }

        $table = (new Product)->getTable();
        $categoryIds = $this->categoryIds;

        return array_values(Product::query()
            ->published()
            ->where("{$table}.stock_status", '!=', StockStatus::OutOfStock->value)
            ->when($this->excludeIds !== [], fn ($q) => $q->whereNotIn("{$table}.id", $this->excludeIds))
            ->when($categoryIds !== [], static fn ($q) => $q->whereExists(static function (QueryBuilder $sub) use ($table, $categoryIds): void {
                $sub->selectRaw('1')
                    ->from('shop_category_product')
                    ->whereColumn('shop_category_product.product_id', "{$table}.id")
                    ->whereIn('shop_category_product.category_id', $categoryIds);
            }))
            ->orderByDesc("{$table}.sales_count")
            ->orderByDesc("{$table}.rating_count")
            ->orderByDesc("{$table}.is_featured")
            ->orderBy("{$table}.sort_order")
            ->orderByDesc("{$table}.id")
            ->limit($this->limit)
            ->pluck("{$table}.id")
            ->map(intval(...))
            ->all());
    }
}
