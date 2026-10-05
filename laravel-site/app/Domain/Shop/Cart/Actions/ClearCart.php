<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Actions;

use App\Domain\Shop\Cart\Contracts\CartRepository;
use App\Domain\Shop\Cart\Data\Cart;

/** Empties the cart (e.g. after an order is placed, L6-05). */
final class ClearCart
{
    public function __construct(private readonly CartRepository $carts) {}

    public function handle(): void
    {
        $this->carts->save(new Cart);
    }
}
