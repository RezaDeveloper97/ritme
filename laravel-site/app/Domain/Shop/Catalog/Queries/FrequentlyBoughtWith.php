<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Queries;

use App\Domain\Shop\Catalog\Data\CategoryTree;
use App\Domain\Shop\Catalog\Data\ProductCardData;
use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Domain\Shop\Catalog\Models\Product;

/**
 * «معمولاً با این می‌خرند»: the product's hand-picked cross-sells (published, purchasable, in order), topped up with
 * best sellers of its primary category subtree, then of the whole shop. Never the product itself. When orders exist
 * (L6-05), co-purchase counts can replace the category top-up without changing callers.
 */
final class FrequentlyBoughtWith
{
    public function __construct(
        private readonly int $productId,
        private readonly CategoryTree $tree,
        private readonly int $limit = 5,
    ) {}

    /**
     * @return list<ProductCardData>
     */
    public function get(): array
    {
        $product = Product::query()->whereKey($this->productId)->first(['id', 'primary_category_id']);
        if ($product === null || $this->limit < 1) {
            return [];
        }

        $table = (new Product)->getTable();
        $ids = array_values($product->crossSells()
            ->where("{$table}.is_published", true)
            ->where("{$table}.stock_status", '!=', StockStatus::OutOfStock->value)
            ->limit($this->limit)
            ->pluck("{$table}.id")
            ->map(intval(...))
            ->all());

        if (count($ids) < $this->limit && $product->primary_category_id !== null) {
            $subtree = $this->tree->descendantIds($product->primary_category_id) ?: [$product->primary_category_id];
            $ids = [...$ids, ...(new BestSellers($this->limit - count($ids), $subtree, [$product->id, ...$ids]))->ids()];
        }

        if (count($ids) < $this->limit) {
            $ids = [...$ids, ...(new BestSellers($this->limit - count($ids), [], [$product->id, ...$ids]))->ids()];
        }

        return ProductCards::load($ids);
    }
}
