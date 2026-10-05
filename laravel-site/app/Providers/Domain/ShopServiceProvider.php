<?php

declare(strict_types=1);

namespace App\Providers\Domain;

use App\Domain\Media\Actions\FindMediaUsages;
use App\Domain\Search\Support\SearchRegistry;
use App\Domain\Seo\Sitemap\SitemapRegistry;
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
use App\Providers\DomainServiceProvider;
use Illuminate\Contracts\Foundation\Application;
use Illuminate\Database\Eloquent\Relations\Relation;

final class ShopServiceProvider extends DomainServiceProvider
{
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

        // Cart (L6-04): session-scoped, live catalog reads (never the `shop` cache). Shipping estimate from
        // `shop.shipping.flat_fee` / `shop.shipping.free_over` (tomans, optional) until a shop settings group exists (L6-06).
        $this->app->scoped(CartRepository::class, SessionCartRepository::class);
        $this->app->bind(CartCatalog::class, LiveCartCatalog::class);
        $this->app->singleton(ShippingRule::class, static fn (Application $app): ShippingRule => ShippingRule::fromToman(
            $app['config']->get('shop.shipping.flat_fee'),
            $app['config']->get('shop.shipping.free_over'),
        ));

        $this->app->singleton(ProductContent::class, static fn (Application $app): ProductContent => ProductContent::fromConfig($app['config']));

        // `/sitemaps/shop-products.xml`, `/sitemaps/shop-categories.xml` (L1-06 registry).
        $this->app->tag([ProductSitemapProvider::class, CategorySitemapProvider::class], SitemapRegistry::TAG);

        // Products in `/search` (L4-04 registry).
        $this->app->tag([ProductSearchProvider::class], SearchRegistry::TAG);
    }

    public function boot(): void
    {
        parent::boot();

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
