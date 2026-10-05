<?php

declare(strict_types=1);

use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Settings\Actions\UpdateSettings;
use App\Domain\Settings\Enums\SettingGroup;
use App\Domain\Shop\Cart\Data\ShippingRule;
use App\Domain\Shop\Catalog\Enums\ReviewStatus;
use App\Domain\Shop\Catalog\Enums\StockStatus;
use App\Domain\Shop\Catalog\Models\Brand;
use App\Domain\Shop\Catalog\Models\Category;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductReview;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use App\Domain\Shop\Ordering\Actions\ChangeOrderStatus;
use App\Domain\Shop\Ordering\Enums\DeliveryWindow;
use App\Domain\Shop\Ordering\Enums\OrderStatus;
use App\Domain\Shop\Ordering\Models\Order;
use App\Domain\Shop\Ordering\Support\OrderCode;
use App\Domain\Shop\Payment\Contracts\PaymentGateway;
use App\Domain\Shop\Payment\Enums\PaymentMethod;
use App\Domain\Shop\Payment\Enums\PaymentStatus;
use App\Filament\Auth\AdminRole;
use App\Filament\Pages\Settings\ShopSettingsPage;
use App\Filament\Resources\Shop\Brands\BrandResource;
use App\Filament\Resources\Shop\Categories\Pages\EditShopCategory;
use App\Filament\Resources\Shop\Categories\Pages\ListShopCategories;
use App\Filament\Resources\Shop\Categories\ShopCategoryResource;
use App\Filament\Resources\Shop\LowStock;
use App\Filament\Resources\Shop\Orders\OrderResource;
use App\Filament\Resources\Shop\Orders\Pages\ListOrders;
use App\Filament\Resources\Shop\Orders\Pages\ViewOrder;
use App\Filament\Resources\Shop\Products\Pages\CreateProduct;
use App\Filament\Resources\Shop\Products\Pages\EditProduct;
use App\Filament\Resources\Shop\Products\ProductResource;
use App\Filament\Resources\Shop\Reviews\Pages\ListProductReviews;
use App\Filament\Resources\Shop\Reviews\ProductReviewResource;
use App\Filament\Widgets\Shop\LowStockProducts;
use App\Filament\Widgets\Shop\ShopOrdersOverview;
use App\Models\User;
use App\Support\Money\Money;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Filament\Actions\Testing\TestAction;
use Filament\Facades\Filament;
use Livewire\Livewire;
use Spatie\Activitylog\Models\Activity;
use Tests\Feature\Admin\AdminMfa;

function shopAdmin(?AdminRole $role = AdminRole::ShopManager): User
{
    $user = User::factory()->create(['name' => 'مدیر فروشگاه']);
    if ($role !== null) {
        $user->assignRole($role->value);
    }

    return AdminMfa::enrol($user); // PII roles must have MFA (L9-04b)
}

/**
 * An order as PlaceOrder leaves it: stock already decremented for tracked lines, sales_count raised for every line.
 *
 * @param  list<array{0: Product, 1: ProductVariant|null, 2: int, 3?: bool}>  $lines  product, variant, quantity, tracked
 */
