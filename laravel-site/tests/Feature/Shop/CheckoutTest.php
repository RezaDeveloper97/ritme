<?php

declare(strict_types=1);

use App\Domain\Contact\Support\FormTimer;
use App\Domain\Settings\Actions\UpdateSettings;
use App\Domain\Settings\Enums\SettingGroup;
use App\Domain\Shop\Cart\Contracts\CartCatalog;
use App\Domain\Shop\Cart\Repositories\SessionCartRepository;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use App\Domain\Shop\Ordering\Enums\DeliveryWindow;
use App\Domain\Shop\Ordering\Enums\OrderStatus;
use App\Domain\Shop\Ordering\Models\Order;
use App\Domain\Shop\Ordering\Support\CheckoutSession;
use App\Domain\Shop\Ordering\Support\DeliverySlots;
use App\Domain\Shop\Ordering\Support\OrderCode;
use App\Domain\Shop\Payment\Enums\PaymentStatus;
use App\Http\Middleware\PageCache;
use App\Notifications\OrderPlacedForCustomer;
use App\Notifications\OrderPlacedForTeam;
use App\Support\Money\Money;
use Carbon\CarbonImmutable;
use Database\Seeders\SettingsSeeder;
use Illuminate\Notifications\AnonymousNotifiable;
use Illuminate\Support\Facades\Notification;
use Illuminate\Testing\TestResponse;

const CHECKOUT_NOW = '2026-10-10 10:00:00';

beforeEach(function (): void {
    $this->travelTo(CarbonImmutable::parse(CHECKOUT_NOW, 'Asia/Tehran'));
    Notification::fake();
    $this->seed(SettingsSeeder::class);

    $this->bodysuit = Product::factory()->published()->priced(485_000)->create(['title' => 'بادی آستین‌بلند نخی', 'slug' => 'bodysuit']);
    $this->milk = ProductVariant::factory()->create(['product_id' => $this->bodysuit->id, 'size' => '۳-۶ ماه', 'color' => 'شیری', 'stock_qty' => 4]);
    $this->pads = Product::factory()->published()->priced(98_000)->create(['title' => 'نوار بهداشتی روزانه', 'slug' => 'pads', 'stock_qty' => 30, 'stock_status' => 'in_stock']);
    $this->socks = Product::factory()->published()->priced(140_000)->create(['title' => 'جوراب نوزادی', 'slug' => 'socks', 'stock_qty' => 1, 'stock_status' => 'in_stock']);
});

function cartAdd(mixed $test, Product $product, int $quantity = 1, ?ProductVariant $variant = null): void
{
    $test->post('/shop/cart', array_filter([
        'product' => $product->id, 'quantity' => (string) $quantity, 'size' => $variant?->size, 'color' => $variant?->color,
    ]))->assertStatus(303);
}

/**
 * Opens the checkout page, reads its hidden fields and returns a valid form (5 s later, past the FormTimer).
 *
 * @return array<string, string>
 */
function checkoutForm(mixed $test, array $overrides = []): array
{
    $html = (string) $test->get('/shop/checkout')->assertOk()->getContent();
    $hidden = static function (string $name) use ($html): string {
        preg_match('/name="'.preg_quote($name, '/').'" value="([^"]*)"/', $html, $m);

        return html_entity_decode($m[1] ?? '');
    };
    $test->travel(5)->seconds();

    return array_merge([
        'name' => 'سارا محمدی',
        'mobile' => '۰۹۱۲ ۱۲۳ ۴۵۶۷',
        'province' => 'tehran',
        'city' => 'تهران',
        'address' => 'خیابان ولیعصر، کوچه یاس، پلاک ۱۲، واحد ۳',
        'postal_code' => '۱۲۳۴۵-۶۷۸۹۰',
        'note' => 'زنگ واحد ۳',
        'delivery' => DeliverySlots::offered()[1]->value(),
        'discreet' => '1',
        'payment' => 'cash_on_delivery',
        'checkout_token' => $hidden('checkout_token'),
        'cart_signature' => $hidden('cart_signature'),
        'form_token' => $hidden('form_token'),
        'website' => '',
    ], $overrides);
}

