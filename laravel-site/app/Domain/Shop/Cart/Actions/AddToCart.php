<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Actions;

use App\Domain\Shop\Cart\Contracts\CartCatalog;
use App\Domain\Shop\Cart\Contracts\CartRepository;
use App\Domain\Shop\Cart\Data\Cart;
use App\Domain\Shop\Cart\Data\CartChange;
use App\Domain\Shop\Cart\Data\CartLine;
use App\Domain\Shop\Cart\Data\CartNotice;
use App\Domain\Shop\Cart\Enums\CartProblem;
use App\Domain\Shop\Cart\Exceptions\CartException;

/**
 * Adds $quantity of a product to the cart. A product with variants needs the size + colour of an active variant
 * (`variantFor`). Stock is checked LIVE: the line never holds more than the stock left (or Cart::MAX_QUANTITY);
 * asking for more adds what is possible and returns a `limited` / `max_quantity` notice. Sold out, unknown product /
 * variant, a full cart or a line already at its limit throw a CartException and change nothing.
 */
final class AddToCart
{
    public function __construct(private readonly CartRepository $carts, private readonly CartCatalog $catalog) {}

    public function handle(int $productId, ?string $size, ?string $color, int $quantity = 1): CartChange
    {
        $product = $this->catalog->find([$productId])[$productId] ?? throw new CartException(CartProblem::NotFound);
        $size = self::clean($size);
        $color = self::clean($color);

        $variant = null;
        if ($product->hasVariants()) {
            $variant = $product->variantFor($size, $color);
            if ($variant === null) {
                throw new CartException($size === null && $color === null ? CartProblem::VariantRequired : CartProblem::VariantUnknown, ['name' => $product->title], $product->slug);
            }
        }

        $available = $product->available($variant);
        if ($available < 1) {
            throw new CartException(CartProblem::OutOfStock, ['name' => $product->title], $product->slug);
        }

        $cart = $this->carts->load();
        $key = CartLine::keyFor($product->id, $variant?->id);
        $existing = $cart->quantityOf($key);
        if ($existing === 0 && $cart->isFull()) {
            throw new CartException(CartProblem::TooManyLines, ['count' => Cart::MAX_LINES], $product->slug);
        }

        $cap = min($available, Cart::MAX_QUANTITY);
        $wanted = $existing + max(1, $quantity);
        $final = min($wanted, $cap);
        if ($final <= $existing) {
            throw new CartException(CartProblem::AllInCart, ['name' => $product->title, 'count' => $existing], $product->slug);
        }

        $cart->put(new CartLine($product->id, $variant?->id, $final, $product->unitPrice($variant)->rial));
        $this->carts->save($cart);

        $notice = null;
        if ($final < $wanted) {
            $notice = $cap < Cart::MAX_QUANTITY
                ? new CartNotice(CartProblem::Limited, ['name' => $product->title, 'count' => $cap])
                : new CartNotice(CartProblem::MaxQuantity, ['name' => $product->title, 'count' => Cart::MAX_QUANTITY]);
        }

        return new CartChange($key, $final, $cart->count(), $product->title, $product->slug, $notice);
    }

    private static function clean(?string $value): ?string
    {
        $value = $value === null ? null : trim($value);

        return $value === '' ? null : $value;
    }
}
