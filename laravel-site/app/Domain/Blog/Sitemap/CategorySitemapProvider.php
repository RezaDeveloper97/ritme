<?php

declare(strict_types=1);

namespace App\Domain\Blog\Sitemap;

use App\Domain\Blog\Data\SitemapEntryData;
use App\Domain\Blog\Queries\SitemapCategories;
use App\Domain\Blog\Support\BlogUrls;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CacheKey;
use Carbon\CarbonImmutable;

/**
 * Sitemap `blog-categories`: categories with published posts, lastmod = newest post content date. Cached in the
 * `sitemap` namespace (bumped by post, category and seo_meta changes).
 *
 * @phpstan-import-type SitemapCategoryRow from SitemapCategories
 */
final class CategorySitemapProvider
{
    public const KEY = 'blog-categories';

    public function __construct(private readonly CacheAside $cache, private readonly BlogUrls $urls) {}

    public function key(): string
    {
        return self::KEY;
    }

    public function count(): int
    {
        return count($this->rows());
    }

    /**
     * @return list<SitemapEntryData>
     */
    public function entries(int $page = 1, int $perPage = 5000): array
    {
        return array_map(fn (array $row): SitemapEntryData => new SitemapEntryData(
            loc: $this->urls->category($row['slug']),
            lastmod: $row['lastmod'] === null ? null : CarbonImmutable::parse($row['lastmod']),
        ), array_slice($this->rows(), (max(1, $page) - 1) * $perPage, max(1, $perPage)));
    }

    /**
     * @return list<SitemapCategoryRow>
     */
    private function rows(): array
    {
        return $this->cache->remember(CacheKey::make('sitemap', 'blog-categories'), null, static fn (): array => (new SitemapCategories)->get());
    }
}
