<?php

declare(strict_types=1);

use App\Domain\Shop\Catalog\Actions\SyncCrossSells;
use App\Domain\Shop\Catalog\Actions\SyncProductCategories;
use App\Domain\Shop\Catalog\Enums\ReviewStatus;
use App\Domain\Shop\Catalog\Models\Brand;
use App\Domain\Shop\Catalog\Models\Category;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductReview;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use App\Http\Controllers\Shop\ProductController;
use App\Http\Middleware\PageCache;
use Database\Seeders\SettingsSeeder;
use Illuminate\Support\Facades\RateLimiter;

/**
 * @return array<string, mixed>|null
 */
function productPageNode(string $html, string $type): ?array
{
    preg_match('#<script type="application/ld\+json">(.*?)</script>#s', $html, $m);
    $graph = json_decode($m[1] ?? '{}', true, flags: JSON_THROW_ON_ERROR)['@graph'] ?? [];

    return collect($graph)->firstWhere('@type', $type);
}

function productPageRobots(string $html): string
{
    preg_match('#<meta name="robots" content="([^"]+)"#', $html, $m);

    return $m[1] ?? '';
}

beforeEach(function (): void {
    $this->seed(SettingsSeeder::class);
    config(['app.env' => 'production']); // non-production pages are always noindex

    $this->baby = Category::factory()->create(['slug' => 'baby', 'name' => 'سیسمونی و نوزاد', 'sort_order' => 1]);
    $this->clothes = Category::factory()->childOf($this->baby)->create(['slug' => 'baby-clothes', 'name' => 'لباس نوزاد', 'sort_order' => 1]);
    $this->brand = Brand::factory()->create(['name' => 'پوشاک پنبه‌ریز', 'slug' => 'panberiz']);

    $this->product = Product::factory()->published()->priced(485_000)->create([
        'title' => 'بادی آستین‌بلند نخی', 'slug' => 'bodysuit', 'sku' => 'BODY-1', 'brand_id' => $this->brand->id,
        'primary_category_id' => $this->clothes->id, 'illustration' => 'product-bodysuit',
        'short_description' => 'بادی پنبه‌ای نرم با دکمه فشاری.',
        'specs' => [['label' => 'جنس', 'value' => 'پنبه ۱۰۰٪']],
        'size_chart' => ['columns' => ['سایز', 'قد (سانت)'], 'rows' => [['۰-۳ ماه', '۵۶ تا ۶۱'], ['۳-۶ ماه', '۶۱ تا ۶۷']]],
    ]);
    app(SyncProductCategories::class)->handle($this->product, [$this->clothes->id]);

    // ۰-۳ sold out in شیری but in stock in صورتی; ۳-۶ in stock in both; ۶-۹ sold out everywhere.
    $rows = [['۰-۳ ماه', 'شیری', '#F6EFE6', 0], ['۳-۶ ماه', 'شیری', '#F6EFE6', 4], ['۶-۹ ماه', 'شیری', '#F6EFE6', 0],
        ['۰-۳ ماه', 'صورتی', '#F4D6DE', 2], ['۳-۶ ماه', 'صورتی', '#F4D6DE', 3], ['۶-۹ ماه', 'صورتی', '#F4D6DE', 0]];
    foreach ($rows as $i => [$size, $color, $hex, $qty]) {
        ProductVariant::factory()->create(['product_id' => $this->product->id, 'size' => $size, 'color' => $color, 'color_hex' => $hex, 'stock_qty' => $qty, 'sort_order' => $i, 'sku' => 'BODY-1-'.$i]);
    }

    $this->socks = Product::factory()->published()->priced(140_000)->create(['title' => 'جوراب نوزادی', 'slug' => 'socks', 'brand_id' => $this->brand->id, 'illustration' => 'product-socks']);
    app(SyncCrossSells::class)->handle($this->product, [$this->socks->id]);
});

// ---- page --------------------------------------------------------------------------------------------------------

