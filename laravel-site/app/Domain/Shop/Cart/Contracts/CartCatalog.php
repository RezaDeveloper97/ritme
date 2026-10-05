<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Contracts;

use App\Domain\Shop\Cart\Data\CartProduct;

/**
 * Live (uncached) read of the published products in a cart: the cart must see the real stock and price, not the
 * up-to-one-cache-lifetime-old copy of ProductRepository.
 */
interface CartCatalog
{
    /**
     * @param  list<int>  $productIds
     * @return array<int, CartProduct> keyed by product id; unknown / unpublished ids are missing
     */
    public function find(array $productIds): array;
}
