<?php

declare(strict_types=1);

namespace App\Domain\Seo\Indexing;

/**
 * The content types the indexing controls address, keyed by their sitemap provider key (SitemapProvider::key()).
 * Each knows the public route names of its pages, so a per-type robots default applies to exactly those pages.
 * A type set to noindex is also left out of the sitemap index.
 */
enum IndexingType: string
{
    case Pages = 'pages';
    case Posts = 'posts';
    case BlogCategories = 'blog-categories';
    case BlogTags = 'blog-tags';
    case BlogAuthors = 'blog-authors';
    case DirectoryPlaces = 'directory-places';
    case DirectoryLandings = 'directory-landings';
    case ShopProducts = 'shop-products';
    case ShopCategories = 'shop-categories';

    /** Robots meta defaults an admin can pick per type ('' = site default index,follow). */
    public const ROBOTS_OPTIONS = [
        'index,follow' => 'ایندکس شود (پیش‌فرض)',
        'noindex,follow' => 'ایندکس نشود، لینک‌ها دنبال شوند',
        'noindex,nofollow' => 'ایندکس نشود، لینک‌ها دنبال نشوند',
    ];

    public const CHANGEFREQ_OPTIONS = [
        'always' => 'همیشه', 'hourly' => 'ساعتی', 'daily' => 'روزانه', 'weekly' => 'هفتگی',
        'monthly' => 'ماهانه', 'yearly' => 'سالانه', 'never' => 'هرگز',
    ];

    public function label(): string
    {
        return match ($this) {
            self::Pages => 'صفحه‌های ثابت',
            self::Posts => 'مقاله‌های مجله',
            self::BlogCategories => 'دسته‌های مجله',
            self::BlogTags => 'برچسب‌های مجله',
            self::BlogAuthors => 'نویسنده‌های مجله',
            self::DirectoryPlaces => 'مکان‌های راهنما',
            self::DirectoryLandings => 'صفحه‌های شهر و دسته راهنما',
            self::ShopProducts => 'محصولات فروشگاه',
            self::ShopCategories => 'دسته‌های فروشگاه',
        };
    }

    /**
     * Route names of this type's pages. Static pages have their own per-page robots (L7-01), so no type default.
     *
     * @return list<string>
     */
    public function routes(): array
    {
        return match ($this) {
            self::Pages => [],
            self::Posts => ['blog.show'],
            self::BlogCategories => ['blog.category'],
            self::BlogTags => ['blog.tag'],
            self::BlogAuthors => ['blog.author'],
            self::DirectoryPlaces => ['directory.place'],
            self::DirectoryLandings => ['directory.city', 'directory.category'],
            self::ShopProducts => ['shop.product'],
            self::ShopCategories => ['shop.category'],
        };
    }

    public static function forRoute(?string $routeName): ?self
    {
        if ($routeName === null) {
            return null;
        }

        foreach (self::cases() as $type) {
            if (in_array($routeName, $type->routes(), true)) {
                return $type;
            }
        }

        return null;
    }

    /**
     * @return array<string, string> value => Persian label
     */
    public static function options(): array
    {
        $options = [];
        foreach (self::cases() as $type) {
            $options[$type->value] = $type->label();
        }

        return $options;
    }

    /**
     * @return array<string, string>
     */
    public static function robotsTypeOptions(): array
    {
        return array_filter(self::options(), static fn (string $key): bool => self::from($key)->routes() !== [], ARRAY_FILTER_USE_KEY);
    }
}
