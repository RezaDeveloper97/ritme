<?php

declare(strict_types=1);

use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Shop\Catalog\Actions\SyncCrossSells;
use App\Domain\Shop\Catalog\Actions\SyncProductCategories;
use App\Domain\Shop\Catalog\Actions\SyncProductGallery;
use App\Domain\Shop\Catalog\Contracts\ProductRepository;
use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Domain\Shop\Catalog\Models\Category;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductSlug;
use App\Support\Money\Money;

it('stores money as integer rials and reads it back as Money', function (): void {
    $product = Product::factory()->priced(485_000, 520_000)->create();

    expect($product->getRawOriginal('price'))->toBe(4_850_000)
        ->and($product->refresh()->price)->toBeInstanceOf(Money::class)
        ->and($product->price->toToman())->toBe(485_000)
        ->and($product->compare_at_price?->toToman())->toBe(520_000);
});

it('drops a compare-at price that is not higher than the price', function (): void {
    $product = Product::factory()->priced(485_000, 485_000)->create();
    expect($product->compare_at_price)->toBeNull();

    $product->update(['compare_at_price' => Money::fromToman(400_000)]);
    expect($product->refresh()->compare_at_price)->toBeNull();
});

it('slugs from the title (Persian kept), keeps history and resolves old slugs', function (): void {
    $product = Product::factory()->published()->create(['title' => 'بادی آستین‌بلند نخی', 'slug' => '']);
    $first = $product->slug;
    expect($first)->not->toBe('');

    $product->update(['slug' => 'long-sleeve-cotton-bodysuit-3']);
    $repo = app(ProductRepository::class);

    expect(ProductSlug::query()->where('slug', $first)->exists())->toBeTrue()
        ->and($repo->currentSlugFor($first))->toBe('long-sleeve-cotton-bodysuit-3')
        ->and(Product::factory()->create(['slug' => $first])->slug)->toBe($first.'-2');

    $product->update(['slug' => $first]);
    expect(ProductSlug::query()->where('slug', $first)->exists())->toBeFalse();
});

it('does not record history for drafts', function (): void {
    $product = Product::factory()->create(['slug' => 'draft-one']);
    $product->update(['slug' => 'draft-two']);

    expect(ProductSlug::query()->count())->toBe(0);
});

it('sanitises the description and normalises structured fields', function (): void {
    $product = Product::factory()->create([
        'description' => '<p onclick="x()">متن <script>alert(1)</script><a href="https://example.com">لینک</a></p>',
        'short_description' => '<b>کوتاه</b>',
        'badge' => ' <i>پنبه ۱۰۰٪</i> ',
        'specs' => ['جنس' => 'پنبه', 'خالی' => ''],
        'size_chart' => ['columns' => ['سایز', 'سن'], 'rows' => [['نوزاد', 'تا ۱ ماه', 'زیادی']]],
        'life_stages' => ['pregnancy', 'bogus', 'pregnancy', 'postpartum'],
        'sku' => '  ',
    ]);

    expect($product->description)->not->toContain('script')->not->toContain('onclick')->toContain('noopener')
        ->and($product->short_description)->toBe('کوتاه')
        ->and($product->badge)->toBe('پنبه ۱۰۰٪')
        ->and($product->specs)->toBe([['label' => 'جنس', 'value' => 'پنبه']])
        ->and($product->size_chart)->toBe(['columns' => ['سایز', 'سن'], 'rows' => [['نوزاد', 'تا ۱ ماه']]])
        ->and($product->lifeStages())->toBe([LifeStage::Pregnancy, LifeStage::Postpartum])
        ->and($product->sku)->toBeNull();
});

it('derives the stock status from the quantity, keeping manual pre-orders', function (): void {
    $product = Product::factory()->create(['stock_qty' => 3]);
    expect($product->stock_status)->toBe(StockStatus::InStock);

    $product->update(['stock_qty' => 0]);
    expect($product->stock_status)->toBe(StockStatus::OutOfStock);

    $product->update(['stock_status' => StockStatus::PreOrder]);
    expect($product->refresh()->stock_status)->toBe(StockStatus::PreOrder)
        ->and($product->stock_status->isPurchasable())->toBeTrue();
});

it('syncs categories with the primary one always attached', function (): void {
    $a = Category::factory()->create();
    $b = Category::factory()->create();
    $product = Product::factory()->create();

    app(SyncProductCategories::class)->handle($product, [$a->id], $b->id);
    expect($product->refresh()->primary_category_id)->toBe($b->id)
        ->and($product->categories()->pluck('shop_categories.id')->sort()->values()->all())->toBe([$a->id, $b->id]);

    $c = Category::factory()->create();
    $product->update(['primary_category_id' => $c->id]);
    expect($product->categories()->count())->toBe(3);
});

it('keeps gallery and cross-sell order and never cross-sells the product itself', function (): void {
    $product = Product::factory()->create();
    $others = Product::factory()->count(3)->create();

    app(SyncCrossSells::class)->handle($product, [$others[2]->id, $product->id, $others[0]->id, $others[2]->id]);
    app(SyncProductGallery::class)->handle($product, []);

    expect($product->crossSells()->pluck('shop_products.id')->all())->toBe([$others[2]->id, $others[0]->id])
        ->and($product->gallery()->count())->toBe(0);
});

it('refuses a category cycle', function (): void {
    $root = Category::factory()->create();
    $child = Category::factory()->childOf($root)->create();

    expect(fn () => $root->update(['parent_id' => $child->id]))->toThrow(InvalidArgumentException::class)
        ->and(Category::factory()->create(['name' => 'لباس نوزاد', 'slug' => ''])->slug)->toBe('لباس-نوزاد');
});
