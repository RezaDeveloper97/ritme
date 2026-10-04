<?php

declare(strict_types=1);

namespace App\Domain\Seo\Sitemap;

use Carbon\CarbonImmutable;

/**
 * One `<url>` of a sitemap: absolute loc, real lastmod (null when unknown — never faked with "now"), the primary image
 * (image:image) and the optional priority / changefreq hints.
 */
final readonly class SitemapEntryData
{
    public function __construct(
        public string $loc,
        public ?CarbonImmutable $lastmod = null,
        public ?string $imageUrl = null,
        public ?string $imageTitle = null,
        public ?float $priority = null,
        public ?string $changefreq = null,
    ) {}
}
