<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Actions;

use App\Domain\Shop\Cart\Contracts\CartCatalog;
use App\Domain\Shop\Cart\Contracts\CartRepository;
use App\Domain\Shop\Cart\Data\Cart;
use App\Domain\Shop\Cart\Data\CartChange;
use App\Domain\Shop\Cart\Data\CartNotice;
use App\Domain\Shop\Cart\Enums\CartProblem;
use App\Domain\Shop\Cart\Exceptions\CartException;

/**
 * Sets a line's quantity (≤ 0 removes it), clamped LIVE to the stock left and Cart::MAX_QUANTITY (with a notice).
 * A line whose product / variant no longer exists is dropped (`removed`); a sold-out line is left as is
 * (`out_of_stock`); an unknown key throws `line_missing`.
 */
final class UpdateCartLine
{
    public function __construct(private readonly CartRepository $carts, private readonly CartCatalog $catalog) {}

    public function handle(string $key, int $quantity): CartChange
    {
        $cart = $this->carts->load();
        $line = $cart->line($key) ?? throw new CartException(CartProblem::LineMissing);

        if ($quantity < 1) {
            $cart->remove($key);
            $this->carts->save($cart);

            return new CartChange($key, 0, $cart->count());
        }

        $product = $this->catalog->find([$line->productId])[$line->productId] ?? null;
        $variant = $line->variantId === null ? null : $product?->findVariant($line->variantId);
        if ($product === null || ($line->variantId !== null && $variant === null) || ($line->variantId === null && $product->hasVariants())) {
            $cart->remove($key);
            $this->carts->save($cart);

            throw new CartException(CartProblem::Removed);
        }

        $cap = min($product->available($variant), Cart::MAX_QUANTITY);
        if ($cap < 1) {
            throw new CartException(CartProblem::OutOfStock, ['name' => $product->title], $product->slug);
        }

        $final = min($quantity, $cap);
        $cart->put($line->withQuantity($final)->withPrice($product->unitPrice($variant)->rial));
        $this->carts->save($cart);

        $notice = null;
        if ($final < $quantity) {
            $notice = $cap < Cart::MAX_QUANTITY
                ? new CartNotice(CartProblem::Limited, ['name' => $product->title, 'count' => $cap])
                : new CartNotice(CartProblem::MaxQuantity, ['name' => $product->title, 'count' => Cart::MAX_QUANTITY]);
        }

        return new CartChange($key, $final, $cart->count(), $product->title, $product->slug, $notice);
    }
}
