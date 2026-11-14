<?php

declare(strict_types=1);

namespace App\Domain\Content\Enums;

/**
 * Main navigation items, in display order (Nov 2026 redesign: خانه · امکانات · مجله · درباره ما · سؤالات متداول).
 * Which item is active is decided by StaticPage::navItem() for registered pages and by route-name prefixes for
 * parameterised pages (blog.show …). Services, shop, directory and tools are reached from the pages and the footer.
 */
enum NavItem: string
{
    case Home = 'home';
    case Features = 'features';
    case Blog = 'blog';
    case About = 'about';
    case Faq = 'faq';

    /** Route-name prefix => item, for routes that are not StaticPage cases (detail pages, filtered lists). */
    private const PREFIXES = [
        'stage.' => self::Features,
        'blog.' => self::Blog,
    ];

    public static function forRoute(?string $routeName): ?self
    {
        if ($routeName === null || $routeName === '') {
            return null;
        }

        $page = StaticPage::tryFrom($routeName);
        if ($page !== null) {
            return $page->navItem();
        }

        foreach (self::PREFIXES as $prefix => $item) {
            if (str_starts_with($routeName, $prefix)) {
                return $item;
            }
        }

        return null;
    }

    public function label(): string
    {
        return match ($this) {
            self::Home => 'خانه',
            self::Features => 'امکانات',
            self::Blog => 'مجله',
            self::About => 'درباره ما',
            self::Faq => 'سؤالات متداول',
        };
    }

    /** The page the item links to. «امکانات» is the feature grid of the home page (fragment()). */
    public function page(): StaticPage
    {
        return match ($this) {
            self::Home, self::Features => StaticPage::Home,
            self::Blog => StaticPage::Blog,
            self::About => StaticPage::About,
            self::Faq => StaticPage::Faq,
        };
    }

    /** Fragment appended to the page URL (`#features` for «امکانات»). */
    public function fragment(): string
    {
        return $this === self::Features ? 'features' : '';
    }

    /** No item has a dropdown in the redesign; kept for the NavLinkData contract. */
    public function hasChevron(): bool
    {
        return false;
    }
}
