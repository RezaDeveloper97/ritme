<?php

declare(strict_types=1);

namespace App\Domain\Content\Enums;

/**
 * Registry of the parameterless marketing pages — the single source for the main nav, breadcrumbs, the sitemap
 * (L1-06) and the admin SEO page list (L7-01). The value is the route name (docs/AUDIT.md §1/§7).
 *
 * Parameterised pages (article, place, product, booked, order, category) are not cases: their nav item comes from
 * NavItem::forRoute() prefixes, their header is light, and their controllers build the breadcrumb trail.
 */
enum StaticPage: string
{
    case Home = 'home';
    case Cycle = 'stage.cycle';
    case Ttc = 'stage.ttc';
    case Pregnancy = 'stage.pregnancy';
    case Postpartum = 'stage.postpartum';
    case Menopause = 'stage.menopause';
    case Teen = 'stage.teen';
    case Services = 'services';
    case Plus = 'plus';
    case Tools = 'tools';
    case About = 'about';
    case SocialResponsibility = 'social-responsibility';
    case Privacy = 'privacy';
    case Terms = 'terms';
    case Faq = 'faq';
    case Contact = 'contact';
    case Blog = 'blog.index';
    case Directory = 'directory.index';
    case DirectoryBusiness = 'directory.business';
    case DirectoryJoin = 'directory.join';
    case DirectoryJoinDone = 'directory.join.done';
    case Shop = 'shop.index';
    case ShopCart = 'shop.cart';
    case ShopCheckout = 'shop.checkout';

    public static function forRoute(?string $routeName): ?self
    {
        return $routeName === null ? null : self::tryFrom($routeName);
    }

    /** @return list<self> */
    public static function stages(): array
    {
        return [self::Cycle, self::Ttc, self::Pregnancy, self::Postpartum, self::Menopause, self::Teen];
    }

    public function routeName(): string
    {
        return $this->value;
    }

    /** URL path, used until/unless the named route exists (L1-05 registers the routes). */
    public function path(): string
    {
        return match ($this) {
            self::Home => '/',
            self::Cycle => '/cycle',
            self::Ttc => '/ttc',
            self::Pregnancy => '/pregnancy',
            self::Postpartum => '/postpartum',
            self::Menopause => '/menopause',
            self::Teen => '/teen',
            self::Services => '/services',
            self::Plus => '/plus',
            self::Tools => '/tools',
            self::About => '/about',
            self::SocialResponsibility => '/social-responsibility',
            self::Privacy => '/privacy',
            self::Terms => '/terms',
            self::Faq => '/faq',
            self::Contact => '/contact',
            self::Blog => '/blog',
            self::Directory => '/directory',
            self::DirectoryBusiness => '/directory/business',
            self::DirectoryJoin => '/directory/join',
            self::DirectoryJoinDone => '/directory/join/done',
            self::Shop => '/shop',
            self::ShopCart => '/shop/cart',
            self::ShopCheckout => '/shop/checkout',
        };
    }

    /** Short page name: breadcrumbs, footer links, admin list. The SEO title is separate (SeoManager). */
    public function label(): string
    {
        return match ($this) {
            self::Home => 'خانه',
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
            self::Privacy => 'حریم خصوصی',
            self::Terms => 'شرایط استفاده',
            self::Faq => 'سؤالات متداول',
            self::Contact => 'تماس با ما',
            self::Blog => 'مجله',
            self::Directory => 'خدمات مادر و کودک',
            self::DirectoryBusiness => 'برای کسب‌وکارها',
            self::DirectoryJoin => 'ثبت مجموعه',
            self::DirectoryJoinDone => 'درخواست ثبت شد',
            self::Shop => 'فروشگاه',
            self::ShopCart => 'سبد خرید',
            self::ShopCheckout => 'تکمیل خرید',
        };
    }

    /** Active main-nav item (AUDIT §2.1): none on contact, faq, plus, privacy, social-responsibility, terms. */
    public function navItem(): ?NavItem
    {
        return match ($this) {
            self::Home => NavItem::Home,
            self::Cycle, self::Ttc, self::Pregnancy, self::Postpartum, self::Menopause, self::Teen => NavItem::Stages,
            self::Services, self::Directory, self::DirectoryBusiness, self::DirectoryJoin, self::DirectoryJoinDone => NavItem::Services,
            self::Shop, self::ShopCart, self::ShopCheckout => NavItem::Shop,
            self::Blog => NavItem::Blog,
            self::Tools => NavItem::Tools,
            self::About => NavItem::About,
            self::Plus, self::SocialResponsibility, self::Privacy, self::Terms, self::Faq, self::Contact => null,
        };
    }

    /** Previous breadcrumb step (null for the home page; Home for top-level pages). */
    public function breadcrumbParent(): ?self
    {
        return match ($this) {
            self::Home => null,
            self::DirectoryBusiness => self::Directory,
            self::DirectoryJoin, self::DirectoryJoinDone => self::DirectoryBusiness,
            self::ShopCart => self::Shop,
            self::ShopCheckout => self::ShopCart,
            default => self::Home,
        };
    }

    /**
     * Home → … → this page.
     *
     * @return list<self>
     */
    public function trail(): array
    {
        $trail = [$this];
        $page = $this;
        while (($page = $page->breadcrumbParent()) !== null) {
            array_unshift($trail, $page);
        }

        return $trail;
    }

    /** AUDIT §1: 10 pages put the header on the dark hero block; every other page is light. */
    public function headerVariant(): HeaderVariant
    {
        return match ($this) {
            self::Home, self::Cycle, self::Ttc, self::Pregnancy, self::Postpartum, self::Menopause, self::Teen,
            self::About, self::Privacy, self::SocialResponsibility => HeaderVariant::Dark,
            default => HeaderVariant::Light,
        };
    }

    /** The page renders the `#download` app CTA, so «دانلود اپ» can stay on the page (else it goes to `/#download`). */
    public function hasAppCta(): bool
    {
        return match ($this) {
            self::Home, self::Cycle, self::Ttc, self::Pregnancy, self::Postpartum, self::Menopause, self::Teen,
            self::Services, self::Plus, self::Tools, self::SocialResponsibility, self::Faq, self::Blog => true,
            default => false,
        };
    }

    /** Transactional / form-result pages stay out of the index and the sitemap. */
    public function indexable(): bool
    {
        return ! in_array($this, [self::DirectoryJoin, self::DirectoryJoinDone, self::ShopCart, self::ShopCheckout], true);
    }

    public function sitemapPriority(): float
    {
        return match ($this) {
            self::Home => 1.0,
            self::Cycle, self::Ttc, self::Pregnancy, self::Postpartum, self::Menopause, self::Teen => 0.9,
            self::Services, self::Tools, self::Blog, self::Shop, self::Directory => 0.8,
            self::Plus, self::About, self::Faq, self::DirectoryBusiness => 0.6,
            self::SocialResponsibility, self::Privacy, self::Contact => 0.5,
            self::Terms => 0.3,
            self::DirectoryJoin, self::DirectoryJoinDone, self::ShopCart, self::ShopCheckout => 0.0,
        };
    }

    /** sitemaps.org changefreq value. */
    public function sitemapChangefreq(): string
    {
        return match ($this) {
            self::Home, self::Blog, self::Shop, self::Directory => 'daily',
            self::Cycle, self::Ttc, self::Pregnancy, self::Postpartum, self::Menopause, self::Teen,
            self::Services, self::Tools, self::Faq => 'weekly',
            self::Privacy, self::Terms => 'yearly',
            default => 'monthly',
        };
    }
}