function placeOrder(mixed $test, array $form): TestResponse
{
    return $test->post('/shop/checkout', $form);
}

// ---- checkout page --------------------------------------------------------------------------------------------------

it('renders the empty checkout: one h1, noindex, no-store, never page-cached, no inline styles', function (): void {
    config(['pagecache.enabled' => true]);

    $response = $this->get('/shop/checkout')->assertOk();
    $html = (string) $response->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('سبد خریدت خالی است')
        ->and($html)->not->toContain('name="checkout_token"')
        ->and($html)->toMatch('#<meta name="robots" content="noindex#')
        ->and($html)->not->toMatch('/\sstyle\s*=/i')
        ->and($response->headers->get('Cache-Control'))->toContain('no-store')
        ->and($this->get('/shop/checkout')->headers->get(PageCache::HEADER))->not->toBe('HIT');
});

it('renders the form for a filled cart: address, slots, discreet switch, COD only, live summary', function (): void {
    cartAdd($this, $this->bodysuit, 2, $this->milk);
    cartAdd($this, $this->pads);

    $html = (string) $this->get('/shop/checkout')->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('action="'.route('shop.checkout.store').'"')
        ->and($html)->toContain('<option value="tehran"')
        ->and($html)->toContain('<datalist id="checkout-cities">')
        ->and(substr_count($html, 'name="delivery"'))->toBe(DeliverySlots::OFFERED_DAYS * count(DeliveryWindow::cases()))
        ->and($html)->toContain('فردا')
        ->and($html)->toMatch('/name="discreet" value="1" checked/')
        ->and($html)->toContain('پرداخت در محل')
        ->and($html)->not->toContain('درگاه بانکی')                          // AUDIT §8: COD only, no gateway option
        ->and(substr_count($html, 'name="payment"'))->toBe(1)
        ->and($html)->toContain('جمع کالاها (۳)')
        ->and($html)->toContain('۱٬۰۶۸٬۰۰۰ تومان')                            // 2 × 485k + 98k, live
        ->and($html)->toContain('هنگام تماس اعلام می‌شود')
        ->and($html)->not->toContain('پرداخت شد')
        ->and($html)->not->toMatch('/\sstyle\s*=/i')
        ->and($html)->toMatch('/name="checkout_token" value="[0-9a-f]{40}"/');
});

// ---- placing an order -----------------------------------------------------------------------------------------------

