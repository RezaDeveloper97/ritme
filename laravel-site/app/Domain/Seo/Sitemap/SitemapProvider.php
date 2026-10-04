<?php

declare(strict_types=1);

namespace App\Domain\Seo\Sitemap;

/**
 * One family of sitemap files (`/sitemaps/{key}.xml`, `/sitemaps/{key}-{page}.xml`). Contexts register theirs by
 * tagging the class with SitemapRegistry::TAG in their service provider:
 *
 *     $this->app->tag([ProductSitemapProvider::class], SitemapRegistry::TAG);
 *
 * Providers return only indexable URLs (published, no `noindex` robots, `sitemap_include` true) with absolute locs
 * and real lastmod dates. They should keep their rows cached in the `sitemap` namespace and have their observers bump
 * it; the rendered documents are cached on top of that by Sitemaps.
 */
interface SitemapProvider
{
    /** URL-safe file key: lowercase letters, digits and inner hyphens, never ending in a digit (`blog-categories`). */
    public function key(): string;

    /** Number of entries (files are split every Sitemaps::PER_PAGE entries). */
    public function count(): int;

    /**
     * @return list<SitemapEntryData>
     */
    public function entries(int $page = 1, int $perPage = 5000): array;
}