it('renders the product page with one h1, every block, no inline styles and the subnav', function (): void {
    $html = (string) $this->get('/shop/product/bodysuit')->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('بادی آستین‌بلند نخی</h1>')
        ->and($html)->toContain('پوشاک پنبه‌ریز')
        ->and($html)->toContain('۴۸۵ هزار')
        ->and($html)->toContain('مشخصات')
        ->and($html)->toContain('پنبه ۱۰۰٪')
        ->and($html)->toContain('id="size"')
        ->and($html)->toContain('<caption class="sr-only">')
        ->and($html)->toContain('نظر خریداران')
        ->and($html)->toContain('معمولاً با این می‌خرند')
        ->and($html)->toContain('href="'.route('shop.product', 'socks').'"')
        ->and($html)->toContain('href="'.route('shop.category', 'baby-clothes').'"')
        ->and($html)->toContain('data-module="product"')
        ->and($html)->not->toMatch('/\sstyle\s*=/i')
        ->and($html)->not->toContain('href="#"')
        ->and($html)->not->toContain('فروشنده بررسی‌شده')
        ->and(productPageRobots($html))->toStartWith('index');
});

it('renders the variant picker as radio inputs inside the add-to-cart form, disabling only sizes sold out everywhere', function (): void {
    $html = (string) $this->get('/shop/product/bodysuit')->getContent();

    preg_match('#<form[^>]*data-cart-slot.*?</form>#s', $html, $form);
    $form = $form[0] ?? '';

    // First variant in stock (۳-۶ ماه · شیری) is preselected.
    expect($form)->toMatch('/name="color" value="شیری"[^>]*checked/')
        ->and($form)->toMatch('/name="size" value="۳-۶ ماه"[^>]*checked/')
        ->and($form)->toMatch('/name="size" value="۶-۹ ماه"[^>]*disabled/')       // sold out in every colour
        ->and($form)->not->toMatch('/name="size" value="۰-۳ ماه"[^>]*disabled/')   // still available in صورتی
        ->and($form)->toContain('data-soldout')                                    // …but struck for شیری
        ->and($form)->toContain('name="quantity"')
        ->and($form)->toContain('fill="#F4D6DE"')                                  // swatch via SVG fill
        ->and($html)->toContain('۳-۶ ماه: قد ۶۱ تا ۶۷ سانت')                      // size hint from the chart
        ->and($html)->toMatch('/data-size-row="۳-۶ ماه" data-current/');

    // No cart route yet (L6-04): a disabled button in a marked slot, no POST target.
    expect($form)->toMatch('/<button type="submit" data-cart-submit\s+disabled/')
        ->and($form)->not->toContain('method="post"');
});

it('emits Product JSON-LD with an AggregateOffer of per-variant offers, availability from stock and the breadcrumb', function (): void {
    $html = (string) $this->get('/shop/product/bodysuit')->getContent();
    $product = productPageNode($html, 'Product');
    $offers = $product['offers'];

    expect($product['name'])->toBe('بادی آستین‌بلند نخی')
        ->and($product['sku'])->toBe('BODY-1')
        ->and($product['brand'])->toBe(['@type' => 'Brand', 'name' => 'پوشاک پنبه‌ریز'])
        ->and($product)->not->toHaveKey('aggregateRating')
        ->and($product)->not->toHaveKey('review')
        ->and($offers['@type'])->toBe('AggregateOffer')
        ->and($offers['lowPrice'])->toBe(4_850_000)
        ->and($offers['priceCurrency'])->toBe('IRR')
        ->and($offers['offerCount'])->toBe(6)
        ->and(array_column($offers['offers'], 'availability'))->toBe([
            'https://schema.org/OutOfStock', 'https://schema.org/InStock', 'https://schema.org/OutOfStock',
            'https://schema.org/InStock', 'https://schema.org/InStock', 'https://schema.org/OutOfStock',
        ])
        ->and($offers['offers'][0])->toHaveKeys(['sku', 'price', 'priceValidUntil', 'url', 'seller'])
        ->and(productPageNode($html, 'ItemPage')['mainEntity']['@id'])->toBe($product['@id'])
        ->and(array_column(productPageNode($html, 'BreadcrumbList')['itemListElement'], 'name'))
        ->toBe(['خانه', 'فروشگاه', 'سیسمونی و نوزاد', 'لباس نوزاد', 'بادی آستین‌بلند نخی'])
        ->and($html)->toContain('<meta property="og:type" content="product">')
        ->and($html)->toContain('<meta property="product:price:amount" content="4850000">')
        ->and($html)->toContain('<meta property="product:price:currency" content="IRR">');
});