function shopAdminOrder(array $lines, string $status = 'pending'): Order
{
    $subtotal = Money::zero();
    foreach ($lines as [$product, $variant, $quantity]) {
        $subtotal = $subtotal->plus($product->price->times($quantity));
    }

    $order = Order::query()->create([
        'code' => OrderCode::generate(), // unique: a 4-hex random prefix collided now and then
        'idempotency_key' => bin2hex(random_bytes(32)),
        'status' => $status,
        'payment_method' => PaymentMethod::CashOnDelivery,
        'payment_status' => PaymentStatus::Unpaid,
        'subtotal' => $subtotal,
        'shipping_fee' => null,
        'total' => $subtotal,
        'items_count' => array_sum(array_map(static fn (array $l): int => $l[2], $lines)),
        'recipient_name' => 'سارا رحیمی',
        'mobile' => '09121234567',
        'province' => 'تهران',
        'city' => 'تهران',
        'address' => 'خیابان آزمایشی، کوچه یاس، پلاک ۷',
        'postal_code' => '1234567890',
        'note' => 'لطفاً قبل از ارسال تماس بگیرید.',
        'delivery_date' => '2026-10-08',
        'delivery_window' => DeliveryWindow::Evening,
        'discreet_packaging' => true,
    ]);

    foreach ($lines as $line) {
        [$product, $variant, $quantity] = $line;
        $tracked = $line[3] ?? true;
        $order->items()->create([
            'product_id' => $product->id,
            'variant_id' => $variant?->id,
            'title' => $product->title,
            'variant_label' => $variant?->size,
            'unit_price' => $product->price,
            'quantity' => $quantity,
            'line_total' => $product->price->times($quantity),
            'stock_tracked' => $tracked,
        ]);
        if ($tracked) {
            $variant !== null
                ? $variant->decrement('stock_qty', $quantity)
                : Product::query()->whereKey($product->id)->decrement('stock_qty', $quantity);
        }
        Product::query()->whereKey($product->id)->increment('sales_count', $quantity);
    }

    return $order;
}

beforeEach(function (): void {
    $this->seed([SettingsSeeder::class, AdminRolesSeeder::class]);
    Filament::setCurrentPanel('admin');
});

it('lets shop managers and super-admins into the shop admin, SEO managers into catalog SEO only, and denies the rest', function (): void {
    $product = Product::factory()->create();
    $order = shopAdminOrder([[$product, null, 1]]);
    $all = [
        ProductResource::getUrl('index'), ProductResource::getUrl('create'), ProductResource::getUrl('edit', ['record' => $product]),
        ShopCategoryResource::getUrl('index'), BrandResource::getUrl('index'), ProductReviewResource::getUrl('index'),
        OrderResource::getUrl('index'), OrderResource::getUrl('view', ['record' => $order]),
        OrderResource::getUrl('invoice', ['record' => $order]), OrderResource::getUrl('export'), ShopSettingsPage::getUrl(),
    ];

    $this->get($all[0])->assertRedirect('/admin/login');

    foreach ([AdminRole::ShopManager, AdminRole::SuperAdmin] as $role) {
        $user = shopAdmin($role);
        foreach ($all as $url) {
            $this->actingAs($user)->get($url)->assertOk();
        }
    }

    $seo = shopAdmin(AdminRole::SeoManager);
    foreach ([$all[0], $all[2], $all[3]] as $url) {
        $this->actingAs($seo)->get($url)->assertOk();
    }
    foreach ([$all[1], $all[4], $all[5], $all[6], $all[7], $all[8], $all[9], $all[10]] as $url) {
        $this->actingAs($seo)->get($url)->assertForbidden();
    }

    foreach ([AdminRole::Editor, AdminRole::DirectoryManager, AdminRole::Support, null] as $role) {
        $user = shopAdmin($role);
        foreach ([$all[0], $all[3], $all[4], $all[6], $all[7], $all[8], $all[9], $all[10]] as $url) {
            $this->actingAs($user)->get($url)->assertForbidden();
        }
    }

    $manager = shopAdmin();
    expect($manager->can('create', Order::class))->toBeFalse()
        ->and($manager->can('delete', $order))->toBeFalse()
        ->and($manager->can('update', $order))->toBeTrue()
        ->and($manager->can('export', Order::class))->toBeTrue()
        ->and($manager->can('create', ProductReview::class))->toBeFalse()
        ->and($seo->can('create', Product::class))->toBeFalse()
        ->and($seo->can('update', $product))->toBeTrue();
});

