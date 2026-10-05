<?php

declare(strict_types=1);

namespace App\Domain\Seo\Sitemap;

use App\Domain\Seo\Indexing\IndexingRules;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CacheKey;
use Carbon\CarbonImmutable;

/**
 * Builds `/sitemap.xml` (index of every non-empty provider file) and `/sitemaps/{key}[-{page}].xml` (page 1 has no
 * suffix), PER_PAGE URLs per file. Rendered documents are cached in the `sitemap` namespace, which content observers
 * bump, so a warm request runs no query. Locs are rewritten onto the canonical https origin (app.url).
 * Admin indexing controls (L7-04, IndexingRules): excluded / noindex types are left out (their file 404s), and a
 * per-type priority / changefreq overrides the provider's values. Settings writes bump `sitemap`.
 */
final class Sitemaps
{
    public const PER_PAGE = 5000;

    public function __construct(
        private readonly SitemapRegistry $registry,
        private readonly SitemapXml $xml,
        private readonly SitemapUrl $urls,
        private readonly CacheAside $cache,
        private readonly IndexingRules $indexing,
    ) {}

    public function index(): string
    {
        return $this->cache->remember(CacheKey::make('sitemap', 'doc', 'index'), null, function (): string {
            $files = [];
            foreach ($this->registry->all() as $key => $provider) {
                if ($this->indexing->excludesSitemap($key)) {
                    continue;
                }
                $pages = $this->pageCount($provider);
                for ($page = 1; $page <= $pages; $page++) {
                    $files[] = [
                        'loc' => $this->urls->file(self::path($key, $page)),
                        'lastmod' => self::newest($this->entries($provider, $page)),
                    ];
                }
            }

            return $this->xml->index($files);
        });
    }

    /**
     * The urlset document of one file, or null when the provider or page does not exist. Page 1 of an empty provider
     * is a valid empty urlset (the index simply leaves it out).
     */
    public function file(string $key, int $page = 1): ?string
    {
        $provider = $this->registry->find($key);
        if ($provider === null || $page < 1 || $this->indexing->excludesSitemap($key)) {
            return null;
        }

        return $this->cache->remember(CacheKey::make('sitemap', 'doc', $key, $page), null, function () use ($provider, $page): string {
            return $page > max(1, $this->pageCount($provider)) ? '' : $this->xml->urlset($this->entries($provider, $page));
        }) ?: null;
    }

    /**
     * Parses a file name without `.xml`: `pages` → [pages, 1], `posts-2` → [posts, 2]. `posts-1` is not canonical
     * (page 1 has no suffix) and yields null.
     *
     * @return array{0: string, 1: int}|null
     */
    public static function parse(string $file): ?array
    {
        if (preg_match('/^([a-z](?:[a-z0-9-]*[a-z])?)(?:-([1-9][0-9]*))?$/', $file, $m) !== 1) {
            return null;
        }
        $page = isset($m[2]) ? (int) $m[2] : 1;
        if (isset($m[2]) && $page === 1) {
            return null;
        }

        return [$m[1], $page];
    }

    public static function path(string $key, int $page): string
    {
        return '/sitemaps/'.$key.($page > 1 ? '-'.$page : '').'.xml';
    }

    private function pageCount(SitemapProvider $provider): int
    {
        return (int) ceil($provider->count() / self::PER_PAGE);
    }

    /**
     * @return list<SitemapEntryData>
     */
    private function entries(SitemapProvider $provider, int $page): array
    {
        $priority = $this->indexing->sitemapPriority($provider->key());
        $changefreq = $this->indexing->sitemapChangefreq($provider->key());

        return array_map(fn (SitemapEntryData $entry): SitemapEntryData => new SitemapEntryData(
            loc: $this->urls->page($entry->loc),
            lastmod: $entry->lastmod,
            imageUrl: $entry->imageUrl === null ? null : $this->urls->file($entry->imageUrl),
            imageTitle: $entry->imageTitle,
            priority: $priority ?? $entry->priority,
            changefreq: $changefreq ?? $entry->changefreq,
        ), $provider->entries($page, self::PER_PAGE));
    }

    /**
     * @param  list<SitemapEntryData>  $entries
     */
    private static function newest(array $entries): ?CarbonImmutable
    {
        $newest = null;
        foreach ($entries as $entry) {
            if ($entry->lastmod !== null && ($newest === null || $entry->lastmod->greaterThan($newest))) {
                $newest = $entry->lastmod;
            }
        }

        return $newest;
    }
}
