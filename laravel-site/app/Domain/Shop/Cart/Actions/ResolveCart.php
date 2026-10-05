<?php

declare(strict_types=1);

namespace App\Domain\Shop\Cart\Actions;

use App\Domain\Shop\Cart\Contracts\CartCatalog;
use App\Domain\Shop\Cart\Contracts\CartRepository;
use App\Domain\Shop\Cart\Data\Cart;
use App\Domain\Shop\Cart\Data\CartItemData;
use App\Domain\Shop\Cart\Data\CartNotice;
use App\Domain\Shop\Cart\Data\CartSummary;
use App\Domain\Shop\Cart\Data\ShippingRule;
use App\Domain\Shop\Cart\Enums\CartProblem;
use App\Support\Money\Money;

/**
 * Re-validates the stored cart against the LIVE catalog and returns the page model:
 *   - product unpublished / deleted, variant gone or deactivated → line dropped (`removed` notice),
 *   - sold out now → line kept, flagged unavailable, left out of count and totals,
 *   - quantity above the stock left → reduced (`limited` notice on the line),
 *   - price differs from the line's snapshot → snapshot updated (`price_changed` notice on the line).
 * Prices always come from the catalog. The cart is saved only when something changed.
 */
final class ResolveCart
{
    public function __construct(
        private readonly CartRepository $carts,
        private readonly CartCatalog $catalog,
        private readonly ShippingRule $shipping,
    ) {}

    public function handle(): CartSummary
    {
        $cart = $this->carts->load();
        $products = $this->catalog->find($cart->productIds());
        $changed = false;
        $items = [];
        $notices = [];

        foreach ($cart->lines() as $line) {
            $product = $products[$line->productId] ?? null;
            $variant = $line->variantId === null ? null : $product?->findVariant($line->variantId);
            if ($product === null || ($line->variantId !== null && $variant === null) || ($line->variantId === null && $product->hasVariants())) {
                $cart->remove($line->key());
                $changed = true;
                $notices[] = new CartNotice(CartProblem::Removed);

                continue;
            }

            $available = $product->available($variant);
            $max = min($available, Cart::MAX_QUANTITY);
            $price = $product->unitPrice($variant);
            $notice = null;
            $quantity = $line->quantity;

            if ($available < 1) {
                $notice = new CartNotice(CartProblem::OutOfStock, ['name' => $product->title]);
            } elseif ($quantity > $max) {
                $quantity = $max;
                $notice = new CartNotice(CartProblem::Limited, ['name' => $product->title, 'count' => $max]);
            }

            if ($line->unitPriceRial !== $price->rial) {
                if ($notice === null && $line->unitPriceRial > 0) {
                    $notice = new CartNotice(CartProblem::PriceChanged, [
                        'name' => $product->title,
                        'old' => Money::fromRial($line->unitPriceRial)->formatShort(),
                        'new' => $price->formatShort(),
                    ]);
                }
            }

            if ($quantity !== $line->quantity || $line->unitPriceRial !== $price->rial) {
                $cart->put($line->withQuantity($quantity)->withPrice($price->rial));
                $changed = true;
            }

            $items[] = new CartItemData(
                key: $line->key(),
                productId: $product->id,
                variantId: $variant?->id,
                title: $product->title,
                slug: $product->slug,
                variantLabel: $variant === null || $variant->label() === '' ? null : $variant->label(),
                brand: $product->brand,
                unitPrice: $price,
                compareAtPrice: $product->compareAt($variant),
                quantity: $quantity,
                maxQuantity: max(1, $max),
                available: $available > 0,
                coverMediaId: $product->coverMediaId,
                illustration: $product->illustration,
                isDemo: $product->isDemo,
                notice: $notice,
            );
        }

        if ($changed) {
            $this->carts->save($cart);
        }

        $buyable = array_values(array_filter($items, static fn (CartItemData $i): bool => $i->available));
        $subtotal = Money::sum(array_map(static fn (CartItemData $i): Money => $i->lineTotal(), $buyable));
        $count = array_sum(array_map(static fn (CartItemData $i): int => $i->quantity, $buyable));
        $quote = $this->shipping->quote($subtotal);

        return new CartSummary(
            items: $items,
            count: $count,
            subtotal: $subtotal,
            shipping: $quote,
            total: $quote->fee === null || $count === 0 ? $subtotal : $subtotal->plus($quote->fee),
            notices: array_values(array_unique($notices, SORT_REGULAR)),
        );
    }
}
