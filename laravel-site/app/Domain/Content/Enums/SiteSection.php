<?php

declare(strict_types=1);

namespace App\Domain\Content\Enums;

/**
 * Public site sections an admin can switch off (admin «صفحه‌ها» settings page). A disabled section answers 404 on
 * every route it owns and drops out of the main nav, footer, sitemap and search. Home, privacy and terms are not
 * sections: they are always on. The value is the settings key (group `pages`).
 */
enum SiteSection: string
{
    case Cycle = 'cycle';
    case Ttc = 'ttc';
    case Pregnancy = 'pregnancy';
    case Postpartum = 'postpartum';
    case Menopause = 'menopause';
    case Teen = 'teen';
    case Services = 'services';
    case Plus = 'plus';
    case Tools = 'tools';
    case About = 'about';
    case SocialResponsibility = 'social_responsibility';
    case Faq = 'faq';
    case Contact = 'contact';
    case Search = 'search';
    case Blog = 'blog';
    case Newsletter = 'newsletter';
    case Directory = 'directory';
    case Shop = 'shop';

    /** Exact route names owned by a single-page section. */
    private const ROUTES = [
        'stage.cycle' => self::Cycle,
        'stage.ttc' => self::Ttc,
        'stage.pregnancy' => self::Pregnancy,
        'stage.postpartum' => self::Postpartum,
        'stage.menopause' => self::Menopause,
        'stage.teen' => self::Teen,
        'services' => self::Services,
        'plus' => self::Plus,
        'tools' => self::Tools,
        'about' => self::About,
        'social-responsibility' => self::SocialResponsibility,
        'faq' => self::Faq,
        'contact' => self::Contact,
        'contact.store' => self::Contact,
        'search' => self::Search,
    ];

    /** Route-name prefixes owned by a multi-page section. */
    private const PREFIXES = [
        'blog.' => self::Blog,
        'newsletter.' => self::Newsletter,
        'directory.' => self::Directory,
        'shop.' => self::Shop,
    ];

    /** Sitemap provider key prefixes (SitemapProvider::key()). */
    private const SITEMAPS = [
        'posts' => self::Blog,
        'blog-' => self::Blog,
        'directory-' => self::Directory,
        'shop-' => self::Shop,
    ];

    /** Search provider keys (SearchProvider::key()). */
    private const SEARCH = [
        'posts' => self::Blog,
        'faq' => self::Faq,
        'places' => self::Directory,
        'products' => self::Shop,
    ];

    public static function forRoute(?string $routeName): ?self
    {
        if ($routeName === null || $routeName === '') {
            return null;
        }

        if (isset(self::ROUTES[$routeName])) {
            return self::ROUTES[$routeName];
        }

        foreach (self::PREFIXES as $prefix => $section) {
            if (str_starts_with($routeName, $prefix)) {
                return $section;
            }
        }

        return null;
    }

    public static function forPage(StaticPage $page): ?self
    {
        return self::forRoute($page->routeName());
    }

    public static function forSitemap(string $key): ?self
    {
        foreach (self::SITEMAPS as $prefix => $section) {
            if (str_starts_with($key, $prefix)) {
                return $section;
            }
        }

        return null;
    }

    public static function forSearchProvider(string $key): ?self
    {
        return self::SEARCH[$key] ?? null;
    }

    public function label(): string
    {
        return match ($this) {
            self::Cycle => 'چرخه و پریود',
            self::Ttc => 'اقدام به بارداری',
            self::Pregnancy => 'بارداری',
            self::Postpartum => 'پس از زایمان و کودک',
            self::Menopause => 'یائسگی',
            self::Teen => 'نوجوان و والدین',
            self::Services => 'خدمات',
            self::Plus => 'رایگان و پلاس',
            self::Tools => 'ابزارها',
            self::About => 'درباره ما',
            self::SocialResponsibility => 'مسئولیت اجتماعی',
            self::Faq => 'سؤالات متداول',
            self::Contact => 'تماس با ما',
            self::Search => 'جستجو',
            self::Blog => 'مجله (بلاگ)',
            self::Newsletter => 'خبرنامه',
            self::Directory => 'خدمات مادر و کودک (دایرکتوری)',
            self::Shop => 'فروشگاه',
        };
    }

    /** Admin helper text: what else goes away with the section. */
    public function hint(): ?string
    {
        return match ($this) {
            self::Blog => 'همه‌ی مقاله‌ها، دسته‌ها، برچسب‌ها، نویسنده‌ها و فید.',
            self::Newsletter => 'فرم عضویت و لینک‌های تأیید و لغو عضویت.',
            self::Directory => 'فهرست مراکز، صفحه‌ی هر مرکز، رزرو و ثبت کسب‌وکار.',
            self::Shop => 'محصولات، دسته‌ها، سبد خرید و ثبت سفارش.',
            self::Faq => 'سؤال‌ها از نتیجه‌ی جستجو هم حذف می‌شوند.',
            default => null,
        };
    }

    /** @return list<self> */
    public static function stages(): array
    {
        return [self::Cycle, self::Ttc, self::Pregnancy, self::Postpartum, self::Menopause, self::Teen];
    }

    /** @return list<self> */
    public static function pages(): array
    {
        return [self::Services, self::Plus, self::Tools, self::About, self::SocialResponsibility, self::Faq, self::Contact, self::Search];
    }

    /** @return list<self> */
    public static function modules(): array
    {
        return [self::Blog, self::Newsletter, self::Directory, self::Shop];
    }
}
