<?php

declare(strict_types=1);

namespace App\Providers\Domain;

use App\Domain\Contact\Support\ReplyChannel;
use App\Domain\Media\Actions\FindMediaUsages;
use App\Domain\Search\Support\SearchRegistry;
use App\Domain\Seo\Sitemap\SitemapRegistry;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\ShopSettings;
use App\Domain\Settings\Enums\SettingGroup;
use App\Domain\Shop\Cart\Contracts\CartCatalog;
use App\Domain\Shop\Cart\Contracts\CartRepository;
use App\Domain\Shop\Cart\Data\ShippingRule;
use App\Domain\Shop\Cart\Queries\LiveCartCatalog;
use App\Domain\Shop\Cart\Repositories\SessionCartRepository;
use App\Domain\Shop\Catalog\Contracts\CatalogRepository;
use App\Domain\Shop\Catalog\Contracts\ProductRepository;
use App\Domain\Shop\Catalog\Models\Brand;
use App\Domain\Shop\Catalog\Models\Category;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductReview;
use App\Domain\Shop\Catalog\Models\ProductVariant;
use App\Domain\Shop\Catalog\Observers\ProductObserver;
use App\Domain\Shop\Catalog\Observers\ProductReviewObserver;
use App\Domain\Shop\Catalog\Observers\ProductVariantObserver;
use App\Domain\Shop\Catalog\Observers\TaxonomyObserver;
use App\Domain\Shop\Catalog\Repositories\CachedCatalogRepository;
use App\Domain\Shop\Catalog\Repositories\CachedProductRepository;
use App\Domain\Shop\Catalog\Repositories\EloquentCatalogRepository;
use App\Domain\Shop\Catalog\Repositories\EloquentProductRepository;
use App\Domain\Shop\Catalog\Search\ProductSearchProvider;
use App\Domain\Shop\Catalog\Sitemap\CategorySitemapProvider;
use App\Domain\Shop\Catalog\Sitemap\ProductSitemapProvider;
use App\Domain\Shop\Catalog\Support\ProductContent;
use App\Domain\Shop\Ordering\Events\OrderPlaced;
use App\Domain\Shop\Ordering\Listeners\SendOrderNotifications;
use App\Domain\Shop\Ordering\Support\CheckoutSession;
use App\Domain\Shop\Payment\Contracts\PaymentGateway;
use App\Domain\Shop\Payment\Gateways\CashOnDeliveryGateway;
use App\Providers\DomainServiceProvider;
use Illuminate\Cache\RateLimiting\Limit;
use Illuminate\Contracts\Foundation\Application;
use Illuminate\Database\Eloquent\Relations\Relation;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Event;
use Illuminate\Support\Facades\RateLimiter;
use Throwable;

final class ShopServiceProvider extends DomainServiceProvider
{
    public const CHECKOUT_PER_10_MINUTES = 5;

    public const CHECKOUT_PER_DAY = 20;

    public const CHECKOUT_PER_MOBILE_PER_DAY = 10;

    protected array $repositories = [
        CatalogRepository::class => [EloquentCatalogRepository::class, CachedCatalogRepository::class],
        ProductRepository::class => [EloquentProductRepository::class, CachedProductRepository::class],
    ];

    protected array $observers = [
        Product::class => ProductObserver::class,
        ProductVariant::class => ProductVariantObserver::class,
        ProductReview::class => ProductReviewObserver::class,
        Category::class => TaxonomyObserver::class,
        Brand::class => TaxonomyObserver::class,
    ];