it('uses a single Offer for a product without variants', function (): void {
    $html = (string) $this->get('/shop/product/socks')->assertOk()->getContent();
    $offers = productPageNode($html, 'Product')['offers'];

    expect($offers['@type'])->toBe('Offer')
        ->and($offers['price'])->toBe(1_400_000)
        ->and($offers['availability'])->toBe('https://schema.org/InStock')
        ->and($html)->not->toContain('name="size"');
});

it('adds aggregateRating and reviews only from real approved reviews; demo samples are labelled and not counted', function (): void {
    ProductReview::factory()->approved()->create(['product_id' => $this->product->id, 'author_name' => 'مادر نیلا', 'rating' => 4, 'body' => 'نرم و خوش‌دوخت است.', 'variant_label' => 'سایز ۳-۶ ماه']);
    ProductReview::factory()->approved()->demo()->create(['product_id' => $this->product->id, 'author_name' => 'مادر نمونه', 'rating' => 5, 'body' => 'نظر نمونه طراحی.']);
    ProductReview::factory()->create(['product_id' => $this->product->id, 'author_name' => 'در انتظار', 'body' => 'هنوز بررسی نشده.']);

    $html = (string) $this->get('/shop/product/bodysuit')->getContent();
    $product = productPageNode($html, 'Product');

    expect($product['aggregateRating']['ratingValue'])->toEqual(4)
        ->and($product['aggregateRating']['ratingCount'])->toBe(1)
        ->and(array_column(array_column($product['review'], 'author'), 'name'))->toBe(['مادر نیلا'])
        ->and($html)->toContain('مادر نمونه')
        ->and($html)->toContain('سایز ۳-۶ ماه')
        ->and($html)->toContain('نمونه</div>')
        ->and($html)->toContain('در امتیاز کالا حساب نمی‌شوند')
        ->and($html)->not->toContain('هنوز بررسی نشده.');
});

it('noindexes demo products', function (): void {
    $this->product->update(['is_demo' => true]);

    $html = (string) $this->get('/shop/product/bodysuit')->getContent();
    expect(productPageRobots($html))->toContain('noindex')
        ->and($html)->toContain('نمونه نمایشی فروشگاه');
});

it('301s an old slug to the current one (query kept) and 404s unknown or unpublished products', function (): void {
    $this->product->update(['slug' => 'cotton-bodysuit']);

    $this->get('/shop/product/bodysuit?utm_source=x')->assertStatus(301)
        ->assertHeader('Location', route('shop.product', 'cotton-bodysuit').'?utm_source=x');
    $this->get('/shop/product/nope')->assertNotFound();

    $this->socks->update(['is_published' => false]);
    $this->get('/shop/product/socks')->assertNotFound();
});

it('paginates approved reviews: ?page=1 301s, pages past the end 404, page 2 has its own title', function (): void {
    ProductReview::factory()->approved()->count(ProductController::REVIEWS_PER_PAGE + 1)->create(['product_id' => $this->product->id, 'body' => 'نظر واقعی خریدار.']);

    $first = (string) $this->get('/shop/product/bodysuit')->getContent();
    expect($first)->toContain('href="'.route('shop.product', ['bodysuit', 'page' => 2]).'#reviews"');

    $this->get('/shop/product/bodysuit?page=1')->assertStatus(301)->assertHeader('Location', route('shop.product', 'bodysuit'));
    $this->get('/shop/product/bodysuit?page=3')->assertNotFound();
    $this->get('/shop/product/bodysuit?page=x')->assertNotFound();
    expect((string) $this->get('/shop/product/bodysuit?page=2')->getContent())->toContain('نظرها، صفحه ۲');
});