it('creates a product with toman prices stored as rials, variants, categories and SEO, and logs publish + stock changes', function (): void {
    $this->actingAs($manager = shopAdmin());
    $department = Category::factory()->create(['name' => 'لباس نوزاد']);
    $bodies = Category::factory()->childOf($department)->create(['name' => 'بادی']);
    $brand = Brand::factory()->create();

    Livewire::test(CreateProduct::class)
        ->fillForm([
            'title' => 'بادی آستین‌بلند نخی',
            'brand_id' => $brand->id,
            'short_description' => 'بادی نخی نرم برای روزهای اول.',
            'price_toman' => 189_000,
            'compare_at_toman' => 219_000,
            'stock_mode' => 'auto',
            'variants' => [
                ['size' => '۰-۳ ماه', 'color' => 'صورتی', 'color_hex' => '#f4c7c3', 'stock_qty' => 4, 'is_active' => true],
                ['size' => '۳-۶ ماه', 'color' => 'صورتی', 'price_toman' => 199_000, 'stock_qty' => 6, 'is_active' => true],
            ],
            'specs' => [['label' => 'جنس', 'value' => 'پنبه ۱۰۰٪']],
            'size_chart_columns' => ['سایز', 'قد (سانتی‌متر)'],
            'size_chart_rows' => [['cells' => '۰-۳ ماه | ۵۰-۶۲'], ['cells' => '۳-۶ ماه | ۶۲-۶۸']],
            'category_ids' => [$department->id, $bodies->id],
            'primary_category_id' => $bodies->id,
            'seoMeta.title' => 'بادی نخی نوزاد آستین‌بلند',
        ])
        ->call('create')
        ->assertHasNoFormErrors();

    $product = Product::query()->where('title', 'بادی آستین‌بلند نخی')->firstOrFail();
    $variants = $product->variants()->get();
    expect($product->price->rial)->toBe(1_890_000)
        ->and($product->compare_at_price?->rial)->toBe(2_190_000)
        ->and($product->is_published)->toBeFalse()
        ->and($product->slug)->not->toBe('')
        ->and($product->stock_qty)->toBe(10)
        ->and($product->stock_status)->toBe(StockStatus::InStock)
        ->and($product->primary_category_id)->toBe($bodies->id)
        ->and($product->categories()->pluck('shop_categories.id')->map(intval(...))->sort()->values()->all())->toBe([$department->id, $bodies->id])
        ->and($product->specs)->toBe([['label' => 'جنس', 'value' => 'پنبه ۱۰۰٪']])
        ->and($product->size_chart)->toBe(['columns' => ['سایز', 'قد (سانتی‌متر)'], 'rows' => [['۰-۳ ماه', '۵۰-۶۲'], ['۳-۶ ماه', '۶۲-۶۸']]])
        ->and($variants)->toHaveCount(2)
        ->and($variants[0]->color_hex)->toBe('#F4C7C3')
        ->and($variants[0]->price)->toBeNull()
        ->and($variants[1]->price?->rial)->toBe(1_990_000)
        ->and(SeoMeta::query()->where('seoable_id', $product->id)->value('title'))->toBe('بادی نخی نوزاد آستین‌بلند')
        ->and(Activity::query()->where('description', 'shop.product.created')->where('subject_id', $product->id)->where('causer_id', $manager->id)->exists())->toBeTrue();

    // Edit: the form shows tomans; publish, change one variant's stock, drop the other.
    Livewire::test(EditProduct::class, ['record' => $product->getRouteKey()])
        ->assertSchemaStateSet(['price_toman' => 189_000, 'compare_at_toman' => 219_000, 'stock_mode' => 'auto'])
        ->fillForm([
            'is_published' => true,
            'variants' => [
                ['id' => $variants[0]->id, 'size' => '۰-۳ ماه', 'color' => 'صورتی', 'color_hex' => '#F4C7C3', 'stock_qty' => 9, 'is_active' => true],
            ],
        ])
        ->call('save')
        ->assertHasNoFormErrors();

    expect($product->refresh()->is_published)->toBeTrue()
        ->and($product->stock_qty)->toBe(9)
        ->and(ProductVariant::query()->where('product_id', $product->id)->count())->toBe(1)
        ->and(Activity::query()->where('description', 'shop.product.status')->where('subject_id', $product->id)->exists())->toBeTrue();

    $stock = Activity::query()->where('description', 'shop.product.stock')->where('subject_id', $product->id)->firstOrFail();
    expect($stock->properties['old']['product'])->toBe(10)
        ->and($stock->properties['attributes']['product'])->toBe(9);

    $this->get(route('shop.product', [$product->slug]))->assertOk()->assertSee('بادی آستین‌بلند نخی');

    // Two variants with the same size + colour are refused.
    Livewire::test(EditProduct::class, ['record' => $product->getRouteKey()])
        ->fillForm(['variants' => [
            ['size' => 'L', 'color' => 'آبی', 'stock_qty' => 1, 'is_active' => true],
            ['size' => 'L', 'color' => 'آبی', 'stock_qty' => 2, 'is_active' => true],
        ]])
        ->call('save')
        ->assertHasErrors(['data.variants']);
});

