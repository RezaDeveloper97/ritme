<?php

declare(strict_types=1);

use App\Domain\Shop\Catalog\Actions\SyncProductCategories;
use App\Domain\Shop\Catalog\Contracts\CatalogRepository;
use App\Domain\Shop\Catalog\Contracts\ProductRepository;
use App\Domain\Shop\Catalog\Data\ProductCriteria;
use App\Domain\Shop\Catalog\Models\Brand;
use App\Domain\Shop\Catalog\Models\Category;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductReview;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use App\Domain\Shop\Catalog\Repositories\CachedCatalogRepository;
use App\Domain\Shop\Catalog\Repositories\CachedProductRepository;
use App\Support\Cache\NamespaceVersions;
use Illuminate\Support\Facades\DB;

beforeEach(function (): void {
    $this->category = Category::factory()->create(['slug' => 'baby-clothes']);
    $this->product = Product::factory()->published()->create(['slug' => 'bodysuit', 'brand_id' => Brand::factory()->create()->id]);
    app(SyncProductCategories::class)->handle($this->product, [$this->category->id]);
    ProductVariant::factory()->create(['product_id' => $this->product->id]);
});

it('binds the cached decorators', function (): void {
    expect(app(ProductRepository::class))->toBeInstanceOf(CachedProductRepository::class)
        ->and(app(CatalogRepository::class))->toBeInstanceOf(CachedCatalogRepository::class);
});

it('serves warm reads without queries', function (): void {
    $products = app(ProductRepository::class);
    $catalog = app(CatalogRepository::class);
    $read = static function () use ($products, $catalog): void {
        $products->findPublishedBySlug('bodysuit');
        $products->list(new ProductCriteria(categoryId: $catalog->findCategory('baby-clothes')?->id));
        $products->bestSellers(5);
        $products->frequentlyBoughtWith(1);
        $products->reviews(1);
        $catalog->brands();
    };

    $read();
    DB::enableQueryLog();
    $read();
    expect(DB::getQueryLog())->toBe([]);
    DB::disableQueryLog();
});

it('round-trips the product page DTO through the cache unchanged', function (): void {
    $repo = app(ProductRepository::class);
    $cold = $repo->findPublishedBySlug('bodysuit');
    $warm = $repo->findPublishedBySlug('bodysuit');

    expect($warm?->toArray())->toBe($cold?->toArray())
        ->and($warm?->price->rial)->toBe($this->product->price->rial)
        ->and($warm?->primaryCategory?->slug)->toBe('baby-clothes');
});

it('bumps shop, sitemap and pages on catalog changes', function (string $change): void {
    $versions = app(NamespaceVersions::class);
    $before = [$versions->version('shop'), $versions->version('sitemap'), $versions->version('pages')];

    match ($change) {
        'product' => $this->product->update(['title' => 'عنوان تازه']),
        'category' => $this->category->update(['name' => 'نام تازه']),
        'brand' => Brand::factory()->create(),
    };

    expect($versions->version('shop'))->toBeGreaterThan($before[0])
        ->and($versions->version('sitemap'))->toBeGreaterThan($before[1])
        ->and($versions->version('pages'))->toBeGreaterThan($before[2]);
})->with(['product', 'category', 'brand']);

it('bumps shop and pages on variant and review changes', function (string $change): void {
    $versions = app(NamespaceVersions::class);
    $before = [$versions->version('shop'), $versions->version('pages')];

    match ($change) {
        'variant' => ProductVariant::query()->first()?->update(['stock_qty' => 0]),
        'review' => ProductReview::factory()->approved()->create(['product_id' => $this->product->id]),
    };

    expect($versions->version('shop'))->toBeGreaterThan($before[0])
        ->and($versions->version('pages'))->toBeGreaterThan($before[1]);
})->with(['variant', 'review']);

it('shows a renamed product after the bump', function (): void {
    $repo = app(ProductRepository::class);
    expect($repo->findPublishedBySlug('bodysuit')?->title)->toBe($this->product->title);

    $this->product->update(['title' => 'بادی تازه']);

    expect($repo->findPublishedBySlug('bodysuit')?->title)->toBe('بادی تازه');
});
