<?php

declare(strict_types=1);

namespace App\Domain\Blog\Sitemap;

use App\Domain\Blog\Queries\SitemapTaxonomy;
use App\Domain\Blog\Support\BlogUrls;
use App\Domain\Seo\Sitemap\SitemapEntryData;
use App\Domain\Seo\Sitemap\SitemapProvider;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CacheKey;
use Carbon\CarbonImmutable;

/**
 * Sitemap `blog-authors`: author pages with at least one published post, cached in the `sitemap` namespace.
 *
 * @phpstan-import-type SitemapTaxonomyRow from SitemapTaxonomy
 */
final class AuthorSitemapProvider implements SitemapProvider
{
    public const KEY = 'blog-authors';

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
            loc: $this->urls->author($row['slug']),
            lastmod: $row['lastmod'] === null ? null : CarbonImmutable::parse($row['lastmod']),
        ), array_slice($this->rows(), (max(1, $page) - 1) * $perPage, max(1, $perPage)));
    }

    /**
     * @return list<SitemapTaxonomyRow>
     */
    private function rows(): array
    {
        return $this->cache->remember(CacheKey::make('sitemap', 'blog-authors'), null, static fn (): array => SitemapTaxonomy::forAuthors());
    }
}
