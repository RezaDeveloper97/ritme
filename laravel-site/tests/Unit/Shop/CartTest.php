<?php

declare(strict_types=1);

use App\Domain\Shop\Cart\Data\Cart;
use App\Domain\Shop\Cart\Data\CartLine;
use App\Domain\Shop\Cart\Data\CartProduct;
use App\Domain\Shop\Cart\Data\ShippingRule;
use App\Domain\Shop\Catalog\Data\VariantData;
use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Support\Money\Money;

function cartProduct(StockStatus $status = StockStatus::InStock, int $stock = 5, array $variants = []): CartProduct
{
    return new CartProduct(1, 'بادی', 'bodysuit', null, Money::fromToman(100_000), null, $stock, $status, null, null, false, $variants);
}

it('keys lines by product + variant, merges by key and counts items', function (): void {
    $cart = new Cart([new CartLine(1, 7, 2, 10), new CartLine(2, null, 1, 20)]);
    $cart->put(new CartLine(1, 7, 3, 10));

    expect($cart->lineCount())->toBe(2)
        ->and($cart->count())->toBe(4)
        ->and($cart->quantityOf('p1-v7'))->toBe(3)
        ->and($cart->has('p2'))->toBeTrue()
        ->and($cart->productIds())->toBe([1, 2]);
});

it('clamps quantities to the per-line cap and removes lines put with zero', function (): void {
    $cart = new Cart;
    $cart->put(new CartLine(1, null, 99, 0));
    expect($cart->quantityOf('p1'))->toBe(Cart::MAX_QUANTITY);

    $cart->put(new CartLine(1, null, 0, 0));
    expect($cart->isEmpty())->toBeTrue();
});

it('caps the number of lines', function (): void {
    $cart = new Cart(array_map(static fn (int $i): CartLine => new CartLine($i, null, 1, 0), range(1, 40)));

    expect($cart->lineCount())->toBe(Cart::MAX_LINES)->and($cart->isFull())->toBeTrue();
});

it('round-trips through the compact session payload and cleans junk', function (): void {
    $cart = new Cart([new CartLine(3, 9, 2, 4_850_000), new CartLine(4, null, 1, 980_000)]);
    expect(Cart::fromArray($cart->toArray())->toArray())->toBe($cart->toArray());

    $junk = Cart::fromArray(['v' => 1, 'l' => [['a'], [5, 'x', 1, 0], [-1, null, 1, 0], [6, null, 0, 0], [7, null, 50, -3], 'zz']]);
    expect($junk->toArray()['l'])->toBe([[7, null, 10, 0]])
        ->and(Cart::fromArray('garbage')->isEmpty())->toBeTrue()
        ->and(Cart::fromArray(['v' => 2, 'l' => [[1, null, 1, 0]]])->isEmpty())->toBeTrue();
});

it('quotes shipping: unknown without a rule, flat fee, free over the threshold with progress', function (): void {
    $none = (new ShippingRule)->quote(Money::fromToman(500_000));
    expect($none->isKnown())->toBeFalse()->and($none->threshold)->toBeNull();

    $rule = ShippingRule::fromToman(45_000, 1_000_000);
    $below = $rule->quote(Money::fromToman(720_000));
    expect($below->fee?->toToman())->toBe(45_000)
        ->and($below->free)->toBeFalse()
        ->and($below->remaining?->toToman())->toBe(280_000)
        ->and($below->progress)->toBe(72);

    $free = $rule->quote(Money::fromToman(1_000_000));
    expect($free->free)->toBeTrue()->and($free->fee?->isZero())->toBeTrue()->and($free->progress)->toBe(100);

    expect(ShippingRule::fromToman('junk', 0)->freeOver)->toBeNull();
});

it('knows how many units can be bought live', function (): void {
    $variant = new VariantData(9, null, '۳-۶ ماه', 'شیری', null, Money::fromToman(120_000), null, 3);

    expect(cartProduct(stock: 4)->available(null))->toBe(4)
        ->and(cartProduct(StockStatus::OutOfStock, 4)->available(null))->toBe(0)
        ->and(cartProduct(StockStatus::PreOrder, 0)->available(null))->toBe(PHP_INT_MAX)
        ->and(cartProduct(variants: [$variant])->available($variant))->toBe(3)
        ->and(cartProduct(variants: [$variant])->unitPrice($variant)->toToman())->toBe(120_000)
        ->and(cartProduct(variants: [$variant])->variantFor('۳-۶ ماه', 'شیری'))->toBe($variant)
        ->and(cartProduct(variants: [$variant])->variantFor('۳-۶ ماه', null))->toBeNull();
});
