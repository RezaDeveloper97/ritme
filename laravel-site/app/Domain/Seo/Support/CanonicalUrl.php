<?php

declare(strict_types=1);

namespace App\Domain\Seo\Support;

/**
 * Normalises a URL (absolute or relative) into a canonical URL on the site's own origin:
 * https, the configured host (+ port), collapsed slashes, no trailing slash, no fragment, and only allow-listed
 * query parameters — tracking and filter parameters are dropped; `page` survives only when it is an integer > 1.
 */
final class CanonicalUrl
{
    /**
     * @param  list<string>  $keep  query parameters kept besides `page`, in addition to the allow-list
     */
    public static function normalize(string $url, string $baseUrl, array $keep = []): string
    {
        $base = parse_url($baseUrl) ?: [];
        $parts = parse_url(trim($url)) ?: [];

        $host = strtolower((string) ($base['host'] ?? $parts['host'] ?? 'localhost'));
        $port = isset($base['port']) ? ':'.$base['port'] : '';

        $path = preg_replace('#/{2,}#', '/', '/'.ltrim((string) ($parts['path'] ?? '/'), '/')) ?? '/';
        $path = $path === '/' ? '' : rtrim($path, '/');

        parse_str((string) ($parts['query'] ?? ''), $query);
        $kept = [];
        foreach ($query as $key => $value) {
            $key = (string) $key;
            if ($key === 'page') {
                if (is_string($value) && ctype_digit($value) && (int) $value > 1) {
                    $kept['page'] = (string) (int) $value;
                }

                continue;
            }
            if (in_array($key, $keep, true)) {
                $kept[$key] = $value;
            }
        }
        ksort($kept);

        $queryString = $kept === [] ? '' : '?'.http_build_query($kept, '', '&', PHP_QUERY_RFC3986);

        return "https://{$host}{$port}{$path}{$queryString}";
    }
}