it('lets SEO managers change only the SEO tab of a product', function (): void {
    $product = Product::factory()->published()->priced(250_000)->create(['title' => 'شیشه شیر']);
    $this->actingAs(shopAdmin(AdminRole::SeoManager));

    Livewire::test(EditProduct::class, ['record' => $product->getRouteKey()])
        ->set('data.title', 'نام دست‌کاری‌شده')
        ->set('data.price_toman', 1)
        ->fillForm(['seoMeta.title' => 'شیشه شیر ضدنفخ'])
        ->call('save');

    expect($product->refresh()->title)->toBe('شیشه شیر')
        ->and($product->price->toToman())->toBe(250_000)
        ->and(SeoMeta::query()->where('seoable_id', $product->id)->where('seoable_type', 'shop_product')->value('title'))->toBe('شیشه شیر ضدنفخ');
});

it('moves an order through its lifecycle, rejects invalid transitions and masks personal data in the list', function (): void {
    $this->actingAs($manager = shopAdmin());
    $product = Product::factory()->published()->create(['stock_qty' => 5]);
    $order = shopAdminOrder([[$product, null, 2]]);

    Livewire::test(ListOrders::class)
        ->assertCanSeeTableRecords([$order])
        ->assertSee('۰۹۱۲•••••۶۷')
        ->assertSee('س… ر…')
        ->assertDontSee('09121234567')
        ->assertDontSee('سارا رحیمی')
        ->assertDontSee('کوچه یاس')
        ->assertActionVisible(TestAction::make('status_confirmed')->table($order))
        ->assertActionHidden(TestAction::make('status_shipped')->table($order))
        ->assertActionHidden(TestAction::make('status_delivered')->table($order))
        ->callAction(TestAction::make('status_confirmed')->table($order))
        ->assertHasNoActionErrors();

    expect($order->refresh()->status)->toBe(OrderStatus::Confirmed);

    // Search by the full mobile number finds the order (the list itself never shows it).
    Livewire::test(ListOrders::class, ['activeTab' => 'all'])
        ->searchTable('۰۹۱۲۱۲۳۴۵۶۷')
        ->assertCanSeeTableRecords([$order]);

    Livewire::test(ViewOrder::class, ['record' => $order->getRouteKey()])
        ->assertSee('سارا رحیمی')
        ->assertSee('کوچه یاس')
        ->assertSee('tel:09121234567', escape: false)
        ->assertActionHidden('status_confirmed')
        ->assertActionHidden('status_delivered')
        ->callAction('status_shipped')
        ->assertHasNoActionErrors()
        ->callAction('status_delivered')
        ->assertHasNoActionErrors();

    expect($order->refresh()->status)->toBe(OrderStatus::Delivered)
        ->and($order->payment_status)->toBe(PaymentStatus::Paid)
        ->and(Activity::query()->where('description', 'shop.order.status')->where('subject_id', $order->id)->where('causer_id', $manager->id)->count())->toBe(3);

    // Final: nothing more is offered, and the action refuses a change the state machine does not allow.
    Livewire::test(ViewOrder::class, ['record' => $order->getRouteKey()])
        ->assertActionHidden('status_cancelled')
        ->assertActionHidden('status_shipped');

    expect(fn () => app(ChangeOrderStatus::class)->handle($order, OrderStatus::Cancelled))->toThrow(InvalidArgumentException::class)
        ->and(fn () => app(ChangeOrderStatus::class)->handle(shopAdminOrder([[$product, null, 1]]), OrderStatus::Delivered))->toThrow(InvalidArgumentException::class)
        ->and($order->refresh()->status)->toBe(OrderStatus::Delivered)
        ->and($product->refresh()->stock_qty)->toBe(2);

    // The customer's order page follows the admin change.
    $this->get(route('shop.order', [$order->code]))->assertOk()->assertSee(OrderStatus::Delivered->label());
});

