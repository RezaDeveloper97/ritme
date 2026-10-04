<?php

declare(strict_types=1);

use App\Domain\Shop\Catalog\Actions\SyncProductCategories;
use App\Domain\Shop\Catalog\Enums\ReviewStatus;
use App\Domain\Shop\Catalog\Models\Brand;
use App\Domain\Shop\Catalog\Models\Category;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductReview;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use App\Http\Controllers\Shop\CategoryController;

/**
 * @return array<string, mixed>
 */
function shopGraph(string $html): array
{
    preg_match('#<script type="application/ld\+json">(.*?)</script>#s', $html, $m);

    return json_decode($m[1] ?? '{}', true, flags: JSON_THROW_ON_ERROR);
}

function shopRobots(string $html): string
{
    preg_match('#<meta name="robots" content="([^"]+)"#', $html, $m);

    return $m[1] ?? '';
}

function shopCanonical(string $html): string
{
    preg_match('#<link rel="canonical" href="([^"]+)"#', $html, $m);

    return $m[1] ?? '';
}

function shopProduct(string $slug, Category $category, int $toman, Brand $brand, array $extra = []): Product
{
    $product = Product::factory()->published()->priced($toman)->create(['slug' => $slug, 'brand_id' => $brand->id, ...$extra]);
    app(SyncProductCategories::class)->handle($product, [$category->id]);

    return $product;
}

beforeEach(function (): void {
    $this->baby = Category::factory()->create(['slug' => 'baby', 'name' => 'سیسمونی و نوزاد', 'intro' => 'لباس، خواب، تغذیه و بیرون رفتن.', 'sort_order' => 1]);
    $this->clothes = Category::factory()->childOf($this->baby)->create(['slug' => 'baby-clothes', 'name' => 'لباس نوزاد', 'sort_order' => 1]);
    $this->feeding = Category::factory()->childOf($this->baby)->create(['slug' => 'feeding', 'name' => 'تغذیه', 'sort_order' => 2]);
    $this->beauty = Category::factory()->create(['slug' => 'beauty', 'name' => 'آرایشی و بهداشتی', 'sort_order' => 2]);
    $this->panberiz = Brand::factory()->create(['name' => 'پوشاک پنبه‌ریز', 'slug' => 'panberiz']);
    $this->mahno = Brand::factory()->create(['name' => 'خانه سیسمونی ماه‌نو', 'slug' => 'mahno']);

    $this->bodysuit = shopProduct('bodysuit', $this->clothes, 485_000, $this->panberiz, ['title' => 'بادی آستین‌بلند نخی', 'illustration' => 'product-bodysuit', 'badge' => 'پنبه ۱۰۰٪']);
    $this->sleepsuit = shopProduct('sleepsuit', $this->clothes, 390_000, $this->panberiz, ['title' => 'سرهمی خواب دکمه‌دار', 'illustration' => 'product-sleepsuit']);
    $this->hat = shopProduct('hat', $this->clothes, 160_000, $this->mahno, ['title' => 'کلاه نوزادی', 'illustration' => 'product-hat', 'stock_qty' => 0]);
    $this->bottle = shopProduct('bottle', $this->feeding, 290_000, $this->mahno, ['title' => 'شیشه شیر', 'illustration' => 'product-bottle']);
    $this->serum = shopProduct('serum', $this->beauty, 420_000, $this->mahno, ['title' => 'سرم آبرسان', 'illustration' => 'product-serum']);

    foreach (['۰-۳ ماه', '۳-۶ ماه'] as $i => $size) {
        ProductVariant::factory()->create(['product_id' => $this->bodysuit->id, 'size' => $size, 'color' => 'شیری', 'color_hex' => '#F6EFE6', 'stock_qty' => 3, 'sort_order' => $i]);
    }
    ProductVariant::factory()->create(['product_id' => $this->sleepsuit->id, 'size' => '۳-۶ ماه', 'color' => 'یاسی', 'color_hex' => '#D9D4F2', 'stock_qty' => 3]);
});

// ---- /shop -------------------------------------------------------------------------------------------------------

it('renders the shop home with one h1, department promos, category tiles, product rows and app CTAs', function (): void {
    config(['app.env' => 'production']); // non-production pages are always noindex
    $html = (string) $this->get('/shop')->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('آنچه برای خودت و کودکت لازم است')
        ->and($html)->toContain('برای آمدن نوزاد آماده شو')                       // promo copy of the `baby` root
        ->and($html)->toContain('href="'.route('shop.category', 'baby').'"')
        ->and($html)->toContain('دسته‌های سیسمونی و نوزاد')
        ->and($html)->toContain('href="'.route('shop.category', 'feeding').'"')     // category tile
        ->and($html)->toContain('href="'.route('shop.product', 'bodysuit').'"')
        ->and($html)->toContain('پوشاک پنبه‌ریز')                                   // brand on the card
        ->and($html)->toContain('۴۸۵ هزار')
        ->and($html)->toContain('لیست سیسمونی')
        ->and($html)->toContain('یادآور خرید قبل از پریود')
        ->and($html)->not->toContain('role="switch"')                                // no inert switch
        ->and($html)->not->toContain('۳۴ از ۸۶')                                     // no fake progress
        ->and($html)->not->toContain('href="'.route('shop.product', 'hat').'"')     // sold out: not a best seller
        ->and($html)->not->toMatch('/\sstyle\s*=/i')
        ->and($html)->not->toMatch('#(src|href)="https?://(?!localhost|127\.0\.0\.1|ritme)#');

    $graph = shopGraph($html)['@graph'];
    $itemList = collect($graph)->firstWhere('@type', 'ItemList');
    expect(array_column($graph, '@type'))->toContain('CollectionPage', 'ItemList', 'BreadcrumbList')
        ->and(array_column($itemList['itemListElement'], 'url'))->toContain(route('shop.product', 'bodysuit'), route('shop.product', 'serum'))
        ->and(shopRobots($html))->not->toContain('noindex');
});