it('places a COD order in one go: live prices, stock decremented, snapshots, cart cleared, PRG, notifications', function (): void {
    app(UpdateSettings::class)->handle(SettingGroup::Contact, ['support_email' => 'support@ritme.test']);
    cartAdd($this, $this->bodysuit, 2, $this->milk);
    cartAdd($this, $this->pads, 3);

    $response = placeOrder($this, checkoutForm($this, ['price' => '1', 'total' => '1', 'unit_price' => '1']));

    $order = Order::query()->with('items')->sole();
    $response->assertStatus(303)->assertRedirect(route('shop.order', [$order->code]));

    expect($order->code)->toMatch(OrderCode::PATTERN)
        ->and($order->status)->toBe(OrderStatus::Pending)
        ->and($order->payment_status)->toBe(PaymentStatus::Unpaid)
        ->and($order->subtotal->rial)->toBe(2 * 4_850_000 + 3 * 980_000)       // client «price» ignored
        ->and($order->shipping_fee)->toBeNull()
        ->and($order->total->rial)->toBe($order->subtotal->rial)
        ->and($order->items_count)->toBe(5)
        ->and($order->mobile)->toBe('09121234567')
        ->and($order->postal_code)->toBe('1234567890')
        ->and($order->province)->toBe('tehran')
        ->and($order->discreet_packaging)->toBeTrue()
        ->and($order->delivery_date->format('Y-m-d'))->toBe('2026-10-11')
        ->and($order->delivery_window)->toBe(DeliveryWindow::Evening)
        ->and($order->items)->toHaveCount(2)
        ->and($order->items[0]->title)->toBe('بادی آستین‌بلند نخی')
        ->and($order->items[0]->variant_label)->toBe('۳-۶ ماه · شیری')
        ->and($order->items[0]->unit_price->rial)->toBe(4_850_000)
        ->and($order->items[0]->line_total->rial)->toBe(9_700_000)
        ->and($this->milk->fresh()->stock_qty)->toBe(2)
        ->and($this->pads->fresh()->stock_qty)->toBe(27)
        ->and($this->pads->fresh()->sales_count)->toBe(3)
        ->and(session(SessionCartRepository::KEY))->toBeNull()
        ->and(session(CheckoutSession::ORDERS_KEY))->toBe([$order->code]);

    Notification::assertSentTo(new AnonymousNotifiable, OrderPlacedForTeam::class, function (OrderPlacedForTeam $n, array $channels, AnonymousNotifiable $to): bool {
        $mail = $n->toMail($to)->render();

        return $to->routes['mail'] === 'support@ritme.test' && ! str_contains((string) $mail, '09121234567') && ! str_contains((string) $mail, 'ولیعصر');
    });
    Notification::assertSentTo(new AnonymousNotifiable, OrderPlacedForCustomer::class, function (OrderPlacedForCustomer $n, array $channels, AnonymousNotifiable $to) use ($order): bool {
        $sms = $n->toSms($to);

        return $to->routes['sms'] === '09121234567' && str_contains($sms, $order->code) && str_contains($sms, 'هنگام تحویل')
            && ! str_contains($sms, 'بادی') && ! str_contains($sms, 'پرداخت شد');
    });
});

it('lets only one of two orders for the last unit through (the other gets a Persian refusal)', function (): void {
    // Both shoppers open the checkout while one pair of socks is left.
    cartAdd($this, $this->socks);
    $formA = checkoutForm($this);
    $sessionA = session()->all();
    $this->flushSession();

    cartAdd($this, $this->socks);
    $formB = checkoutForm($this);
    $sessionB = session()->all();

    $this->flushSession();
    $this->withSession($sessionA);
    placeOrder($this, $formA)->assertStatus(303);

    $this->flushSession();
    $this->withSession($sessionB);
    placeOrder($this, $formB)->assertRedirect(route('shop.checkout').'#checkout-form')
        ->assertSessionHas('checkout_error', 'جوراب نوزادی الان ناموجود است. آن را از سبد حذف کن و دوباره ثبت کن.');

    expect(Order::query()->count())->toBe(1)
        ->and($this->socks->fresh()->stock_qty)->toBe(0)
        ->and($this->socks->fresh()->sales_count)->toBe(1);
});

it('rolls every line back when AdjustStock refuses one inside the transaction (stale read race)', function (): void {
    cartAdd($this, $this->pads, 2);
    cartAdd($this, $this->socks);
    $form = checkoutForm($this);

    // Another order takes the last socks between the live read and the decrement: the catalog still says 1 left.
    $live = app(CartCatalog::class);
    $this->app->instance(CartCatalog::class, new class($live) implements CartCatalog
    {
        public function __construct(private readonly CartCatalog $inner) {}

        public function find(array $productIds): array
        {
            $products = $this->inner->find($productIds);
            Product::query()->where('slug', 'socks')->update(['stock_qty' => 0]);

            return $products;
        }
    });

    placeOrder($this, $form)->assertRedirect(route('shop.checkout').'#checkout-form')
        ->assertSessionHas('checkout_error', 'موجودی جوراب نوزادی همین حالا تمام شد و سفارشی ثبت نشد. سبد را بررسی کن و دوباره ثبت کن.');

    expect(Order::query()->count())->toBe(0)
        ->and($this->pads->fresh()->stock_qty)->toBe(30)                  // first line's decrement rolled back
        ->and($this->pads->fresh()->sales_count)->toBe(0);
    Notification::assertNothingSent();
});

