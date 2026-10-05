<?php

declare(strict_types=1);

namespace App\Support\Http;

/**
 * Guard for URLs the server itself requests (sitemap pings, L9-04): only `https://` to a public host name — never an
 * IP literal, `localhost`, a single-label or `.local`/`.internal`/`.localhost`/`.test` name, credentials in the URL
 * or a non-default port. That keeps an admin-entered endpoint from probing the hosting box or the internal network
 * (SSRF). Name resolution is not checked (no DNS in tests); callers also disable redirects.
 */
final class OutboundUrl
{
    private const BLOCKED_SUFFIXES = ['.local', '.localhost', '.internal', '.lan', '.home', '.test', '.invalid', '.arpa'];

    public static function isSafe(string $url): bool
    {
        if (! str_starts_with(strtolower($url), 'https://') || filter_var($url, FILTER_VALIDATE_URL) === false) {
            return false;
        }

        $parts = parse_url($url);
        if (! is_array($parts) || ! isset($parts['host']) || isset($parts['user']) || isset($parts['pass'])) {
            return false;
        }
        if (isset($parts['port']) && $parts['port'] !== 443) {
            return false;
        }

        $host = strtolower(rtrim($parts['host'], '.'));
        $bare = trim($host, '[]');
        if (filter_var($bare, FILTER_VALIDATE_IP) !== false) {
            return false; // IP literals (incl. 127.0.0.1, 169.254.169.254, ::1) are never a search-engine endpoint
        }
        if ($host === 'localhost' || ! str_contains($host, '.') || preg_match('/^[a-z0-9.-]+$/', $host) !== 1) {
            return false;
        }
        foreach (self::BLOCKED_SUFFIXES as $suffix) {
            if (str_ends_with($host, $suffix)) {
                return false;
            }
        }

        return true;
    }
}
