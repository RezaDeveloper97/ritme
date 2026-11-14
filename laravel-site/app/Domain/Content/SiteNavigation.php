<?php

declare(strict_types=1);

namespace App\Domain\Content;

use App\Domain\Content\Data\FooterColumnData;
use App\Domain\Content\Data\LinkData;
use App\Domain\Content\Data\NavLinkData;
use App\Domain\Content\Enums\NavItem;
use App\Domain\Content\Enums\StaticPage;
use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use Illuminate\Routing\Router;
use Illuminate\Routing\UrlGenerator;

/**
 * Builds the site navigation DTOs (main nav, footer columns, breadcrumb trails) from the StaticPage registry.
 *
 * Links use the named route when it is registered and fall back to the registry path, so the shell renders
 * before every page route exists. Nav/footer URLs are root-relative (cacheable across hosts); breadcrumb URLs are
 * absolute because they also feed the BreadcrumbList JSON-LD.
 */
final class SiteNavigation
{
    public function __construct(
        private readonly Router $router,
        private readonly UrlGenerator $url,
        private readonly SiteSections $sections,
    ) {}

    public function url(StaticPage $page, string $fragment = '', bool $absolute = false): string
    {
        $url = $this->router->has($page->routeName())
            ? $this->url->route($page->routeName(), [], $absolute)
            : ($absolute ? $this->url->to($page->path()) : $page->path());

        return $fragment === '' ? $url : $url.'#'.ltrim($fragment, '#');
    }

    /**
     * @return list<NavLinkData>
     */
    public function main(?string $routeName): array
    {
        $active = NavItem::forRoute($routeName);

        $links = [];
        foreach (NavItem::cases() as $item) {
            $page = $this->navTarget($item);
            if ($page === null) {
                continue;
            }

            $links[] = new NavLinkData(
                label: $item->label(),
                url: $this->url($page, $item->fragment()),
                active: $item === $active,
                current: $routeName !== null && $item->fragment() === '' && $page->routeName() === $routeName,
                chevron: $item->hasChevron(),
            );
        }

        return $links;
    }

    /** The page a nav item opens, or null when it is switched off in the admin «صفحه‌ها» settings. */
    private function navTarget(NavItem $item): ?StaticPage
    {
        $page = $item->page();

        return $this->sections->pageEnabled($page) ? $page : null;
    }

    /**
     * «دانلود اپ» target: the page's own `#download` app CTA when it renders one, else the home page's.
     */
    public function downloadHref(?string $routeName, ?bool $pageHasAppCta = null): string
    {
        $hasCta = $pageHasAppCta ?? StaticPage::forRoute($routeName)?->hasAppCta() ?? false;

        return $hasCta ? '#download' : $this->url(StaticPage::Home, 'download');
    }

    /**
     * Footer link columns (Nov 2026 redesign: مرحله‌ها · ریتمی · اعتماد). Enamad and socials come from settings.
     *
     * @return list<FooterColumnData>
     */
    public function footer(): array
    {
        // Links into switched-off sections become null and are dropped below, then columns left empty.
        $link = fn (StaticPage $page, ?string $label = null, string $fragment = ''): ?LinkData => $this->sections->pageEnabled($page)
            ? new LinkData($label ?? $page->label(), $this->url($page, $fragment))
            : null;

        $columns = [
            'مرحله‌ها' => array_map(static fn (StaticPage $page): ?LinkData => $link($page), StaticPage::stages()),
            'ریتمی' => [
                $link(StaticPage::About),
                $link(StaticPage::SocialResponsibility),
                $link(StaticPage::Plus),
                $link(StaticPage::Blog),
                $link(StaticPage::Contact),
            ],
            'اعتماد' => [
                $link(StaticPage::Privacy),
                $link(StaticPage::Faq),
                $link(StaticPage::Terms),
            ],
        ];

        $visible = [];
        foreach ($columns as $title => $links) {
            $links = array_values(array_filter($links));
            if ($links !== []) {
                $visible[] = new FooterColumnData($title, $links);
            }
        }

        return $visible;
    }

    /**
     * Home → … → page, absolute URLs; the current page is last and has no URL.
     *
     * @return list<BreadcrumbItem>
     */
    public function breadcrumbs(StaticPage $page): array
    {
        return array_map(
            fn (StaticPage $step): BreadcrumbItem => new BreadcrumbItem(
                $step->label(),
                $step === $page ? null : $this->url($step, absolute: true),
            ),
            $page->trail(),
        );
    }
}