it('refuses when the price changed after the page was opened, with the cart notice', function (): void {
    cartAdd($this, $this->pads);
    $form = checkoutForm($this);
    $this->pads->update(['price' => Money::fromToman(120_000)]);

    placeOrder($this, $form)->assertRedirect(route('shop.checkout').'#checkout-form')
        ->assertSessionHas('checkout_error', fn (string $m): bool => str_contains($m, 'سبد خرید از زمان باز کردن این صفحه تغییر کرد'))
        ->assertSessionHas('checkout_notices', ['قیمت نوار بهداشتی روزانه از ۹۸ هزار تومان به ۱۲۰ هزار تومان تغییر کرد.'])
        ->assertSessionHasInput('name', 'سارا محمدی');

    expect(Order::query()->count())->toBe(0);

    // The re-rendered form carries the new signature: the next submit goes through at the new price.
    placeOrder($this, checkoutForm($this))->assertStatus(303);
    expect(Order::query()->sole()->total->rial)->toBe(1_200_000);
});

it('returns the first order on a double submit instead of placing a second one', function (): void {
    cartAdd($this, $this->pads);
    $form = checkoutForm($this);

    $first = placeOrder($this, $form)->assertStatus(303);
    $second = placeOrder($this, $form)->assertStatus(303);

    expect($second->headers->get('Location'))->toBe($first->headers->get('Location'))
        ->and(Order::query()->count())->toBe(1)
        ->and($this->pads->fresh()->stock_qty)->toBe(29);
});

it('rejects a stale or foreign form token', function (): void {
    cartAdd($this, $this->pads);
    $form = checkoutForm($this, ['checkout_token' => str_repeat('a', 40)]);

    placeOrder($this, $form)->assertRedirect(route('shop.checkout'))
        ->assertSessionHas('checkout_error', 'این فرم دیگر معتبر نیست. صفحه به‌روز شد؛ دوباره ثبت کن.');
    expect(Order::query()->count())->toBe(0);
});

it('enforces the COD cap: message + disabled button on the page, refusal on submit', function (): void {
    config(['shop.cod_max_amount' => 100_000]);
    cartAdd($this, $this->bodysuit, 1, $this->milk);

    $html = (string) $this->get('/shop/checkout')->assertOk()->getContent();
    expect($html)->toContain('پرداخت در محل فقط برای سفارش‌های تا ۱۰۰٬۰۰۰ تومان ممکن است')
        ->and($html)->toContain('فقط برای سفارش‌های تا ۱۰۰٬۰۰۰ تومان')
        ->and($html)->toMatch('/<button type="submit" disabled/');

    placeOrder($this, checkoutForm($this))
        ->assertSessionHas('checkout_error', 'پرداخت در محل فقط برای سفارش‌های تا ۱۰۰٬۰۰۰ تومان ممکن است؛ سفارشی ثبت نشد.');
    expect(Order::query()->count())->toBe(0)
        ->and($this->milk->fresh()->stock_qty)->toBe(4);
});

it('validates in Persian and keeps the typed input', function (): void {
    cartAdd($this, $this->pads);

    placeOrder($this, checkoutForm($this, ['mobile' => '12345', 'province' => 'atlantis', 'postal_code' => '123', 'address' => 'کوتاه', 'delivery' => '2026-12-30|morning']))
        ->assertRedirect(route('shop.checkout').'#checkout-form')
        ->assertSessionHasErrors([
            'mobile' => 'شماره موبایل معتبر وارد کن؛ مثلاً ۰۹۱۲۳۴۵۶۷۸۹.',
            'province' => 'استان را از فهرست انتخاب کن.',
            'postal_code' => 'کد پستی باید ۱۰ رقم باشد؛ اگر نمی‌دانی، خالی بگذار.',
            'address' => 'آدرس کامل را بنویس؛ دست‌کم ۱۰ حرف و حداکثر ۵۰۰ حرف.',
            'delivery' => 'یکی از زمان‌های تحویل را انتخاب کن.',
        ])
        ->assertSessionHasInput('name', 'سارا محمدی');

    placeOrder($this, checkoutForm($this, ['payment' => 'bank_gateway']))->assertSessionHasErrors(['payment' => 'روش پرداخت را انتخاب کن.']);
    expect(Order::query()->count())->toBe(0);
});

