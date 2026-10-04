<?php

declare(strict_types=1);

namespace App\Domain\Seo\Sitemap;

use App\Domain\Seo\Support\CanonicalUrl;
use Illuminate\Contracts\Config\Repository as Config;

/**
 * Absolute https URLs on the configured origin (app.url), whatever host the request that built the document came in
 * on: page locs are canonicalised exactly like the `<link rel=canonical>` (CanonicalUrl), file URLs (images, sitemap
 * files) keep their path and query. Every URL is moved onto that origin — the site serves no third-party files.
 */
final class SitemapUrl
{
    public function __construct(private readonly Config $config) {}

    public function page(string $url): string
    {
        return CanonicalUrl::normalize($url, $this->baseUrl());
    }

    /** Image / sitemap-file URL; a relative path is resolved against app.url (sub-folder base path included). */
    public function file(string $pathOrUrl): string
    {
        $parts = parse_url(trim($pathOrUrl)) ?: [];

        $path = (string) ($parts['path'] ?? '/');
        if (! isset($parts['host'])) {
            $path = $this->basePath().'/'.ltrim($path, '/');
        }
        $query = isset($parts['query']) ? '?'.$parts['query'] : '';

        return $this->origin().'/'.ltrim($path, '/').$query;
    }

    private function baseUrl(): string
    {
        return (string) $this->config->get('app.url');
    }

    private function host(): string
    {
        return strtolower((string) (parse_url($this->baseUrl(), PHP_URL_HOST) ?? 'localhost'));
    }

    private function origin(): string
    {
        $port = parse_url($this->baseUrl(), PHP_URL_PORT);

        return 'https://'.$this->host().(is_int($port) ? ':'.$port : '');
    }

    private function basePath(): string
    {
        return rtrim((string) parse_url($this->baseUrl(), PHP_URL_PATH), '/');
    }
}
