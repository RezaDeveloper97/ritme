<?php

declare(strict_types=1);

use App\Domain\Shop\Cart\Actions\UpdateCartLine;
use App\Domain\Shop\Cart\Contracts\CartCatalog;
use App\Domain\Shop\Cart\Contracts\CartRepository;
use App\Domain\Shop\Cart\Data\Cart;
use App\Domain\Shop\Cart\Data\CartLine;
use App\Domain\Shop\Cart\Data\CartProduct;
use App\Domain\Shop\Cart\Enums\CartProblem;
use App\Domain\Shop\Cart\Exceptions\CartException;
use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Support\Money\Money;

/**
 * @param  array<int, CartProduct>  $products
 * @return array{0: UpdateCartLine, 1: object{cart: Cart}}
 */
function updateCartLine(Cart $cart, array $products): array
{
    $store = new class($cart) implements CartRepository
    {
        public function __construct(public Cart $cart) {}

        public function load(): Cart
        {
            return Cart::fromArray($this->cart->toArray());
        }

        public function save(Cart $cart): void
        {
            $this->cart = $cart;
        }
    };
    $catalog = new class($products) implements CartCatalog
    {
        /** @param array<int, CartProduct> $products */
        public function __construct(private readonly array $products) {}

        public function find(array $productIds): array
        {
            return array_intersect_key($this->products, array_flip($productIds));
        }
    };

    return [new UpdateCartLine($store, $catalog), $store];
}

function plainProduct(int $stock = 5, StockStatus $status = StockStatus::InStock, int $toman = 120_000): CartProduct
{
    return new CartProduct(1, 'بادی', 'bodysuit', null, Money::fromToman($toman), null, $stock, $status, null, null, false, []);
}

it('sets the quantity and re-reads the live price', function (): void {
    [$action, $store] = updateCartLine(new Cart([new CartLine(1, null, 1, 10)]), [1 => plainProduct(toman: 150_000)]);

    $change = $action->handle(CartLine::keyFor(1, null), 3);

    expect($change->quantity)->toBe(3)
        ->and($change->count)->toBe(3)
        ->and($change->notice)->toBeNull()
        ->and($store->cart->line(CartLine::keyFor(1, null))?->unitPriceRial)->toBe(Money::fromToman(150_000)->rial);
});

it('removes the line for a quantity below one', function (): void {
    [$action, $store] = updateCartLine(new Cart([new CartLine(1, null, 2, 10)]), [1 => plainProduct()]);

    $change = $action->handle(CartLine::keyFor(1, null), 0);

    expect($change->quantity)->toBe(0)->and($change->count)->toBe(0)->and($store->cart->isEmpty())->toBeTrue();
});

it('clamps to the stock left with a limited notice', function (): void {
    [$action] = updateCartLine(new Cart([new CartLine(1, null, 1, 10)]), [1 => plainProduct(stock: 2)]);

    $change = $action->handle(CartLine::keyFor(1, null), 5);

    expect($change->quantity)->toBe(2)
        ->and($change->notice?->problem)->toBe(CartProblem::Limited)
        ->and($change->notice?->params)->toBe(['name' => 'بادی', 'count' => 2]);
});

it('clamps to the per-line maximum when stock is not the limit', function (): void {
    [$action] = updateCartLine(new Cart([new CartLine(1, null, 1, 10)]), [1 => plainProduct(status: StockStatus::PreOrder)]);

    $change = $action->handle(CartLine::keyFor(1, null), 99);

    expect($change->quantity)->toBe(Cart::MAX_QUANTITY)->and($change->notice?->problem)->toBe(CartProblem::MaxQuantity);
});

it('refuses sold-out lines but keeps them in the cart', function (): void {
    [$action, $store] = updateCartLine(new Cart([new CartLine(1, null, 1, 10)]), [1 => plainProduct(stock: 0, status: StockStatus::OutOfStock)]);

    expect(fn () => $action->handle(CartLine::keyFor(1, null), 2))
        ->toThrow(fn (CartException $e) => expect($e->problem)->toBe(CartProblem::OutOfStock)->and($e->productSlug)->toBe('bodysuit'));
    expect($store->cart->has(CartLine::keyFor(1, null)))->toBeTrue();
});

it('drops lines whose product or variant is gone', function (?CartProduct $product, ?int $variantId): void {
    $key = CartLine::keyFor(1, $variantId);
    [$action, $store] = updateCartLine(new Cart([new CartLine(1, $variantId, 1, 10)]), $product === null ? [] : [1 => $product]);

    expect(fn () => $action->handle($key, 2))->toThrow(fn (CartException $e) => expect($e->problem)->toBe(CartProblem::Removed));
    expect($store->cart->has($key))->toBeFalse();
})->with([
    'unpublished product' => [null, null],
    'deleted variant' => [fn () => plainProduct(), 7],
]);

it('throws line_missing for an unknown key', function (): void {
    [$action] = updateCartLine(new Cart, []);

    expect(fn () => $action->handle('nope', 1))->toThrow(fn (CartException $e) => expect($e->problem)->toBe(CartProblem::LineMissing));
});
