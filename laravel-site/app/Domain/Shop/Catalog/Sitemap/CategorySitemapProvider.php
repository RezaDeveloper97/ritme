<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Sitemap;

use App\Domain\Media\Contracts\MediaRepository;
use App\Domain\Seo\Sitemap\SitemapEntryData;
use App\Domain\Seo\Sitemap\SitemapProvider;
use App\Domain\Shop\Catalog\Queries\SitemapCategories;
use App\Domain\Shop\Catalog\Support\ShopUrls;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CacheKey;
use Carbon\CarbonImmutable;

/**
 * Sitemap `shop-categories`: visible category listings that contain real (non-demo) published products. Cached in
 * the `sitemap` namespace.
 *
 * @phpstan-import-type SitemapCategoryRow from SitemapCategories
 */
final class CategorySitemapProvider implements SitemapProvider
{
    public const KEY = 'shop-categories';

    public function __construct(
        private readonly CacheAside $cache,
        private readonly MediaRepository $media,
        private readonly ShopUrls $urls,
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
                loc: $this->urls->category($row['slug']),
                lastmod: $row['lastmod'] === null ? null : CarbonImmutable::parse($row['lastmod']),
                imageUrl: $cover === null ? null : $this->urls->absolute($cover->url),
                imageTitle: $cover === null ? null : ($cover->alt ?? $row['name']),
            );
        }, $rows);
    }

    /**
     * @return list<SitemapCategoryRow>
     */
    private function rows(): array
    {
        return $this->cache->remember(CacheKey::make('sitemap', self::KEY), null, static fn (): array => (new SitemapCategories)->get());
    }
}
