<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Actions;

use App\Domain\Shop\Cart\Contracts\CartRepository;
use App\Domain\Shop\Cart\Data\CartChange;

/** Removes a line (idempotent: an unknown key changes nothing). */
final class RemoveFromCart
{
    public function __construct(private readonly CartRepository $carts) {}

    public function handle(string $key): CartChange
    {
        $cart = $this->carts->load();
        $cart->remove($key);
        $this->carts->save($cart);

        return new CartChange($key, 0, $cart->count());
    }
}
