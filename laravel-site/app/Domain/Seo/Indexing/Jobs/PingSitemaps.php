<?php

declare(strict_types=1);

namespace App\Domain\Seo\Indexing\Jobs;

use App\Domain\Seo\Indexing\IndexingRules;
use App\Domain\Seo\Sitemap\SitemapUrl;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;
use Throwable;

/**
 * After "regenerate sitemap": GETs every admin-listed ping endpoint (`{sitemap}` = the URL-encoded sitemap index URL)
 * from the queue, production only. Google and Bing retired their sitemap ping endpoints (2023 / 2022), so the list is
 * empty by default; IndexNow is the supported push channel. Failures are logged, never retried.
 */
final class PingSitemaps implements ShouldQueue
{
    use Queueable;

    public int $tries = 1;

    public int $timeout = 60;

    public function handle(IndexingRules $indexing, SitemapUrl $urls, Config $config): void
    {
        if ($config->get('app.env') !== 'production') {
            return;
        }

        $sitemap = rawurlencode($urls->file('/sitemap.xml'));
        foreach ($indexing->settings()->sitemapPingUrls as $endpoint) {
            if (! str_starts_with($endpoint, 'https://')) {
                continue;
            }
            try {
                $status = Http::timeout(10)->get(str_replace('{sitemap}', $sitemap, $endpoint))->status();
                if ($status >= 400) {
                    Log::warning('Sitemap ping failed.', ['endpoint' => $endpoint, 'status' => $status]);
                }
            } catch (Throwable $e) {
                Log::warning('Sitemap ping failed.', ['endpoint' => $endpoint, 'error' => $e->getMessage()]);
            }
        }
    }
}