it('returns only tracked stock and lowers sales_count when an order is cancelled', function (): void {
    $this->actingAs($manager = shopAdmin());
    $simple = Product::factory()->published()->create(['stock_qty' => 5]);
    $withVariants = Product::factory()->published()->create();
    $variant = ProductVariant::factory()->create(['product_id' => $withVariants->id, 'size' => 'S', 'stock_qty' => 4]);
    $preorder = Product::factory()->published()->create(['stock_qty' => 0, 'stock_status' => StockStatus::PreOrder]);

    $order = shopAdminOrder([[$simple, null, 2], [$withVariants, $variant, 3], [$preorder, null, 1, false]], 'confirmed');
    expect($simple->refresh()->stock_qty)->toBe(3)
        ->and($variant->refresh()->stock_qty)->toBe(1)
        ->and($simple->sales_count)->toBe(2);

    Livewire::test(ViewOrder::class, ['record' => $order->getRouteKey()])
        ->callAction('status_cancelled')
        ->assertHasNoActionErrors();

    expect($order->refresh()->status)->toBe(OrderStatus::Cancelled)
        ->and($order->payment_status)->toBe(PaymentStatus::Unpaid)
        ->and($simple->refresh()->stock_qty)->toBe(5)
        ->and($simple->sales_count)->toBe(0)
        ->and($variant->refresh()->stock_qty)->toBe(4)
        ->and($withVariants->refresh()->stock_qty)->toBe(4)
        ->and($withVariants->sales_count)->toBe(0)
        ->and($preorder->refresh()->stock_qty)->toBe(0)
        ->and($preorder->sales_count)->toBe(0);

    $entry = Activity::query()->where('description', 'shop.order.status')->where('subject_id', $order->id)->firstOrFail();
    expect($entry->causer_id)->toBe($manager->id)
        ->and($entry->properties['attributes']['status'])->toBe('cancelled')
        ->and($entry->properties['stock']['restored'])->toHaveCount(2)
        ->and($entry->properties['stock']['skipped'])->toBe([]);

    // Cancelled is final: a second cancel cannot return the stock twice.
    expect(fn () => app(ChangeOrderStatus::class)->handle($order, OrderStatus::Cancelled))->toThrow(InvalidArgumentException::class)
        ->and($simple->refresh()->stock_qty)->toBe(5);
});

