<?php

declare(strict_types=1);

namespace App\Domain\Seo\Sitemap;

use App\Domain\Blog\Support\PostListIndexing;
use App\Domain\Content\Enums\StaticPage;
use App\Domain\Directory\Support\PlaceListIndexing;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Support\Robots;
use App\Domain\Shop\Catalog\Support\ProductListIndexing;
use Illuminate\Contracts\Routing\UrlGenerator;
use Illuminate\Routing\Router;

/**
 * Sitemap `pages`: the parameterless marketing pages of the StaticPage registry that are indexable, have a registered
 * route and are not excluded by their seo_meta row (`sitemap_include` false or a `noindex` robots value).
 * Listing pages whose controller answers `noindex` because they show nothing real (`/blog` without posts, `/directory`
 * and `/shop` with demo-only or no items) are skipped by the same rule (Post/Place/ProductListIndexing, L7-05b); the
 * `sitemap` namespace is bumped by the Post/Place/Product observers, so they reappear once real content exists.
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
        private readonly PostListIndexing $posts,
        private readonly PlaceListIndexing $places,
        private readonly ProductListIndexing $products,
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

            if (! $this->listingIndexable($page)) {
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

    /**
     * False for a listing page that would render `noindex` because it is empty or demo-only; true for every other page.
     */
    private function listingIndexable(StaticPage $page): bool
    {
        return match ($page) {
            StaticPage::Blog => $this->posts->indexIndexable(),
            StaticPage::Directory => $this->places->indexIndexable(),
            StaticPage::Shop => $this->products->homeIndexable(),
            default => true,
        };
    }
}
