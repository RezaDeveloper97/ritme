<?php

declare(strict_types=1);

namespace App\Domain\Seo\Sitemap;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Support\Robots;
use Illuminate\Contracts\Routing\UrlGenerator;
use Illuminate\Routing\Router;

/**
 * Sitemap `pages`: the parameterless marketing pages of the StaticPage registry that are indexable, have a registered
 * route and are not excluded by their seo_meta row (`sitemap_include` false or a `noindex` robots value).
 * Priority / changefreq come from seo_meta, falling back to the registry. Static pages have no content timestamp yet,
 * so lastmod is omitted rather than faked.
 */
final class PagesSitemapProvider implements SitemapProvider
{
    public const KEY = 'pages';

    public function __construct(
        private readonly SeoMetaRepository $seo,
        private readonly Router $router,
        private readonly UrlGenerator $url,
    ) {}

    public function key(): string
    {
        return self::KEY;
    }

    public function count(): int
    {
        return count($this->all());
    }

    public function entries(int $page = 1, int $perPage = 5000): array
    {
        return array_slice($this->all(), (max(1, $page) - 1) * max(1, $perPage), max(1, $perPage));
    }

    /**
     * @return list<SitemapEntryData>
     */
    private function all(): array
    {
        $entries = [];
        foreach (StaticPage::cases() as $page) {
            if (! $page->indexable() || ! $this->router->has($page->routeName())) {
                continue;
            }

            $meta = $this->seo->forRoute($page->routeName());
            if ($meta !== null && (! $meta->sitemapInclude || ! Robots::parse($meta->robots)->index)) {
                continue;
            }

            $entries[] = new SitemapEntryData(
                loc: $this->url->route($page->routeName()),
                priority: $meta->sitemapPriority ?? $page->sitemapPriority(),
                changefreq: $meta->sitemapChangefreq ?? $page->sitemapChangefreq(),
            );
        }

        return $entries;
    }
}