it('adds internal notes, prints a self-contained invoice and exports a minimal CSV', function (): void {
    $this->actingAs($manager = shopAdmin());
    $product = Product::factory()->published()->priced(129_000)->create(['title' => 'پستانک سیلیکونی']);
    $order = shopAdminOrder([[$product, null, 2]]);
    shopAdminOrder([[$product, null, 1]], 'delivered');

    Livewire::test(ViewOrder::class, ['record' => $order->getRouteKey()])
        ->callAction('note', ['note' => 'تماس گرفته شد؛ عصر پنجشنبه تحویل.'])
        ->assertHasNoActionErrors();

    expect(Activity::query()->where('description', 'shop.order.note')->where('subject_id', $order->id)->where('causer_id', $manager->id)->exists())->toBeTrue();
    Livewire::test(ViewOrder::class, ['record' => $order->getRouteKey()])->assertSee('تماس گرفته شد؛ عصر پنجشنبه تحویل.');

    $invoice = (string) $this->get(OrderResource::getUrl('invoice', ['record' => $order]))
        ->assertOk()
        ->assertHeader('Cache-Control', 'no-store, private')
        ->getContent();
    expect($invoice)->toContain('dir="rtl"')
        ->toContain($order->code)
        ->toContain('پستانک سیلیکونی')
        ->toContain('۲۵۸٬۰۰۰ تومان')
        ->toContain('سارا رحیمی')
        ->toContain('کوچه یاس')
        ->not->toContain('http://')
        ->not->toContain('https://')
        ->not->toContain('<link')
        ->not->toContain('<script src');

    $csv = $this->get(OrderResource::getUrl('export', ['status' => 'pending']))
        ->assertOk()
        ->assertHeader('Cache-Control', 'no-store, private')
        ->streamedContent();

    expect($csv)->toStartWith("\xEF\xBB\xBF")
        ->toContain($order->code)
        ->toContain('258000')
        ->toContain('0912•••••67')
        ->not->toContain('09121234567')
        ->not->toContain('سارا رحیمی')
        ->not->toContain('کوچه یاس')
        ->and(substr_count(trim($csv), "\n"))->toBe(1)
        ->and(Activity::query()->where('description', 'shop.order.exported')->where('causer_id', $manager->id)->exists())->toBeTrue();
});

it('moderates product reviews one by one and in bulk, recalculating the rating and logging every change', function (): void {
    $this->actingAs($manager = shopAdmin());
    $product = Product::factory()->published()->create();
    $a = ProductReview::factory()->create(['product_id' => $product->id, 'rating' => 5, 'status' => ReviewStatus::Pending]);
    $b = ProductReview::factory()->create(['product_id' => $product->id, 'rating' => 3, 'status' => ReviewStatus::Pending]);
    $c = ProductReview::factory()->create(['product_id' => $product->id, 'rating' => 1, 'status' => ReviewStatus::Pending]);

    Livewire::test(ListProductReviews::class)
        ->assertCanSeeTableRecords([$a, $b, $c])
        ->selectTableRecords([$a->id, $b->id])
        ->callAction(TestAction::make('approveBulk')->table()->bulk())
        ->assertHasNoActionErrors();

    expect($product->refresh()->rating_count)->toBe(2)
        ->and($product->rating_avg)->toBe(4.0);

    Livewire::test(ListProductReviews::class)
        ->callAction(TestAction::make('reject')->table($c))
        ->assertHasNoActionErrors();

    expect($c->refresh()->status)->toBe(ReviewStatus::Rejected)
        ->and(Activity::query()->where('description', 'shop.review.status')->where('causer_id', $manager->id)->count())->toBe(3);
});

it('keeps the category tree acyclic, drag-sorts siblings and refuses to delete a category in use', function (): void {
    $this->actingAs(shopAdmin());
    $root = Category::factory()->create(['name' => 'لباس']);
    $child = Category::factory()->childOf($root)->create(['name' => 'بادی', 'sort_order' => 1]);
    $grandchild = Category::factory()->childOf($child)->create(['name' => 'بادی نوزاد']);
    $sibling = Category::factory()->childOf($root)->create(['name' => 'سرهمی', 'sort_order' => 2]);

    $options = ShopCategoryResource::parentOptions($child);
    expect($options)->not->toHaveKey($child->id)
        ->and($options)->not->toHaveKey($grandchild->id)
        ->and($options[$root->id])->toBe('لباس')
        ->and(ShopCategoryResource::parentOptions(null)[$grandchild->id])->toBe('لباس › بادی › بادی نوزاد');

    Livewire::test(ListShopCategories::class)
        ->filterTable('parent', (string) $root->id)
        ->assertCanSeeTableRecords([$child, $sibling])
        ->assertCanNotSeeTableRecords([$root, $grandchild])
        ->call('reorderTable', [(string) $sibling->id, (string) $child->id]);

    expect($sibling->refresh()->sort_order)->toBeLessThan($child->refresh()->sort_order);

    Livewire::test(EditShopCategory::class, ['record' => $child->getRouteKey()])->assertActionDisabled('delete');
    Livewire::test(EditShopCategory::class, ['record' => $sibling->getRouteKey()])->assertActionEnabled('delete');
});

