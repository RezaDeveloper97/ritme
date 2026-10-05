<?php

declare(strict_types=1);

use App\Domain\Shop\Cart\Data\ShippingRule;
use App\Domain\Shop\Cart\Repositories\SessionCartRepository;
use App\Domain\Shop\Catalog\Models\Brand;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use App\Http\Controllers\Shop\CartController;
use App\Http\Middleware\PageCache;
use App\Support\Money\Money;
use Database\Seeders\SettingsSeeder;
use Illuminate\Testing\TestResponse;

beforeEach(function (): void {
    $this->seed(SettingsSeeder::class);

    $brand = Brand::factory()->create(['name' => 'پوشاک پنبه‌ریز', 'slug' => 'panberiz']);
    $this->bodysuit = Product::factory()->published()->priced(485_000)->create([
        'title' => 'بادی آستین‌بلند نخی', 'slug' => 'bodysuit', 'brand_id' => $brand->id, 'illustration' => 'product-bodysuit',
    ]);
    // ۳-۶ ماه · شیری: 4 left · ۰-۳ ماه · شیری: sold out · ۳-۶ ماه · صورتی: own price 500k, 2 left.
    $this->milk = ProductVariant::factory()->create(['product_id' => $this->bodysuit->id, 'size' => '۳-۶ ماه', 'color' => 'شیری', 'stock_qty' => 4, 'sort_order' => 1]);
    $this->soldOut = ProductVariant::factory()->create(['product_id' => $this->bodysuit->id, 'size' => '۰-۳ ماه', 'color' => 'شیری', 'stock_qty' => 0, 'sort_order' => 2]);
    $this->pink = ProductVariant::factory()->create(['product_id' => $this->bodysuit->id, 'size' => '۳-۶ ماه', 'color' => 'صورتی', 'stock_qty' => 2, 'sort_order' => 3, 'price' => Money::fromToman(500_000)]);

    $this->pads = Product::factory()->published()->priced(98_000)->create(['title' => 'نوار بهداشتی روزانه', 'slug' => 'pads', 'stock_qty' => 30, 'stock_status' => 'in_stock', 'illustration' => 'product-socks']);
    $this->socks = Product::factory()->published()->priced(140_000)->create(['title' => 'جوراب نوزادی', 'slug' => 'socks', 'stock_qty' => 5, 'stock_status' => 'in_stock']);
});

function addBodysuit(mixed $test, array $extra = []): TestResponse
{
    return $test->post('/shop/cart', ['product' => $test->bodysuit->id, 'size' => '۳-۶ ماه', 'color' => 'شیری', 'quantity' => '1', ...$extra]);
}

function cartLineKey(mixed $test, ?ProductVariant $variant = null, ?Product $product = null): string
{
    return 'p'.($product ?? $test->bodysuit)->id.($variant === null ? '' : '-v'.$variant->id);
}

// ---- page ----------------------------------------------------------------------------------------------------------

it('renders the empty cart: one h1, noindex, no-store, never page-cached, suggestions, no inline styles', function (): void {
    config(['pagecache.enabled' => true]);

    $first = $this->get('/shop/cart')->assertOk();
    $html = (string) $first->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('سبد خریدت خالی است')
        ->and($html)->toContain('شاید لازم داشته باشی')
        ->and($html)->toContain('href="'.route('shop.product', 'pads').'"')
        ->and($html)->toMatch('#<meta name="robots" content="noindex#')
        ->and($html)->not->toMatch('/\sstyle\s*=/i')
        ->and($first->headers->get('Cache-Control'))->toContain('no-store')
        ->and($first->headers->get(PageCache::HEADER))->not->toBe('HIT');

    expect($this->get('/shop/cart')->headers->get(PageCache::HEADER))->not->toBe('HIT');
});

// ---- add (no JS) ---------------------------------------------------------------------------------------------------

