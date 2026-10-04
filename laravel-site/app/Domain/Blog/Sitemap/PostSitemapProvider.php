<?php

declare(strict_types=1);

namespace App\Domain\Blog\Sitemap;

use App\Domain\Blog\Queries\SitemapPosts;
use App\Domain\Blog\Support\BlogUrls;
use App\Domain\Media\Contracts\MediaRepository;
use App\Domain\Seo\Sitemap\SitemapEntryData;
use App\Domain\Seo\Sitemap\SitemapProvider;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CacheKey;
use Carbon\CarbonImmutable;

/**
 * Sitemap `posts`: every indexable published post with lastmod (content date) and its cover as image:image.
 * Rows are cached in the `sitemap` namespace (bumped by post and seo_meta changes); cover URLs come from the cached
 * MediaRepository at build time.
 *
 * @phpstan-import-type SitemapPostRow from SitemapPosts
 */
final class PostSitemapProvider implements SitemapProvider
{
    public const KEY = 'posts';

    public function __construct(
        private readonly CacheAside $cache,
        private readonly MediaRepository $media,
        private readonly BlogUrls $urls,
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
        $rows = array_slice($this->rows(), (max(1, $page) - 1) * $perPage, max(1, $perPage));
        $mediaIds = array_values(array_filter(array_map(static fn (array $row): ?int => $row['coverMediaId'], $rows)));
        $covers = $mediaIds === [] ? [] : $this->media->findMany($mediaIds);

        return array_map(function (array $row) use ($covers): SitemapEntryData {
            $cover = $row['coverMediaId'] === null ? null : ($covers[$row['coverMediaId']] ?? null);

            return new SitemapEntryData(
                loc: $this->urls->post($row['slug']),
                lastmod: $row['lastmod'] === null ? null : CarbonImmutable::parse($row['lastmod']),
                imageUrl: $cover === null ? null : $this->urls->absolute($cover->url),
                imageTitle: $cover === null ? null : ($cover->alt ?? $row['title']),
            );
        }, $rows);
    }

    /**
     * @return list<SitemapPostRow>
     */
    private function rows(): array
    {
        return $this->cache->remember(CacheKey::make('sitemap', 'blog-posts'), null, static fn (): array => (new SitemapPosts)->get());
    }
}
