<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Actions;

use App\Domain\Shop\Catalog\Models\Product;
use App\Support\Cache\NamespaceBumper;

/**
 * Replaces a product's categories and sets the primary one (first id when none is given; always included in the
 * pivot). Pivot writes fire no model events, so the caches are bumped here (`shop`, `sitemap`, `pages`).
 */
final class SyncProductCategories
{
    public function __construct(private readonly NamespaceBumper $bumper) {}

    /**
     * @param  array<int>  $categoryIds  any keys; duplicates are dropped
     */
    public function handle(Product $product, array $categoryIds, ?int $primaryCategoryId = null): void
    {
        $ids = array_values(array_unique(array_filter(array_map(intval(...), $categoryIds), static fn (int $id): bool => $id > 0)));
        $primary = $primaryCategoryId ?? ($ids[0] ?? null);
        if ($primary !== null && ! in_array($primary, $ids, true)) {
            array_unshift($ids, $primary);
        }

        $product->categories()->sync($ids);

        if ($product->primary_category_id !== $primary) {
            $product->primary_category_id = $primary;
            $product->save(); // the observer bumps the caches
        } else {
            $this->bumper->bumpFor($product, ['shop', 'sitemap']);
        }
    }
}
