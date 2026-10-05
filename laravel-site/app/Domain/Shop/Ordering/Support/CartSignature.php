<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Support;

use App\Domain\Shop\Cart\Data\CartItemData;
use App\Domain\Shop\Cart\Data\CartSummary;

/**
 * Fingerprint of what the shopper saw on the checkout page: the buyable lines (key, quantity, unit price) and the
 * shipping fee. Sent back with the form; when the live cart no longer matches (price changed, stock reduced, cart
 * edited in another tab) the order is refused with «سبد تغییر کرد» instead of charging an amount nobody saw. Not a
 * security token — prices always come from the catalog.
 */
final class CartSignature
{
    public static function of(CartSummary $cart): string
    {
        $parts = [];
        foreach ($cart->items as $item) {
            if ($item->available) {
                $parts[] = self::line($item);
            }
        }
        $parts[] = 'ship:'.($cart->shipping->fee->rial ?? 'x');

        return substr(hash('sha256', implode('|', $parts)), 0, 32);
    }

    private static function line(CartItemData $item): string
    {
        return $item->key.':'.$item->quantity.':'.$item->unitPrice->rial;
    }
}
