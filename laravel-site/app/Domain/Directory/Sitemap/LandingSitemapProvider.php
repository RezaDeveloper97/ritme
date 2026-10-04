<?php

declare(strict_types=1);

namespace App\Domain\Directory\Sitemap;

use App\Domain\Directory\Data\LandingComboData;
use App\Domain\Directory\Queries\LandingCombos;
use App\Domain\Directory\Support\DirectoryUrls;
use App\Domain\Seo\Sitemap\SitemapEntryData;
use App\Domain\Seo\Sitemap\SitemapProvider;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CacheKey;

/**
 * Sitemap `directory-landings`: `/directory/{city}` and `/directory/{city}/{category}` pages that have at least one
 * published, non-demo place (empty combinations are thin content and stay out). lastmod = newest place update.
 */
final class LandingSitemapProvider implements SitemapProvider
{
    public const KEY = 'directory-landings';

    public function __construct(private readonly CacheAside $cache, private readonly DirectoryUrls $urls) {}

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

        return array_map(fn (LandingComboData $combo): SitemapEntryData => new SitemapEntryData(
            loc: $combo->categorySlug === null ? $this->urls->city($combo->citySlug) : $this->urls->category($combo->citySlug, $combo->categorySlug),
            lastmod: $combo->lastmod,
        ), $rows);
    }

    /**
     * @return list<LandingComboData>
     */
    private function rows(): array
    {
        /** @var list<array<string, mixed>> $rows */
        $rows = $this->cache->remember(
            CacheKey::make('sitemap', 'directory-landings'),
            null,
            static fn (): array => array_map(static fn (LandingComboData $c): array => $c->toArray(), (new LandingCombos(includeDemo: false))->get()),
        );

        return array_map(LandingComboData::fromArray(...), $rows);
    }
}
