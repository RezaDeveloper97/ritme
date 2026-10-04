<?php

declare(strict_types=1);

use App\Domain\Shop\Catalog\Contracts\CatalogRepository;
use App\Domain\Shop\Catalog\Contracts\ProductRepository;
use App\Domain\Shop\Catalog\Data\ProductCriteria;
use App\Domain\Shop\Catalog\Models\Category;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductReview;
use App\Domain\Shop\Catalog\Sitemap\CategorySitemapProvider;
use App\Domain\Shop\Catalog\Sitemap\ProductSitemapProvider;
use Database\Seeders\ShopSeeder;

it('seeds the design catalog idempotently, every product marked as demo', function (): void {
    $this->seed(ShopSeeder::class);
    $this->seed(ShopSeeder::class);

    $repo = app(ProductRepository::class);
    $product = $repo->findPublishedBySlug(ShopSeeder::DEMO_PRODUCT_SLUG);
    $tree = app(CatalogRepository::class)->categoryTree();
    $clothes = $tree->findBySlug(ShopSeeder::DEMO_CATEGORY_SLUG);

    expect(Product::query()->count())->toBe(11)
        ->and(Product::query()->where('is_demo', false)->count())->toBe(0)
        ->and(Category::query()->count())->toBe(18)
        ->and(array_map(static fn ($c): string => $c->name, $tree->roots()))->toBe(['سیسمونی و نوزاد', 'آرایشی و بهداشتی'])
        ->and(app(CatalogRepository::class)->brands())->toHaveCount(4)
        ->and($product?->title)->toBe('بادی آستین‌بلند نخی · ۳ عدد')
        ->and($product?->isDemo)->toBeTrue()
        ->and($product?->price->formatShort())->toBe('۴۸۵ هزار تومان')
        ->and($product?->price->rial)->toBe(4_850_000)
        ->and($product?->brand?->name)->toBe('پوشاک پنبه‌ریز')
        ->and($product?->primaryCategory?->slug)->toBe('baby-clothes')
        ->and($product?->sizes())->toBe(['۰-۳ ماه', '۳-۶ ماه', '۶-۹ ماه', '۹-۱۲ ماه', '۱۲-۱۸ ماه'])
        ->and($product?->colors())->toHaveCount(4)
        ->and($product?->variantFor('۱۲-۱۸ ماه', 'شیری')?->isInStock())->toBeFalse()
        ->and($product?->stockQty)->toBe(19 * 6)
        ->and($product?->specs)->toHaveCount(5)
        ->and($product?->sizeChart['rows'] ?? [])->toHaveCount(5)
        ->and($product?->description)->toContain('نمونه نمایشی')
        ->and(array_map(static fn ($c): string => $c->slug, $repo->frequentlyBoughtWith($product->id ?? 0)))
        ->toBe(['baby-socks-5-pairs', 'baby-hat-and-mittens', 'muslin-baby-blanket', 'snap-sleepsuit', 'baby-bottle-240ml'])
        ->and($repo->list(new ProductCriteria(categoryId: $clothes?->id))->total)->toBe(5);
});

it('never seeds ratings or sales: demo reviews are shown labelled but not counted', function (): void {
    $this->seed(ShopSeeder::class);
    $repo = app(ProductRepository::class);
    $product = $repo->findPublishedBySlug(ShopSeeder::DEMO_PRODUCT_SLUG);
    $reviews = $repo->reviews($product->id ?? 0);

    expect(Product::query()->where('rating_count', '>', 0)->count())->toBe(0)
        ->and(Product::query()->where('sales_count', '>', 0)->count())->toBe(0)
        ->and(ProductReview::query()->where('is_demo', false)->count())->toBe(0)
        ->and($product?->rating())->toBeNull()
        ->and($reviews->total)->toBe(3)
        ->and(collect($reviews->items)->every(static fn ($r): bool => $r->isDemo && ! $r->isVerifiedPurchase))->toBeTrue();
});

it('keeps demo products out of the sitemaps', function (): void {
    $this->seed(ShopSeeder::class);

    expect(app(ProductSitemapProvider::class)->count())->toBe(0)
        ->and(app(CategorySitemapProvider::class)->count())->toBe(0);
});

it('keeps demo copy inside the content red lines', function (): void {
    $this->seed(ShopSeeder::class);

    $text = Product::query()->get()->map(static fn (Product $p): string => implode(' ', [$p->title, $p->short_description, $p->description, $p->badge, json_encode($p->specs, JSON_UNESCAPED_UNICODE)]))->implode(' ')
        .' '.ProductReview::query()->pluck('body')->implode(' ')
        .' '.Category::query()->pluck('intro')->implode(' ');

    foreach (['حتماً', 'حتما', 'قطعاً', 'قطعا', 'دقیق‌ترین', 'دقیقترین', 'تضمینی', 'تشخیص', 'بهترین', 'درمان'] as $word) {
        expect($text)->not->toContain($word);
    }
});
