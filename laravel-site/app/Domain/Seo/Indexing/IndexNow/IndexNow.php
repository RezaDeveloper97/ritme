<?php

declare(strict_types=1);

namespace App\Domain\Seo\Indexing\IndexNow;

use App\Domain\Seo\Indexing\IndexingRules;
use App\Domain\Seo\Sitemap\SitemapUrl;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;
use RuntimeException;

/**
 * IndexNow (Bing, Yandex, Seznam, Naver … share submissions through api.indexnow.org). Off by default; active only
 * when the admin switch is on, a key exists and the app runs in production — so staging, local and tests never submit.
 * This is a server-side outbound call made from a queued job (SubmitToIndexNow); public pages still make zero
 * external requests. The key is served at /{key}.txt (keyLocation) while active.
 */
final class IndexNow
{
    public const ENDPOINT = 'https://api.indexnow.org/indexnow';

    public const MAX_URLS = 10_000;

    public function __construct(
        private readonly IndexingRules $indexing,
        private readonly SitemapUrl $urls,
        private readonly Config $config,
    ) {}

    /** Admin switch on + key present (the key file is served even outside production, for checking). */
    public function configured(): bool
    {
        $settings = $this->indexing->settings();

        return $settings->indexNowEnabled && $settings->indexNowKey !== null && IndexNowKey::isValid($settings->indexNowKey);
    }

    /** Configured and in production: submissions really go out. */
    public function active(): bool
    {
        return $this->configured() && $this->config->get('app.env') === 'production';
    }

    public function key(): ?string
    {
        return $this->configured() ? $this->indexing->settings()->indexNowKey : null;
    }

    public function keyLocation(): ?string
    {
        $key = $this->key();

        return $key === null ? null : $this->urls->file('/'.$key.'.txt');
    }

    /**
     * Submits page URLs of this site (www / http variants moved onto the canonical origin; other hosts dropped). Returns the number submitted.
     * 4xx answers are logged and dropped (a retry would fail the same way); 429 / 5xx throw so the job retries.
     *
     * @param  list<string>  $urls
     */
    public function submit(array $urls): int
    {
        $key = $this->key();
        if (! $this->active() || $key === null) {
            return 0;
        }

        $host = (string) parse_url($this->urls->file('/'), PHP_URL_HOST);
        $list = [];
        foreach ($urls as $url) {
            $urlHost = strtolower((string) parse_url($url, PHP_URL_HOST));
            if ($urlHost === '' || $urlHost === strtolower($host) || $urlHost === 'www.'.strtolower($host)) {
                $list[$this->urls->page($url)] = true; // canonical https origin, tracking params dropped
            }
        }
        $list = array_slice(array_keys($list), 0, self::MAX_URLS);
        if ($list === []) {
            return 0;
        }

        $response = Http::timeout(10)->acceptJson()->asJson()->post(self::ENDPOINT, [
            'host' => $host,
            'key' => $key,
            'keyLocation' => $this->keyLocation(),
            'urlList' => $list,
        ]);

        $status = $response->status();
        if ($status === 429 || $status >= 500) {
            throw new RuntimeException("IndexNow answered HTTP {$status}; retrying later.");
        }
        if ($status >= 400) {
            Log::warning('IndexNow rejected the submission.', ['status' => $status, 'urls' => count($list)]);

            return 0;
        }

        return count($list);
    }
}
