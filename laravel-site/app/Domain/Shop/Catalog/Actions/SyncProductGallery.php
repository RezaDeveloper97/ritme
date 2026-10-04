<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Actions;

use App\Domain\Shop\Catalog\Models\Product;
use App\Support\Cache\NamespaceBumper;

/**
 * Replaces a product's gallery with the given media ids in this order (sort_order = position). Bumps `shop` and
 * `sitemap` (+ `pages`).
 */
final class SyncProductGallery
{
    public function __construct(private readonly NamespaceBumper $bumper) {}

    /**
     * @param  list<int>  $mediaIds
     */
    public function handle(Product $product, array $mediaIds): void
    {
        $ordered = [];
        foreach (array_values(array_unique(array_map(intval(...), $mediaIds))) as $position => $mediaId) {
            $ordered[$mediaId] = ['sort_order' => $position];
        }

        $product->gallery()->sync($ordered);
        $this->bumper->bumpFor($product, ['shop', 'sitemap']);
    }
}