it('serves warm requests from the page cache with a per-visitor CSRF token in the review form', function (): void {
    $this->get('/shop/product/bodysuit');
    $hit = $this->get('/shop/product/bodysuit');

    expect($hit->headers->get(PageCache::HEADER))->toBe('HIT')
        ->and((string) $hit->getContent())->toMatch('/name="_token" value="[A-Za-z0-9]{20,}"/')
        ->and((string) $hit->getContent())->toContain('action="'.route('shop.product.review', 'bodysuit').'"');
});

// ---- review form -------------------------------------------------------------------------------------------------

it('stores a submitted review as pending with the chosen size and thanks the visitor', function (): void {
    $this->post('/shop/product/bodysuit/reviews', [
        'author_name' => 'مادر سارا', 'rating' => '4', 'body' => 'پارچه نرم است و خوب شسته می‌شود.', 'size' => '۳-۶ ماه',
    ])->assertRedirect(route('shop.product', 'bodysuit').'#review-form')->assertSessionHas('review_submitted', true);

    $review = ProductReview::query()->sole();
    expect($review->status)->toBe(ReviewStatus::Pending)
        ->and($review->author_name)->toBe('مادر سارا')
        ->and($review->variant_label)->toBe('سایز ۳-۶ ماه')
        ->and($review->is_verified_purchase)->toBeFalse()
        ->and($review->ip_hash)->not->toBeNull();

    $this->followingRedirects()->get('/shop/product/bodysuit')->assertSee('نظرت ثبت شد')->assertDontSee('پارچه نرم است و خوب شسته می‌شود.');
});

it('sends invalid reviews back with Persian errors and the old input', function (): void {
    $this->post('/shop/product/bodysuit/reviews', ['author_name' => 'م', 'rating' => '9', 'body' => 'کوتاه', 'size' => 'XXL'])
        ->assertRedirect(route('shop.product', 'bodysuit').'#review-form')
        ->assertSessionHasErrors(['author_name', 'rating', 'body', 'size'])
        ->assertSessionHasInput('body', 'کوتاه');

    $html = (string) $this->get('/shop/product/bodysuit')->getContent();
    expect($html)->toContain('لطفاً موارد مشخص‌شده را اصلاح کن.')
        ->and($html)->toContain('یک امتیاز از ۱ تا ۵ انتخاب کن.')
        ->and(ProductReview::query()->count())->toBe(0);
});

it('answers the honeypot like a real submit without storing anything', function (): void {
    $this->post('/shop/product/bodysuit/reviews', [
        'author_name' => 'ربات', 'rating' => '5', 'body' => 'متن تبلیغاتی بسیار طولانی', ProductController::HONEYPOT => 'https://spam.example',
    ])->assertRedirect(route('shop.product', 'bodysuit').'#review-form')->assertSessionHas('review_submitted', true);

    expect(ProductReview::query()->count())->toBe(0);
});

it('rate limits review posts per IP and 404s reviews for unknown products', function (): void {
    RateLimiter::clear('');
    $payload = ['author_name' => 'مادر سارا', 'rating' => '5', 'body' => 'تجربه خوبی بود، ممنون.'];

    for ($i = 0; $i < ProductController::REVIEWS_PER_10_MINUTES; $i++) {
        $this->post('/shop/product/bodysuit/reviews', $payload)->assertRedirect();
    }
    $this->post('/shop/product/bodysuit/reviews', $payload)->assertStatus(429);

    $this->withServerVariables(['REMOTE_ADDR' => '10.0.0.9'])->post('/shop/product/nope/reviews', $payload)->assertNotFound();
});
