<?php

declare(strict_types=1);

namespace App\Domain\Blog\Sitemap;

use App\Domain\Blog\Queries\SitemapTaxonomy;
use App\Domain\Blog\Support\BlogUrls;
use App\Domain\Blog\Support\TagIndexing;
use App\Domain\Seo\Sitemap\SitemapEntryData;
use App\Domain\Seo\Sitemap\SitemapProvider;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CacheKey;
use Carbon\CarbonImmutable;

/**
 * Sitemap `blog-tags`: tag pages that are indexable (TagIndexing threshold), cached in the `sitemap` namespace.
 *
 * @phpstan-import-type SitemapTaxonomyRow from SitemapTaxonomy
 */
final class TagSitemapProvider implements SitemapProvider
{
    public const KEY = 'blog-tags';

    public function __construct(
        private readonly CacheAside $cache,
        private readonly BlogUrls $urls,
        private readonly TagIndexing $indexing,
    ) {}

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
            loc: $this->urls->tag($row['slug']),
            lastmod: $row['lastmod'] === null ? null : CarbonImmutable::parse($row['lastmod']),
        ), array_slice($this->rows(), (max(1, $page) - 1) * $perPage, max(1, $perPage)));
    }

    /**
     * @return list<SitemapTaxonomyRow>
     */
    private function rows(): array
    {
        $min = $this->indexing->minPosts();

        return $this->cache->remember(CacheKey::make('sitemap', 'blog-tags', $min), null, static fn (): array => SitemapTaxonomy::forTags($min));
    }
}