it('never shows invented ratings and shows real review ratings only', function (): void {
    $html = (string) $this->get('/shop')->getContent();
    expect($html)->not->toContain('امتیاز ');

    ProductReview::factory()->create(['product_id' => $this->bodysuit->id, 'rating' => 4, 'status' => ReviewStatus::Approved]);
    ProductReview::factory()->create(['product_id' => $this->sleepsuit->id, 'rating' => 5, 'status' => ReviewStatus::Approved, 'is_demo' => true]);

    $html = (string) $this->get('/shop')->getContent();
    expect(substr_count($html, 'امتیاز ۴ از ۵ از ۱ نظر'))->toBe(1)
        ->and($html)->not->toContain('امتیاز ۵ از ۵');                                // demo review not counted
});

it('labels demo products and keeps a demo-only shop out of the index', function (): void {
    config(['app.env' => 'production']); // non-production pages are always noindex
    Product::query()->update(['is_demo' => true]);

    $html = (string) $this->get('/shop')->assertOk()->getContent();
    $category = (string) $this->get('/shop/category/baby-clothes')->assertOk()->getContent();

    expect($html)->toContain(__('shop.demo_note'))
        ->and(shopRobots($html))->toContain('noindex')
        ->and($category)->toContain(__('shop.demo_note'))
        ->and(shopRobots($category))->toContain('noindex');
});

it('renders an empty shop as a noindex page with one h1', function (): void {
    Product::query()->delete();

    $html = (string) $this->get('/shop')->assertOk()->getContent();
    expect(substr_count($html, '<h1'))->toBe(1)->and(shopRobots($html))->toContain('noindex');
});

it('does not bake the cart count into the cached HTML', function (): void {
    $html = (string) $this->get('/shop')->getContent();

    expect($html)->toContain('data-module="cart-badge"')
        ->and($html)->toMatch('#<span data-cart-count hidden[^>]*></span>#');
});

// ---- /shop/category/{slug} ---------------------------------------------------------------------------------------

it('renders a category with breadcrumbs, subnav, sorts, filters and its subtree products', function (): void {
    config(['app.env' => 'production']); // non-production pages are always noindex
    $html = (string) $this->get('/shop/category/baby')->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('href="'.route('shop.product', 'bottle').'"')      // subcategory product
        ->and($html)->toContain('href="'.route('shop.product', 'hat').'"')         // sold out listed (last)
        ->and($html)->not->toContain('href="'.route('shop.product', 'serum').'"')
        ->and($html)->toContain('ناموجود')
        ->and($html)->toContain('href="'.route('shop.category', 'beauty').'"')     // department switch
        ->and($html)->toContain('href="'.route('shop.category', 'feeding').'"')
        ->and($html)->toContain('?sort=newest"')
        ->and($html)->toContain('name="brand[]" value="panberiz"')
        ->and($html)->toContain('name="size[]" value="۳-۶ ماه"')
        ->and($html)->toContain('fill="#F6EFE6"')                                    // swatch without inline style
        ->and($html)->toContain('name="stock" value="1"')
        ->and($html)->toContain('method="get"')
        ->and($html)->not->toMatch('/\sstyle\s*=/i')
        ->and(shopRobots($html))->not->toContain('noindex')
        ->and(shopCanonical($html))->toEndWith('/shop/category/baby');

    $graph = shopGraph($html)['@graph'];
    $breadcrumb = collect($graph)->firstWhere('@type', 'BreadcrumbList');
    $itemList = collect($graph)->firstWhere('@type', 'ItemList');
    expect(array_column($graph, '@type'))->toContain('CollectionPage', 'ItemList', 'BreadcrumbList')
        ->and(array_column($breadcrumb['itemListElement'], 'name'))->toBe(['خانه', 'فروشگاه', 'سیسمونی و نوزاد'])
        ->and($itemList['numberOfItems'])->toBe(4);
});