it('adds a variant without JS: PRG to the cart, ids + quantities in the session, plain count cookie, live prices', function (): void {
    addBodysuit($this, ['quantity' => '۲', 'price' => '1', 'unit_price' => '1'])
        ->assertStatus(303)->assertRedirect('/shop/cart')
        ->assertSessionHas('cart_status', 'بادی آستین‌بلند نخی به سبد اضافه شد.')
        ->assertPlainCookie(CartController::COUNT_COOKIE, '2');

    expect(session(SessionCartRepository::KEY))->toBe(['v' => 1, 'l' => [[$this->bodysuit->id, $this->milk->id, 2, 4_850_000]]]);

    $html = (string) $this->get('/shop/cart')->assertOk()->assertPlainCookie(CartController::COUNT_COOKIE, '2')->getContent();
    expect($html)->toContain('بادی آستین‌بلند نخی')
        ->and($html)->toContain('۳-۶ ماه · شیری')
        ->and($html)->toContain('۹۷۰ هزار')                    // 2 × 485k from the catalog, client «price» ignored
        ->and($html)->toContain('جمع کالاها (۲)')
        ->and($html)->toContain('محاسبه در مرحله بعد')
        ->and($html)->toContain('action="'.route('shop.cart.update', [cartLineKey($this, $this->milk)]).'"')
        ->and($html)->toContain('name="quantity" value="3"')
        ->and($html)->toContain('href="'.route('shop.checkout').'"')
        ->and($html)->not->toMatch('/\sstyle\s*=/i')
        ->and(substr_count($html, '<h1'))->toBe(1);
});

it('merges repeated adds of the same variant into one line priced from the variant', function (): void {
    addBodysuit($this, ['color' => 'صورتی']);
    addBodysuit($this, ['color' => 'صورتی'])->assertPlainCookie(CartController::COUNT_COOKIE, '2');

    expect(session(SessionCartRepository::KEY)['l'])->toBe([[$this->bodysuit->id, $this->pink->id, 2, 5_000_000]]);
});

it('clamps an add to the live stock and refuses more once everything is in the cart', function (): void {
    addBodysuit($this, ['quantity' => '9'])
        ->assertRedirect('/shop/cart')
        ->assertSessionHas('cart_status', 'از بادی آستین‌بلند نخی فقط ۴ عدد موجود است؛ تعداد در سبد ۴ شد.')
        ->assertPlainCookie(CartController::COUNT_COOKIE, '4');

    addBodysuit($this)
        ->assertRedirect(route('shop.product', 'bodysuit').'#buy')
        ->assertSessionHas('cart_error', 'همه موجودی قابل خرید بادی آستین‌بلند نخی (۴ عدد) در سبدت است.');

    expect(session(SessionCartRepository::KEY)['l'][0][2])->toBe(4);
});

it('refuses sold-out, unknown and missing variants, unknown products and junk input without touching the cart', function (array $input, string $message): void {
    $this->post('/shop/cart', ['product' => $this->bodysuit->id, ...$input])
        ->assertStatus(303)
        ->assertSessionHas('cart_error', $message);

    expect(session(SessionCartRepository::KEY))->toBeNull();
})->with([
    'sold out' => [['size' => '۰-۳ ماه', 'color' => 'شیری'], 'بادی آستین‌بلند نخی در حال حاضر ناموجود است.'],
    'unknown combination' => [['size' => '۶-۹ ماه', 'color' => 'شیری'], 'این ترکیب سایز و رنگ برای بادی آستین‌بلند نخی وجود ندارد.'],
    'no variant chosen' => [[], 'لطفاً سایز و رنگ بادی آستین‌بلند نخی را انتخاب کن.'],
    'quantity too large' => [['size' => '۳-۶ ماه', 'color' => 'شیری', 'quantity' => '11'], 'درخواست معتبر نبود. دوباره امتحان کن.'],
    'quantity junk' => [['size' => '۳-۶ ماه', 'color' => 'شیری', 'quantity' => 'abc'], 'درخواست معتبر نبود. دوباره امتحان کن.'],
]);

