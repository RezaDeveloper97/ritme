<?php

declare(strict_types=1);

use App\Domain\Shop\Catalog\Actions\AdjustStock;
use App\Domain\Shop\Catalog\Contracts\ProductRepository;
use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use App\Support\Money\Money;

it('sums the stock of active variants into the product', function (): void {
    $product = Product::factory()->published()->create(['stock_qty' => 99]);
    ProductVariant::factory()->create(['product_id' => $product->id, 'size' => '۰-۳ ماه', 'stock_qty' => 2]);
    ProductVariant::factory()->create(['product_id' => $product->id, 'size' => '۳-۶ ماه', 'stock_qty' => 5]);
    ProductVariant::factory()->inactive()->create(['product_id' => $product->id, 'size' => '۶-۹ ماه', 'stock_qty' => 50]);

    expect($product->refresh()->stock_qty)->toBe(7)
        ->and($product->stock_status)->toBe(StockStatus::InStock);

    ProductVariant::query()->where('product_id', $product->id)->get()->each->update(['stock_qty' => 0]);
    expect($product->refresh()->stock_qty)->toBe(0)
        ->and($product->stock_status)->toBe(StockStatus::OutOfStock);

    // Editing the product itself must not overwrite the variant aggregate.
    $product->update(['title' => 'عنوان تازه', 'stock_qty' => 40]);
    expect($product->refresh()->stock_qty)->toBe(0)
        ->and($product->stock_status)->toBe(StockStatus::OutOfStock)
        ->and($product->title)->toBe('عنوان تازه');
});

it('gives variants the product price unless they have their own', function (): void {
    $product = Product::factory()->published()->priced(390_000, 460_000)->create();
    ProductVariant::factory()->create(['product_id' => $product->id, 'size' => 'کوچک', 'color' => 'شیری', 'color_hex' => '#f6efe6']);
    ProductVariant::factory()->create(['product_id' => $product->id, 'size' => 'بزرگ', 'color' => 'شیری', 'price' => Money::fromToman(420_000), 'compare_at_price' => Money::fromToman(400_000), 'stock_qty' => 0]);

    $data = app(ProductRepository::class)->findPublishedBySlug($product->slug);
    $small = $data?->variantFor('کوچک', 'شیری');
    $large = $data?->variantFor('بزرگ', 'شیری');

    expect($small?->price->toToman())->toBe(390_000)
        ->and($small?->compareAtPrice?->toToman())->toBe(460_000)
        ->and($small?->colorHex)->toBe('#F6EFE6')
        ->and($small?->isInStock())->toBeTrue()
        ->and($small?->label())->toBe('کوچک · شیری')
        ->and($large?->price->toToman())->toBe(420_000)
        ->and($large?->compareAtPrice)->toBeNull()
        ->and($large?->isInStock())->toBeFalse()
        ->and($data?->sizes())->toBe(['کوچک', 'بزرگ'])
        ->and($data?->colors())->toBe([['name' => 'شیری', 'hex' => '#F6EFE6']])
        ->and($data?->variantFor('متوسط', 'شیری'))->toBeNull();
});

it('decrements variant stock atomically and refuses overselling', function (): void {
    $product = Product::factory()->published()->create();
    $variant = ProductVariant::factory()->create(['product_id' => $product->id, 'stock_qty' => 2]);
    $adjust = app(AdjustStock::class);

    expect($adjust->handle($product->id, $variant->id, -2))->toBeTrue()
        ->and($variant->refresh()->stock_qty)->toBe(0)
        ->and($product->refresh()->stock_status)->toBe(StockStatus::OutOfStock)
        ->and($adjust->handle($product->id, $variant->id, -1))->toBeFalse()
        ->and($variant->refresh()->stock_qty)->toBe(0)
        ->and($adjust->handle($product->id, $variant->id, 3))->toBeTrue()
        ->and($product->refresh()->stock_qty)->toBe(3)
        ->and($product->stock_status)->toBe(StockStatus::InStock);
});

it('adjusts products without variants and requires a variant otherwise', function (): void {
    $simple = Product::factory()->create(['stock_qty' => 1]);
    $adjust = app(AdjustStock::class);

    expect($adjust->handle($simple->id, null, -1))->toBeTrue()
        ->and($simple->refresh()->stock_status)->toBe(StockStatus::OutOfStock)
        ->and($adjust->handle($simple->id, null, -1))->toBeFalse();

    $withVariants = Product::factory()->create();
    $inactive = ProductVariant::factory()->inactive()->create(['product_id' => $withVariants->id, 'stock_qty' => 5]);

    expect(fn () => $adjust->handle($withVariants->id, null, -1))->toThrow(InvalidArgumentException::class)
        ->and($adjust->handle($withVariants->id, $inactive->id, -1))->toBeFalse()
        ->and($adjust->handle($simple->id, $inactive->id, -1))->toBeFalse();
});

it('shows fresh stock after an adjustment (cache bumped)', function (): void {
    $product = Product::factory()->published()->create(['stock_qty' => 1]);
    $repo = app(ProductRepository::class);
    expect($repo->findPublishedBySlug($product->slug)?->isPurchasable())->toBeTrue();

    app(AdjustStock::class)->handle($product->id, null, -1);

    expect($repo->findPublishedBySlug($product->slug)?->stockStatus)->toBe(StockStatus::OutOfStock);
});