    public function register(): void
    {
        parent::register();

        // Cart (L6-04): session-scoped, live catalog reads (never the `shop` cache). Shipping estimate from the shop
        // settings (L6-06, tomans) with `shop.shipping.flat_fee` / `shop.shipping.free_over` as the fallback. Bound
        // (not singleton) so an admin change applies to the next resolve; the settings read is cached.
        $this->app->scoped(CartRepository::class, SessionCartRepository::class);
        $this->app->bind(CartCatalog::class, LiveCartCatalog::class);
        $this->app->bind(ShippingRule::class, static function (Application $app): ShippingRule {
            $shop = self::shopSettings($app);

            return ShippingRule::fromToman(
                $shop->shippingFlatFee ?? $app['config']->get('shop.shipping.flat_fee'),
                $shop->freeShippingOver ?? $app['config']->get('shop.shipping.free_over'),
            );
        });

        // Checkout (L6-05): cash on delivery only (tasks/README.md decision) behind the PaymentGateway contract. COD cap
        // from ShopSettings.cod_max_amount (L6-06, tomans) with `shop.cod_max_amount` as the fallback (null = no cap).
        $this->app->bind(PaymentGateway::class, static fn (Application $app): PaymentGateway => CashOnDeliveryGateway::fromToman(
            self::shopSettings($app)->codMaxAmount ?? $app['config']->get('shop.cod_max_amount'),
        ));
        $this->app->scoped(CheckoutSession::class);

        $this->app->singleton(ProductContent::class, static fn (Application $app): ProductContent => ProductContent::fromConfig($app['config']));

        // `/sitemaps/shop-products.xml`, `/sitemaps/shop-categories.xml` (L1-06 registry).
        $this->app->tag([ProductSitemapProvider::class, CategorySitemapProvider::class], SitemapRegistry::TAG);

        // Products in `/search` (L4-04 registry).
        $this->app->tag([ProductSearchProvider::class], SearchRegistry::TAG);
    }

    /**
     * The shop settings group; defaults (everything unset → config fallback) when settings cannot be read yet
     * (fresh install before the migrations ran).
     */
    private static function shopSettings(Application $app): ShopSettings
    {
        try {
            $shop = $app->make(SettingsRepository::class)->group(SettingGroup::Shop);
        } catch (Throwable) {
            return new ShopSettings;
        }

        return $shop instanceof ShopSettings ? $shop : new ShopSettings;
    }

    public function boot(): void
    {
        parent::boot();

        // Orders → queued team mail + customer SMS (after commit).
        Event::listen(OrderPlaced::class, SendOrderNotifications::class);

        // POST /shop/checkout: per IP (a household may order twice) and per mobile.
        RateLimiter::for('shop-checkout', static function (Request $request): array {
            $ip = (string) $request->ip();
            $mobile = $request->input('mobile');
            $mobile = is_string($mobile) && ! str_contains($mobile, '@') ? ReplyChannel::parse($mobile)?->phone : null;

            $limits = [
                Limit::perMinutes(10, self::CHECKOUT_PER_10_MINUTES)->by('shop-checkout:m:'.$ip),
                Limit::perDay(self::CHECKOUT_PER_DAY)->by('shop-checkout:d:'.$ip),
            ];
            if ($mobile !== null) {
                $limits[] = Limit::perDay(self::CHECKOUT_PER_MOBILE_PER_DAY)->by('shop-checkout:p:'.$mobile);
            }

            return $limits;
        });

        // Stable morph names for seo_meta.seoable_type (class names may move).
        Relation::morphMap([
            'shop_product' => Product::class,
            'shop_category' => Category::class,
        ]);

        // Media in use must never be offered for bulk deletion (admin media library, L2-03).
        FindMediaUsages::column('shop_products', 'cover_media_id', 'تصویر اصلی محصول', 'title');
        FindMediaUsages::column('shop_product_media', 'media_id', 'گالری محصول', 'product_id');
        FindMediaUsages::column('shop_categories', 'cover_media_id', 'تصویر دسته فروشگاه', 'name');
        FindMediaUsages::column('shop_brands', 'logo_media_id', 'لوگوی برند', 'name');
        FindMediaUsages::html('shop_products', 'description', 'تصویر داخل توضیح محصول', 'title');
    }
}