it('refuses unpublished products and products out of stock', function (): void {
    $hidden = Product::factory()->priced(10_000)->create(['slug' => 'hidden', 'stock_qty' => 3, 'stock_status' => 'in_stock']);
    $gone = Product::factory()->published()->outOfStock()->priced(10_000)->create(['slug' => 'gone', 'title' => 'کالای تمام‌شده']);

    $this->post('/shop/cart', ['product' => $hidden->id])->assertSessionHas('cart_error', 'این کالا دیگر در فروشگاه نیست.');
    $this->post('/shop/cart', ['product' => $gone->id])->assertSessionHas('cart_error', 'کالای تمام‌شده در حال حاضر ناموجود است.');
    $this->post('/shop/cart', ['product' => $this->pads->id])->assertSessionHas('cart_status');

    expect(session(SessionCartRepository::KEY)['l'])->toBe([[$this->pads->id, null, 1, 980_000]]);
});

// ---- JSON (cart module) --------------------------------------------------------------------------------------------

it('answers JSON for the cart module, 422 for a refused add', function (): void {
    $this->postJson('/shop/cart', ['product' => $this->pads->id, 'quantity' => 2])
        ->assertOk()
        ->assertJson(['ok' => true, 'count' => 2, 'quantity' => 2, 'message' => 'نوار بهداشتی روزانه به سبد اضافه شد.', 'cart_url' => route('shop.cart')])
        ->assertPlainCookie(CartController::COUNT_COOKIE, '2');

    $this->postJson('/shop/cart', ['product' => $this->bodysuit->id, 'size' => '۰-۳ ماه', 'color' => 'شیری'])
        ->assertStatus(422)
        ->assertJson(['ok' => false, 'count' => 2, 'message' => 'بادی آستین‌بلند نخی در حال حاضر ناموجود است.']);

    $key = cartLineKey($this, null, $this->pads);
    $this->postJson('/shop/cart/'.$key, ['quantity' => '3'])->assertOk()->assertJson(['ok' => true, 'count' => 3, 'quantity' => 3]);
    $this->postJson('/shop/cart/'.$key.'/remove')->assertOk()->assertJson(['ok' => true, 'count' => 0, 'quantity' => 0])
        ->assertCookieExpired(CartController::COUNT_COOKIE);
});

// ---- update / remove (no JS) ---------------------------------------------------------------------------------------

it('updates quantities with the − / + submit buttons, clamped to the live stock, and removes lines', function (): void {
    addBodysuit($this);
    $key = cartLineKey($this, $this->milk);

    $this->post('/shop/cart/'.$key, ['quantity' => '3'])->assertRedirect('/shop/cart')
        ->assertSessionHas('cart_status', 'تعداد بادی آستین‌بلند نخی به‌روز شد.')->assertPlainCookie(CartController::COUNT_COOKIE, '3');
    $this->post('/shop/cart/'.$key, ['quantity' => '8'])->assertSessionHas('cart_status', 'از بادی آستین‌بلند نخی فقط ۴ عدد موجود است؛ تعداد در سبد ۴ شد.');
    expect(session(SessionCartRepository::KEY)['l'][0][2])->toBe(4);

    $this->post('/shop/cart/'.$key, ['quantity' => '-1'])->assertSessionHas('cart_error', 'درخواست معتبر نبود. دوباره امتحان کن.');
    $this->post('/shop/cart/p999-v1', ['quantity' => '1'])->assertSessionHas('cart_error', 'این کالا در سبدت نبود.');

    $this->post('/shop/cart/'.$key.'/remove')->assertRedirect('/shop/cart')
        ->assertSessionHas('cart_status', 'کالا از سبد حذف شد.')->assertCookieExpired(CartController::COUNT_COOKIE);
    expect(session(SessionCartRepository::KEY))->toBeNull();

    $this->post('/shop/cart/bad-key/remove')->assertNotFound();
});