it('shows today\'s orders, revenue and low stock to shop managers only', function (): void {
    $low = Product::factory()->published()->create(['title' => 'کرم سوختگی پا', 'stock_qty' => 2]);
    $fine = Product::factory()->published()->create(['title' => 'حوله کلاه‌دار', 'stock_qty' => 40]);
    $variantLow = Product::factory()->published()->create(['title' => 'جوراب نوزاد']);
    ProductVariant::factory()->create(['product_id' => $variantLow->id, 'size' => 'S', 'stock_qty' => 1]);
    ProductVariant::factory()->create(['product_id' => $variantLow->id, 'size' => 'M', 'color' => null, 'stock_qty' => 30]);
    $preorder = Product::factory()->published()->create(['title' => 'گهواره پیش‌سفارش', 'stock_qty' => 0, 'stock_status' => StockStatus::PreOrder]);

    shopAdminOrder([[$fine, null, 2]]);
    shopAdminOrder([[$fine, null, 1]], 'cancelled');

    expect(ShopOrdersOverview::revenue(7)->toToman())->toBe(500_000)
        ->and(ShopOrdersOverview::revenue(30)->toToman())->toBe(500_000);

    $this->actingAs(shopAdmin());
    Livewire::test(LowStockProducts::class)
        ->assertCanSeeTableRecords([$low, $variantLow])
        ->assertCanNotSeeTableRecords([$fine, $preorder])
        ->assertSee('S: ۱');
    Livewire::test(ShopOrdersOverview::class)->assertSee('سفارش‌های امروز')->assertSee('۵۰۰٬۰۰۰ تومان');
    $this->get('/admin')->assertOk()->assertSeeLivewire(LowStockProducts::class)->assertSeeLivewire(ShopOrdersOverview::class);

    $this->actingAs(shopAdmin(AdminRole::Editor));
    expect(ShopOrdersOverview::canView())->toBeFalse()->and(LowStockProducts::canView())->toBeFalse();
    $this->get('/admin')->assertOk()->assertDontSeeLivewire(LowStockProducts::class)->assertDontSeeLivewire(ShopOrdersOverview::class);
});

it('edits shop settings in tomans and applies them to shipping, the COD cap and the low-stock threshold', function (): void {
    expect(app(ShippingRule::class)->flatFee)->toBeNull()
        ->and(app(PaymentGateway::class)->limit())->toBeNull();

    // config stays the fallback while a setting is empty
    config(['shop.cod_max_amount' => 5_000_000]);
    expect(app(PaymentGateway::class)->limit()?->toToman())->toBe(5_000_000);

    $this->actingAs($manager = shopAdmin());
    Livewire::test(ShopSettingsPage::class)
        ->fillForm(['shipping_flat_fee' => 45_000, 'free_shipping_over' => 900_000, 'cod_max_amount' => 20_000_000, 'low_stock_threshold' => 5])
        ->call('save')
        ->assertHasNoFormErrors();

    $rule = app(ShippingRule::class);
    expect($rule->flatFee?->rial)->toBe(450_000)
        ->and($rule->freeOver?->rial)->toBe(9_000_000)
        ->and(app(PaymentGateway::class)->limit()?->rial)->toBe(200_000_000)
        ->and(LowStock::threshold())->toBe(5)
        ->and(Activity::query()->where('description', 'settings.shop')->where('causer_id', $manager->id)->exists())->toBeTrue();

    // 0 = always free shipping
    app(UpdateSettings::class)->handle(SettingGroup::Shop, ['shipping_flat_fee' => 0]);
    expect(app(ShippingRule::class)->flatFee?->isZero())->toBeTrue();
});
