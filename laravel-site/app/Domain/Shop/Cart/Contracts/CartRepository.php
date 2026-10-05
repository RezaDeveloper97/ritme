<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Contracts;

use App\Domain\Shop\Cart\Data\Cart;

/** Where the shopper's cart lives (session). */
interface CartRepository
{
    public function load(): Cart;

    public function save(Cart $cart): void;
}