// ---- re-validation on read -----------------------------------------------------------------------------------------

it('re-checks stock, price and availability live on every read', function (): void {
    addBodysuit($this, ['quantity' => '4']);
    $this->post('/shop/cart', ['product' => $this->pads->id]);
    $this->post('/shop/cart', ['product' => $this->socks->id]);

    // Behind the cart's back: stock drops, the price changes, a product is unpublished.
    $this->milk->update(['stock_qty' => 2]);
    $this->pads->update(['price' => Money::fromToman(120_000)]);
    $this->socks->update(['is_published' => false]);

    $html = (string) $this->get('/shop/cart')->assertOk()->assertPlainCookie(CartController::COUNT_COOKIE, '3')->getContent();

    expect($html)->toContain('از بادی آستین‌بلند نخی فقط ۲ عدد موجود است؛ تعداد در سبد ۲ شد.')
        ->and($html)->toContain('قیمت نوار بهداشتی روزانه از ۹۸ هزار تومان به ۱۲۰ هزار تومان تغییر کرد.')
        ->and($html)->toContain('یکی از کالاها دیگر در فروشگاه نیست و از سبد حذف شد.')
        ->and($html)->not->toContain('جوراب نوزادی</a>')
        ->and($html)->toContain('۱٬۰۹۰ هزار تومان'); // 2 × 485k + 120k

    expect(session(SessionCartRepository::KEY)['l'])->toBe([
        [$this->bodysuit->id, $this->milk->id, 2, 4_850_000],
        [$this->pads->id, null, 1, 1_200_000],
    ]);

    // Sold out now: kept, flagged, out of the totals and the count.
    $this->milk->update(['stock_qty' => 0]);
    $html = (string) $this->get('/shop/cart')->assertPlainCookie(CartController::COUNT_COOKIE, '1')->getContent();
    expect($html)->toContain('ناموجود')
        ->and($html)->toContain('جمع کالاها (۱)')
        ->and($html)->toContain('بادی آستین‌بلند نخی در حال حاضر ناموجود است.');
});

it('ignores a tampered session payload', function (): void {
    $this->withSession([SessionCartRepository::KEY => ['v' => 1, 'l' => [['x'], [$this->pads->id, null, 999, -5], [$this->pads->id, 'v', 1, 1]]]])
        ->get('/shop/cart')->assertOk()->assertPlainCookie(CartController::COUNT_COOKIE, '10');

    expect(session(SessionCartRepository::KEY)['l'])->toBe([[$this->pads->id, null, 10, 980_000]]);
});

it('shows the free-shipping bar and a known fee from the shipping rule', function (): void {
    app()->instance(ShippingRule::class, ShippingRule::fromToman(45_000, 1_000_000));
    $this->post('/shop/cart', ['product' => $this->pads->id, 'quantity' => 2]);

    $html = (string) $this->get('/shop/cart')->getContent();
    expect($html)->toContain('تا ارسال رایگان: ۸۰۴ هزار تومان')
        ->and($html)->toContain('ارسال رایگان برای خرید از ۱٬۰۰۰ هزار تومان')
        ->and($html)->toContain('۴۵ هزار تومان')
        ->and($html)->toContain('fill-stage-teen');
});

it('posts the product page form to the cart', function (): void {
    $html = (string) $this->get('/shop/product/bodysuit')->assertOk()->getContent();

    expect($html)->toContain('action="'.route('shop.cart.add').'"')
        ->and($html)->toContain('data-cart-toast');

    addBodysuit($this, ['size' => '۰-۳ ماه']);
    expect((string) $this->get('/shop/product/bodysuit')->getContent())->toContain('بادی آستین‌بلند نخی در حال حاضر ناموجود است.');
});
