<?php

declare(strict_types=1);

namespace App\Domain\Seo\Sitemap;

use App\Support\Cache\CacheAside;
use App\Support\Cache\CacheKey;
use Illuminate\Contracts\Config\Repository as Config;

/**
 * Dynamic robots.txt. Production: the configured RobotsRules + a `Sitemap:` line on the canonical origin.
 * Any other environment (local, staging, testing): `Disallow: /` and no sitemap, so a non-production copy is never
 * crawled. Cached in the `sitemap` namespace.
 */
final class RobotsTxt
{
    public function __construct(
        private readonly RobotsRules $rules,
        private readonly SitemapUrl $urls,
        private readonly CacheAside $cache,
        private readonly Config $config,
    ) {}

    public function render(): string
    {
        if ($this->config->get('app.env') !== 'production') {
            return "User-agent: *\nDisallow: /\n";
        }

        return $this->cache->remember(CacheKey::make('sitemap', 'robots'), null, fn (): string => rtrim($this->rules->rules())
            ."\n\nSitemap: ".$this->urls->file('/sitemap.xml')."\n");
    }
}
