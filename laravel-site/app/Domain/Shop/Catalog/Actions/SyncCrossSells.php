<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Actions;

use App\Domain\Shop\Catalog\Models\Product;
use App\Support\Cache\NamespaceBumper;

/**
 * Replaces a product's hand-picked «معمولاً با این می‌خرند» list (in this order; the product itself is ignored). Bumps
 * `shop` (+ `pages`).
 */
final class SyncCrossSells
{
    public function __construct(private readonly NamespaceBumper $bumper) {}

    /**
     * @param  list<int>  $productIds
     */
    public function handle(Product $product, array $productIds): void
    {
        $ordered = [];
        $position = 0;
        foreach (array_unique(array_map(intval(...), $productIds)) as $id) {
            if ($id > 0 && $id !== $product->id) {
                $ordered[$id] = ['sort_order' => $position++];
            }
        }

        $product->crossSells()->sync($ordered);
        $this->bumper->bumpFor($product, ['shop']);
    }
}
