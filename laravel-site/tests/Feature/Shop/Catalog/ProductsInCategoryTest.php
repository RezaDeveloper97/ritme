<?php

declare(strict_types=1);

use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Shop\Catalog\Actions\SyncProductCategories;
use App\Domain\Shop\Catalog\Contracts\CatalogRepository;
use App\Domain\Shop\Catalog\Contracts\ProductRepository;
use App\Domain\Shop\Catalog\Data\ProductCriteria;
use App\Domain\Shop\Catalog\Enums\ProductSort;
use App\Domain\Shop\Catalog\Models\Brand;
use App\Domain\Shop\Catalog\Models\Category;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductReview;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use App\Support\Money\Money;

beforeEach(function (): void {
    $this->baby = Category::factory()->create(['slug' => 'baby', 'name' => 'سیسمونی و نوزاد']);
    $this->clothes = Category::factory()->childOf($this->baby)->create(['slug' => 'baby-clothes', 'name' => 'لباس نوزاد', 'sort_order' => 1]);
    $this->feeding = Category::factory()->childOf($this->baby)->create(['slug' => 'feeding', 'name' => 'تغذیه', 'sort_order' => 2]);
    $this->beauty = Category::factory()->create(['slug' => 'beauty']);
    $this->panberiz = Brand::factory()->create(['name' => 'پوشاک پنبه‌ریز', 'slug' => 'panberiz']);
    $this->mahno = Brand::factory()->create(['name' => 'خانه سیسمونی ماه‌نو', 'slug' => 'mahno']);

    $make = function (string $slug, Category $category, int $toman, Brand $brand, array $extra = []): Product {
        $product = Product::factory()->published()->priced($toman)->create(['slug' => $slug, 'brand_id' => $brand->id, ...$extra]);
        app(SyncProductCategories::class)->handle($product, [$category->id]);

        return $product;
    };

    $this->bodysuit = $make('bodysuit', $this->clothes, 485_000, $this->panberiz, ['title' => 'بادی آستین‌بلند نخی', 'life_stages' => ['postpartum'], 'sales_count' => 5]);
    $this->sleepsuit = $make('sleepsuit', $this->clothes, 390_000, $this->panberiz, ['title' => 'سرهمی خواب دکمه‌دار', 'sales_count' => 9]);
    $this->bottle = $make('bottle', $this->feeding, 290_000, $this->mahno, ['title' => 'شیشه شیر ۲۴۰ میلی‌لیتری', 'life_stages' => ['postpartum', 'pregnancy']]);
    $this->soldOut = $make('sold-out', $this->clothes, 100_000, $this->mahno, ['title' => 'کلاه نوزادی', 'stock_qty' => 0, 'sales_count' => 99]);
    $this->lipstick = $make('lipstick', $this->beauty, 310_000, $this->mahno, ['title' => 'رژلب مات']);
    $this->draft = Product::factory()->priced(50_000)->create(['slug' => 'draft']);
    app(SyncProductCategories::class)->handle($this->draft, [$this->clothes->id]);

    foreach (['۰-۳ ماه' => 0, '۳-۶ ماه' => 4] as $size => $stock) {
        ProductVariant::factory()->create(['product_id' => $this->bodysuit->id, 'size' => $size, 'color' => 'شیری', 'color_hex' => '#F6EFE6', 'stock_qty' => $stock]);
    }
    ProductVariant::factory()->create(['product_id' => $this->sleepsuit->id, 'size' => '۰-۳ ماه', 'color' => 'صورتی', 'stock_qty' => 3]);

    $this->list = fn (array $args = []): array => array_map(
        static fn ($card): string => $card->slug,
        app(ProductRepository::class)->list(new ProductCriteria(...$args))->items,
    );
});

it('lists a category with its subcategories, published only, sold-out last', function (): void {
    expect(($this->list)(['categoryId' => $this->baby->id]))->toBe(['sleepsuit', 'bodysuit', 'bottle', 'sold-out'])
        ->and(($this->list)(['categoryId' => $this->clothes->id]))->toBe(['sleepsuit', 'bodysuit', 'sold-out'])
        ->and(($this->list)())->toContain('lipstick')->not->toContain('draft');
});

