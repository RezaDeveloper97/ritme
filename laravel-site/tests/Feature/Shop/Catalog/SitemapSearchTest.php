<?php

declare(strict_types=1);

use App\Domain\Search\Actions\SearchSite;
use App\Domain\Search\Support\SearchRegistry;
use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Seo\Sitemap\SitemapRegistry;
use App\Domain\Shop\Catalog\Actions\SyncProductCategories;
use App\Domain\Shop\Catalog\Contracts\ProductRepository;
use App\Domain\Shop\Catalog\Models\Brand;
use App\Domain\Shop\Catalog\Models\Category;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Search\ProductSearchProvider;
use App\Domain\Shop\Catalog\Sitemap\CategorySitemapProvider;
use App\Domain\Shop\Catalog\Sitemap\ProductSitemapProvider;

beforeEach(function (): void {
    config(['app.url' => 'https://ritme.ir']);
    url()->forceRootUrl('https://ritme.ir');
    url()->forceScheme('https');
    $this->baby = Category::factory()->create(['slug' => 'baby']);
    $this->clothes = Category::factory()->childOf($this->baby)->create(['slug' => 'baby-clothes']);
    $this->feeding = Category::factory()->childOf($this->baby)->create(['slug' => 'feeding']);
    $this->inCategory = static function (Product $product, Category $category): Product {
        app(SyncProductCategories::class)->handle($product, [$category->id]);

        return $product;
    };
});

it('registers the shop sitemaps and search provider', function (): void {
    $keys = array_map(static fn ($p): string => $p->key(), app(SitemapRegistry::class)->all());

    expect($keys)->toContain('shop-products', 'shop-categories')
        ->and(array_map(static fn ($p): string => $p::class, app(SearchRegistry::class)->all()))->toContain(ProductSearchProvider::class);
});

it('lists indexable, published, non-demo products in the sitemap', function (): void {
    Product::factory()->published()->create(['slug' => 'bodysuit']);
    Product::factory()->published()->demo()->create(['slug' => 'demo']);
    Product::factory()->create(['slug' => 'draft']);
    $hidden = Product::factory()->published()->create(['slug' => 'hidden']);
    SeoMeta::query()->create(['seoable_type' => $hidden->getMorphClass(), 'seoable_id' => $hidden->id, 'robots' => 'noindex, follow']);

    $provider = app(ProductSitemapProvider::class);
    $entries = $provider->entries();

    expect($provider->count())->toBe(1)
        ->and($entries[0]->loc)->toBe('https://ritme.ir/shop/product/bodysuit')
        ->and($entries[0]->lastmod)->not->toBeNull()
        ->and($hidden->getMorphClass())->toBe('shop_product');

    Product::factory()->published()->create(['slug' => 'new-one']);
    expect($provider->count())->toBe(2);
});

it('lists only categories whose subtree has a real product', function (): void {
    ($this->inCategory)(Product::factory()->published()->create(), $this->clothes);
    ($this->inCategory)(Product::factory()->published()->demo()->create(), $this->feeding);
    ($this->inCategory)(Product::factory()->create(), $this->feeding);
    $hidden = Category::factory()->childOf($this->baby)->create(['slug' => 'hidden']);
    ($this->inCategory)(Product::factory()->published()->create(), $hidden);
    SeoMeta::query()->create(['seoable_type' => $hidden->getMorphClass(), 'seoable_id' => $hidden->id, 'sitemap_include' => false]);
    $inactive = Category::factory()->create(['slug' => 'inactive', 'is_active' => false]);
    ($this->inCategory)(Product::factory()->published()->create(), $inactive);

    $locs = array_map(static fn ($e): string => $e->loc, app(CategorySitemapProvider::class)->entries());

    expect($locs)->toBe(['https://ritme.ir/shop/category/baby', 'https://ritme.ir/shop/category/baby-clothes']);

    $this->get('/sitemaps/shop-categories.xml')->assertOk()->assertSee('https://ritme.ir/shop/category/baby-clothes', false);
});

it('finds published products in site search', function (): void {
    $brand = Brand::factory()->create(['name' => 'پوشاک پنبه‌ریز']);
    Product::factory()->published()->priced(485_000)->create([
        'slug' => 'bodysuit', 'title' => 'بادی آستین‌بلند نخی', 'brand_id' => $brand->id, 'short_description' => 'بسته سه‌تایی پنبه‌ای.',
    ]);
    Product::factory()->create(['title' => 'بادی پیش‌نویس']);

    $hits = array_values(array_filter(app(SearchSite::class)->handle('بادی')->hits, static fn ($h): bool => $h->type === 'products'));

    expect($hits)->toHaveCount(1)
        ->and($hits[0]->title)->toBe('بادی آستین‌بلند نخی')
        ->and($hits[0]->url)->toBe('https://ritme.ir/shop/product/bodysuit')
        ->and($hits[0]->typeLabel)->toBe('فروشگاه')
        ->and($hits[0]->snippet)->toContain('پوشاک پنبه‌ریز')->toContain('۴۸۵ هزار تومان')
        ->and($hits[0]->score)->toBe(2);
});

it('emits schema.org Product data in rials with availability and no invented rating', function (): void {
    $product = Product::factory()->published()->priced(485_000)->create(['slug' => 'bodysuit', 'sku' => 'BD-1', 'stock_qty' => 0]);
    ($this->inCategory)($product, $this->clothes);

    $schema = app(ProductRepository::class)->findPublishedBySlug('bodysuit')
        ?->toSchema('https://ritme.ir/shop/product/bodysuit', ['https://ritme.ir/media/a.webp']);

    expect($schema?->offer->price)->toBe(4_850_000)
        ->and($schema?->offer->priceCurrency)->toBe('IRR')
        ->and($schema?->offer->toNode()['availability'])->toBe('https://schema.org/OutOfStock')
        ->and($schema?->sku)->toBe('BD-1')
        ->and($schema?->category)->toBe($this->clothes->name)
        ->and($schema?->rating)->toBeNull();
});
