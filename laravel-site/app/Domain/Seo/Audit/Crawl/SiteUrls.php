<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Crawl;

use Illuminate\Contracts\Config\Repository as Config;

/**
 * Turns hrefs / sitemap locs into site paths the crawler can fetch: internal = relative, or absolute on the app.url
 * host (any scheme or port). Paths are kept percent-decoded (Persian slugs read naturally in reports and compare
 * equal however a link was encoded) and re-encoded per segment for fetching. The app.url base path (sub-folder
 * installs) is stripped.
 */
final class SiteUrls
{
    private const NON_HTTP = ['mailto:', 'tel:', 'sms:', 'geo:', 'javascript:', 'data:', 'intent:', 'blob:', 'about:', 'whatsapp:', 'tg:'];

    public function __construct(private readonly Config $config) {}

    public function host(): string
    {
        return strtolower((string) (parse_url($this->baseUrl(), PHP_URL_HOST) ?? 'localhost'));
    }

    /**
     * @return array{path: string, query: string, fragment: string}|null null for external and non-HTTP URLs
     */
    public function resolve(string $href, string $currentPath = '/'): ?array
    {
        $href = trim($href);
        if ($href === '') {
            return null;
        }
        $lower = strtolower($href);
        foreach (self::NON_HTTP as $scheme) {
            if (str_starts_with($lower, $scheme)) {
                return null;
            }
        }

        // parse_url() is not multibyte-safe (it mangles Persian slugs / fragments): encode non-ASCII bytes first.
        $ascii = (string) preg_replace_callback('/[^\x21-\x7e]/', static fn (array $m): string => rawurlencode($m[0]), $href);
        $parts = parse_url(str_starts_with($ascii, '//') ? 'https:'.$ascii : $ascii);
        if ($parts === false) {
            return null;
        }
        if (isset($parts['scheme']) && ! in_array(strtolower($parts['scheme']), ['http', 'https'], true)) {
            return null;
        }
        if (isset($parts['host']) && strtolower($parts['host']) !== $this->host()) {
            return null;
        }

        $path = (string) ($parts['path'] ?? '');
        if ($path === '') {
            $path = isset($parts['host']) ? '/' : $currentPath; // "#x" / "?page=2" stay on the current page
        } elseif (! str_starts_with($path, '/')) {
            $directory = rtrim(str_contains($currentPath, '/') ? substr($currentPath, 0, (int) strrpos($currentPath, '/')) : '', '/');
            $path = $directory.'/'.$path;
        }
        if (isset($parts['host'])) {
            $base = $this->basePath();
            if ($base !== '' && str_starts_with($path, $base)) {
                $path = substr($path, strlen($base)) ?: '/';
            }
        }

        return [
            'path' => self::decode($path),
            'query' => (string) ($parts['query'] ?? ''),
            'fragment' => rawurldecode((string) ($parts['fragment'] ?? '')),
        ];
    }

    /** The decoded path (always starting with "/") + query in the form the kernel is asked for. */
    public static function encode(string $path, string $query = ''): string
    {
        $encoded = implode('/', array_map(static fn (string $segment): string => rawurlencode($segment), explode('/', $path)));

        return $encoded.($query === '' ? '' : '?'.$query);
    }

    public static function decode(string $path): string
    {
        $path = rawurldecode($path);

        return str_starts_with($path, '/') ? $path : '/'.$path;
    }

    private function baseUrl(): string
    {
        return (string) $this->config->get('app.url');
    }

    private function basePath(): string
    {
        return rtrim((string) parse_url($this->baseUrl(), PHP_URL_PATH), '/');
    }
}
