<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Crawl;

use App\Domain\Seo\Sitemap\Sitemaps;

/**
 * Every page path listed in the served sitemaps (index → provider files), read from the same cached documents
 * `/sitemap.xml` serves, so admin exclusions and noindex types are already applied.
 */
final class SitemapPaths
{
    public function __construct(private readonly Sitemaps $sitemaps, private readonly SiteUrls $urls) {}

    /**
     * @return list<string> decoded paths
     */
    public function all(): array
    {
        $paths = [];
        foreach (self::locs($this->sitemaps->index()) as $fileUrl) {
            $name = basename((string) parse_url($fileUrl, PHP_URL_PATH), '.xml');
            $parsed = Sitemaps::parse($name);
            if ($parsed === null) {
                continue;
            }
            foreach (self::locs((string) $this->sitemaps->file($parsed[0], $parsed[1])) as $loc) {
                $resolved = $this->urls->resolve($loc);
                if ($resolved !== null) {
                    $paths[] = $resolved['path'].($resolved['query'] === '' ? '' : '?'.$resolved['query']);
                }
            }
        }

        return array_values(array_unique($paths));
    }

    /**
     * @return list<string>
     */
    private static function locs(string $xml): array
    {
        preg_match_all('#<loc>\s*(.*?)\s*</loc>#s', $xml, $matches);

        return array_map(static fn (string $loc): string => html_entity_decode($loc, ENT_QUOTES | ENT_XML1, 'UTF-8'), $matches[1]);
    }
}
