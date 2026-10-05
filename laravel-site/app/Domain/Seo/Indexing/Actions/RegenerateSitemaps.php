<?php

declare(strict_types=1);

namespace App\Domain\Seo\Indexing\Actions;

use App\Domain\Seo\Indexing\IndexingRules;
use App\Domain\Seo\Indexing\Jobs\PingSitemaps;
use App\Domain\Seo\Sitemap\Sitemaps;
use App\Support\Cache\NamespaceVersions;
use Illuminate\Contracts\Bus\Dispatcher;
use Illuminate\Contracts\Config\Repository as Config;

/**
 * "Regenerate sitemap": drops every cached sitemap / robots document (`sitemap` namespace + the page cache), rebuilds
 * the index right away and — in production, when ping endpoints are listed — queues PingSitemaps. Returns the number
 * of sitemap files listed in the fresh index.
 */
final class RegenerateSitemaps
{
    public function __construct(
        private readonly NamespaceVersions $versions,
        private readonly Sitemaps $sitemaps,
        private readonly IndexingRules $indexing,
        private readonly Dispatcher $bus,
        private readonly Config $config,
    ) {}

    public function handle(): int
    {
        $this->versions->bump('sitemap', 'pages');
        $files = substr_count($this->sitemaps->index(), '<sitemap>');

        if ($this->config->get('app.env') === 'production' && $this->indexing->settings()->sitemapPingUrls !== []) {
            $this->bus->dispatch(new PingSitemaps);
        }

        return $files;
    }
}