it('sorts like the design', function (): void {
    $id = $this->baby->id;
    $this->bottle->forceFill(['created_at' => now()->addDay()])->saveQuietly();
    ProductReview::factory()->approved()->create(['product_id' => $this->bottle->id, 'rating' => 5]);

    expect(($this->list)(['categoryId' => $id, 'sort' => ProductSort::Cheapest]))->toBe(['bottle', 'sleepsuit', 'bodysuit', 'sold-out'])
        ->and(($this->list)(['categoryId' => $id, 'sort' => ProductSort::Newest])[0])->toBe('bottle')
        ->and(($this->list)(['categoryId' => $id, 'sort' => ProductSort::TopRated])[0])->toBe('bottle');
});

it('filters by brand, price, stock, life stage and text', function (): void {
    $id = $this->baby->id;

    expect(($this->list)(['categoryId' => $id, 'brandIds' => [$this->mahno->id]]))->toBe(['bottle', 'sold-out'])
        ->and(($this->list)(['categoryId' => $id, 'minPrice' => Money::fromToman(300_000), 'maxPrice' => Money::fromToman(400_000)]))->toBe(['sleepsuit'])
        ->and(($this->list)(['categoryId' => $id, 'inStockOnly' => true]))->not->toContain('sold-out')
        ->and(($this->list)(['lifeStage' => LifeStage::Pregnancy]))->toBe(['bottle'])
        ->and(($this->list)(['text' => 'شيشه']))->toBe(['bottle'])          // Arabic yeh matches
        ->and(($this->list)(['text' => '240']))->toBe(['bottle'])            // Latin digits match Persian
        ->and(($this->list)(['text' => 'پنبه‌ریز']))->toBe(['sleepsuit', 'bodysuit']) // brand name
        ->and(($this->list)(['text' => '100%']))->toBe([]);
});

it('filters by variant size and colour, optionally in stock', function (): void {
    expect(($this->list)(['sizes' => ['۰-۳ ماه']]))->toBe(['sleepsuit', 'bodysuit'])
        ->and(($this->list)(['sizes' => ['۰-۳ ماه'], 'inStockOnly' => true]))->toBe(['sleepsuit'])
        ->and(($this->list)(['colors' => ['شیری']]))->toBe(['bodysuit'])
        ->and(($this->list)(['sizes' => ['۳-۶ ماه'], 'colors' => ['صورتی']]))->toBe([]);
});

it('paginates and computes facets before the other filters', function (): void {
    $page = app(ProductRepository::class)->list(new ProductCriteria(categoryId: $this->baby->id, brandIds: [$this->mahno->id], perPage: 1, page: 2));

    expect($page->total)->toBe(2)
        ->and($page->lastPage())->toBe(2)
        ->and($page->items[0]->slug)->toBe('sold-out')
        ->and($page->facets->categoryCounts)->toBe([$this->clothes->id => 3, $this->feeding->id => 1])
        ->and($page->facets->sizes)->toBe(['۰-۳ ماه', '۳-۶ ماه'])
        ->and($page->facets->colors)->toBe([['name' => 'شیری', 'hex' => '#F6EFE6'], ['name' => 'صورتی', 'hex' => null]])
        ->and(array_column($page->facets->brands, 'count', 'slug'))->toEqual(['panberiz' => 2, 'mahno' => 2])
        ->and($page->facets->minPrice?->toToman())->toBe(100_000)
        ->and($page->facets->maxPrice?->toToman())->toBe(485_000);
});

it('builds cards with brand, money and variant flag', function (): void {
    $card = app(ProductRepository::class)->list(new ProductCriteria(categoryId: $this->clothes->id))->items[1];

    expect($card->slug)->toBe('bodysuit')
        ->and($card->brand?->name)->toBe('پوشاک پنبه‌ریز')
        ->and($card->price->formatShort())->toBe('۴۸۵ هزار تومان')
        ->and($card->hasVariants)->toBeTrue()
        ->and($card->rating())->toBeNull()
        ->and(app(CatalogRepository::class)->findCategory('baby-clothes')?->parentId)->toBe($this->baby->id);
});

it('flags filtered criteria and gives a stable cache key', function (): void {
    $a = new ProductCriteria(categoryId: 1, sizes: ['b', 'a', 'a', ' ']);
    $b = new ProductCriteria(categoryId: 1, sizes: ['a', 'b']);

    expect($a->sizes)->toBe(['a', 'b'])
        ->and($a->cacheKey())->toBe($b->cacheKey())
        ->and($a->isFiltered())->toBeTrue()
        ->and((new ProductCriteria(categoryId: 1, sort: ProductSort::Newest, page: 3))->isFiltered())->toBeFalse()
        ->and((new ProductCriteria(perPage: 1000))->perPage)->toBe(ProductCriteria::MAX_PER_PAGE);
});
