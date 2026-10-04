<?php

declare(strict_types=1);

namespace App\Domain\Seo\Sitemap;

use Carbon\CarbonImmutable;

/**
 * Renders sitemaps.org documents (sitemapindex / urlset with the Google image extension). Every value is XML-escaped;
 * URLs must already be absolute.
 */
final class SitemapXml
{
    private const CHANGEFREQ = ['always', 'hourly', 'daily', 'weekly', 'monthly', 'yearly', 'never'];

    /**
     * @param  list<array{loc: string, lastmod: CarbonImmutable|null}>  $files
     */
    public function index(array $files): string
    {
        $xml = '<?xml version="1.0" encoding="UTF-8"?>'."\n"
            .'<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">'."\n";

        foreach ($files as $file) {
            $xml .= '  <sitemap>'."\n"
                .'    <loc>'.self::escape($file['loc']).'</loc>'."\n"
                .($file['lastmod'] === null ? '' : '    <lastmod>'.$file['lastmod']->toAtomString().'</lastmod>'."\n")
                .'  </sitemap>'."\n";
        }

        return $xml.'</sitemapindex>'."\n";
    }

    /**
     * @param  list<SitemapEntryData>  $entries  locs and image URLs already absolute
     */
    public function urlset(array $entries): string
    {
        $xml = '<?xml version="1.0" encoding="UTF-8"?>'."\n"
            .'<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:image="http://www.google.com/schemas/sitemap-image/1.1">'."\n";

        foreach ($entries as $entry) {
            $xml .= '  <url>'."\n"
                .'    <loc>'.self::escape($entry->loc).'</loc>'."\n";
            if ($entry->lastmod !== null) {
                $xml .= '    <lastmod>'.$entry->lastmod->toAtomString().'</lastmod>'."\n";
            }
            if ($entry->changefreq !== null && in_array($entry->changefreq, self::CHANGEFREQ, true)) {
                $xml .= '    <changefreq>'.$entry->changefreq.'</changefreq>'."\n";
            }
            if ($entry->priority !== null) {
                $xml .= '    <priority>'.number_format(min(1.0, max(0.0, $entry->priority)), 1, '.', '').'</priority>'."\n";
            }
            if ($entry->imageUrl !== null) {
                // image:title / image:caption are deprecated by Google (2022) — only image:loc is emitted.
                $xml .= '    <image:image><image:loc>'.self::escape($entry->imageUrl).'</image:loc></image:image>'."\n";
            }
            $xml .= '  </url>'."\n";
        }

        return $xml.'</urlset>'."\n";
    }

    private static function escape(string $value): string
    {
        return htmlspecialchars($value, ENT_XML1 | ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8');
    }
}