it('drops honeypot and too-fast posts without storing anything', function (): void {
    cartAdd($this, $this->pads);

    placeOrder($this, checkoutForm($this, ['website' => 'https://spam.example']))
        ->assertRedirect(route('shop.checkout'))->assertSessionHas('checkout_error', 'ثبت سفارش انجام نشد. لطفاً چند لحظه بعد دوباره امتحان کن.');

    $form = checkoutForm($this);
    $form['form_token'] = app(FormTimer::class)->issue(); // issued "now" → too fast
    placeOrder($this, $form)->assertRedirect(route('shop.checkout'));

    expect(Order::query()->count())->toBe(0)
        ->and($this->pads->fresh()->stock_qty)->toBe(30);
});

it('rate limits checkout posts per IP', function (): void {
    for ($i = 0; $i < 5; $i++) {
        $this->post('/shop/checkout', ['website' => 'x'])->assertRedirect();
    }
    $this->post('/shop/checkout', ['website' => 'x'])->assertStatus(429);
});

// ---- order page -----------------------------------------------------------------------------------------------------

it('shows the order page to its owner: one h1, noindex, no-store, masked data, due on delivery, never "paid"', function (): void {
    config(['pagecache.enabled' => true]);
    cartAdd($this, $this->bodysuit, 1, $this->milk);
    placeOrder($this, checkoutForm($this))->assertStatus(303);
    $order = Order::query()->sole();

    $response = $this->get('/shop/order/'.strtolower($order->code))->assertOk();
    $html = (string) $response->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('سفارشت ثبت شد')
        ->and($html)->toContain($order->code)
        ->and($html)->toContain('۰۹۱۲•••••۶۷')
        ->and($html)->not->toContain('09121234567')
        ->and($html)->not->toContain('ولیعصر')
        ->and($html)->not->toContain('1234567890')
        ->and($html)->toContain('تهران · تهران')
        ->and($html)->toContain('قابل پرداخت هنگام تحویل')
        ->and($html)->toContain('۴۸۵٬۰۰۰ تومان')
        ->and($html)->toContain('در انتظار تأیید فروشگاه')
        ->and($html)->toContain('بسته‌بندی ساده')
        ->and($html)->not->toContain('پرداخت شد')
        ->and($html)->not->toContain('رسید پرداخت')
        ->and($html)->toMatch('#<meta name="robots" content="noindex#')
        ->and($html)->not->toMatch('/\sstyle\s*=/i')
        ->and($response->headers->get('Cache-Control'))->toContain('no-store')
        ->and($this->get('/shop/order/'.$order->code)->headers->get(PageCache::HEADER))->not->toBe('HIT');
});

it('hides the recipient rows from other sessions and 404s unknown codes', function (): void {
    cartAdd($this, $this->pads);
    placeOrder($this, checkoutForm($this))->assertStatus(303);
    $order = Order::query()->sole();

    $this->flushSession();
    $html = (string) $this->get('/shop/order/'.$order->code)->assertOk()->getContent();
    expect($html)->toContain($order->code)
        ->and($html)->not->toContain('سارا محمدی')
        ->and($html)->not->toContain('۰۹۱۲•••••۶۷');

    $this->get('/shop/order/AAAA-BBBB-CCCC')->assertNotFound();
});
