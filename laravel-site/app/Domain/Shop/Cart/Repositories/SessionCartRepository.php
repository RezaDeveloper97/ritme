<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Repositories;

use App\Domain\Shop\Cart\Contracts\CartRepository;
use App\Domain\Shop\Cart\Data\Cart;
use Illuminate\Contracts\Session\Session;

/**
 * Cart in the session under `shop_cart` as Cart::toArray() (ids, quantities, price snapshots — a few hundred bytes).
 * An empty cart removes the key.
 */
final class SessionCartRepository implements CartRepository
{
    public const KEY = 'shop_cart';

    public function __construct(private readonly Session $session) {}

    public function load(): Cart
    {
        return Cart::fromArray($this->session->get(self::KEY));
    }

    public function save(Cart $cart): void
    {
        if ($cart->isEmpty()) {
            $this->session->forget(self::KEY);

            return;
        }

        $this->session->put(self::KEY, $cart->toArray());
    }
}