it('filters by brand, size, price and stock with noindex,follow and the canonical on the category', function (): void {
    config(['app.env' => 'production']); // non-production pages are always noindex
    $url = '/shop/category/baby-clothes?brand%5B%5D=panberiz&size%5B%5D='.rawurlencode('۰-۳ ماه');
    $html = (string) $this->get($url)->assertOk()->getContent();

    expect($html)->toContain('href="'.route('shop.product', 'bodysuit').'"')
        ->and($html)->not->toContain('href="'.route('shop.product', 'sleepsuit').'"')
        ->and($html)->toContain('۱ کالا با فیلترهای فعلی')
        ->and($html)->toContain('حذف فیلتر پوشاک پنبه‌ریز')
        ->and(shopRobots($html))->toBe('noindex,follow')
        ->and(shopCanonical($html))->toEndWith('/shop/category/baby-clothes');

    $cheap = (string) $this->get('/shop/category/baby-clothes?max=200000')->getContent();
    expect($cheap)->toContain('href="'.route('shop.product', 'hat').'"')
        ->and($cheap)->not->toContain('href="'.route('shop.product', 'bodysuit').'"');

    $stock = (string) $this->get('/shop/category/baby-clothes?stock=1')->getContent();
    expect($stock)->not->toContain('href="'.route('shop.product', 'hat').'"');

    $sorted = (string) $this->get('/shop/category/baby-clothes?sort=cheapest')->getContent();
    expect(shopRobots($sorted))->toBe('noindex,follow')
        ->and(strpos($sorted, route('shop.product', 'sleepsuit')))->toBeLessThan(strpos($sorted, route('shop.product', 'bodysuit')));
});

it('301s every non-canonical query to one canonical URL', function (string $query, string $target): void {
    $this->get('/shop/category/baby-clothes'.$query)->assertStatus(301)
        ->assertHeader('Location', route('shop.category', 'baby-clothes').$target);
})->with([
    'page 1' => ['?page=1', ''],
    'default sort' => ['?sort=best-selling', ''],
    'empty form fields' => ['?brand%5B%5D=panberiz&min=&max=&sort=best-selling', '?brand%5B%5D=panberiz'],
    'scalar list' => ['?brand=panberiz', '?brand%5B%5D=panberiz'],
    'unknown brand' => ['?brand%5B%5D=nope', ''],
    'unknown size' => ['?size%5B%5D=xxl', ''],
    'order' => ['?stock=1&brand%5B%5D=panberiz', '?brand%5B%5D=panberiz&stock=1'],
    'sorted brands' => ['?brand%5B%5D=panberiz&brand%5B%5D=mahno', '?brand%5B%5D=mahno&brand%5B%5D=panberiz'],
    'persian digits' => ['?min=%DB%B1%DB%B0%DB%B0', '?min=100'],
    'swapped prices' => ['?min=500&max=100', '?min=100&max=500'],
    'unknown param' => ['?foo=bar', ''],
    'invalid sort' => ['?sort=random', ''],
]);

it('keeps tracking parameters on the redirect', function (): void {
    $this->get('/shop/category/baby-clothes?page=1&utm_source=telegram')->assertStatus(301)
        ->assertHeader('Location', route('shop.category', 'baby-clothes').'?utm_source=telegram');
});

it('answers 404 for unknown categories and invalid or out-of-range pages', function (string $url): void {
    $this->get($url)->assertNotFound();
})->with(['/shop/category/nope', '/shop/category/baby-clothes?page=0', '/shop/category/baby-clothes?page=x', '/shop/category/baby-clothes?page=3']);

it('paginates with self-canonical, indexable pages', function (): void {
    config(['app.env' => 'production']); // non-production pages are always noindex
    foreach (range(1, CategoryController::PER_PAGE + 2) as $i) {
        shopProduct("extra-{$i}", $this->feeding, 100_000 + $i, $this->mahno, ['title' => "محصول {$i}"]);
    }

    $first = (string) $this->get('/shop/category/feeding')->assertOk()->getContent();
    $second = (string) $this->get('/shop/category/feeding?page=2')->assertOk()->getContent();

    expect($first)->toContain('href="'.route('shop.category', 'feeding').'?page=2"')
        ->and($second)->toContain('صفحه ۲')
        ->and(shopCanonical($second))->toEndWith('/shop/category/feeding?page=2')
        ->and(shopRobots($second))->not->toContain('noindex');
});

it('shows an empty state with a reset link when filters match nothing', function (): void {
    $html = (string) $this->get('/shop/category/baby-clothes?min=900000')->assertOk()->getContent();

    expect($html)->toContain(__('shop.category.empty'))
        ->and($html)->toContain(__('shop.category.reset'))
        ->and(shopRobots($html))->toContain('noindex');
});

it('reads the listing from the cache on a warm request', function (): void {
    $this->get('/shop/category/baby-clothes?brand%5B%5D=panberiz')->assertOk();

    DB::enableQueryLog();
    $this->get('/shop/category/baby-clothes?brand%5B%5D=panberiz')->assertOk();
    $shopQueries = array_filter(DB::getQueryLog(), static fn (array $q): bool => str_contains($q['query'], 'shop_'));

    expect($shopQueries)->toBe([]);
});
