<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Queries;

use App\Domain\Shop\Cart\Contracts\CartCatalog;
use App\Domain\Shop\Cart\Data\CartProduct;
use App\Domain\Shop\Catalog\Models\Product;

/**
 * One query for the products + one for their variants (+ brands), straight from the database.
 */
final class LiveCartCatalog implements CartCatalog
{
    public function find(array $productIds): array
    {
        if ($productIds === []) {
            return [];
        }

        $products = [];
        foreach (Product::query()->published()->whereKey($productIds)->with(['brand', 'variants'])->get() as $product) {
            $products[$product->id] = CartProduct::fromModel($product);
        }

        return $products;
    }
}
