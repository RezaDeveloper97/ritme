<?php

declare(strict_types=1);

namespace App\Domain\Content\Enums;

/**
 * Main navigation items, in display order (docs/AUDIT.md §2.1). Which item is active is decided by
 * StaticPage::navItem() for registered pages and by route-name prefixes for parameterised pages (blog.show …).
 */
enum NavItem: string
{
    case Home = 'home';
    case Stages = 'stages';
    case Services = 'services';
    case Shop = 'shop';
    case Blog = 'blog';
    case Tools = 'tools';
    case About = 'about';

    /** Route-name prefix => item, for routes that are not StaticPage cases (detail pages, filtered lists). */
    private const PREFIXES = [
        'stage.' => self::Stages,
        'directory.' => self::Services,
        'shop.' => self::Shop,
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
            self::Stages => 'مرحله‌ها',
            self::Services => 'خدمات',
            self::Shop => 'فروشگاه',
            self::Blog => 'مجله',
            self::Tools => 'ابزارها',
            self::About => 'درباره ما',
        };
    }

    /** The page the item links to. «مرحله‌ها» has no overview page in the design: it opens the first stage. */
    public function page(): StaticPage
    {
        return match ($this) {
            self::Home => StaticPage::Home,
            self::Stages => StaticPage::Cycle,
            self::Services => StaticPage::Services,
            self::Shop => StaticPage::Shop,
            self::Blog => StaticPage::Blog,
            self::Tools => StaticPage::Tools,
            self::About => StaticPage::About,
        };
    }

    /** «مرحله‌ها» carries a decorative chevron in the design; there is no dropdown. */
    public function hasChevron(): bool
    {
        return $this === self::Stages;
    }
}
