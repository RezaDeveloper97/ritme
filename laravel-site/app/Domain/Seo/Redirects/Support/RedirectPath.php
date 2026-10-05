<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Support;

/**
 * Path rules shared by the redirect map, SaveRedirect and the 404 monitor: paths are stored decoded (Persian stays
 * readable), with one leading slash, no duplicate / trailing slash, no query or fragment, and are compared
 * case-insensitively through key() / hash(). Regex sources are full-path PCRE patterns, anchored and matched
 * case-insensitively in UTF-8 mode.
 */
final class RedirectPath
{
    public const MAX_LENGTH = 700;

    public static function normalize(string $path): string
    {
        $path = trim($path);
        $path = (string) preg_replace('~[?#].*$~s', '', $path);

        $decoded = rawurldecode($path);
        if (mb_check_encoding($decoded, 'UTF-8')) {
            $path = $decoded;
        }

        $path = '/'.ltrim($path, '/');
        $path = (string) preg_replace('~/{2,}~', '/', $path);

        return $path === '/' ? '/' : rtrim($path, '/');
    }

    public static function key(string $path): string
    {
        return mb_strtolower(self::normalize($path), 'UTF-8');
    }

    public static function hash(string $path): string
    {
        return sha1(self::key($path));
    }

    /**
     * The local part of a target: `['path' => …, 'query' => …|null]` for `/path?x` or an absolute URL on the site's
     * own host; null for an external URL.
     *
     * @return array{path: string, query: string|null}|null
     */
    public static function local(string $target, string $appUrl): ?array
    {
        $target = trim($target);
        if ($target === '' || str_starts_with($target, '//')) {
            return null;
        }

        if (! str_starts_with($target, '/')) {
            $parts = parse_url($target);
            $host = is_array($parts) ? strtolower((string) ($parts['host'] ?? '')) : '';
            if ($host === '' || $host !== strtolower((string) parse_url($appUrl, PHP_URL_HOST))) {
                return null;
            }
            $base = rtrim((string) parse_url($appUrl, PHP_URL_PATH), '/');
            $path = (string) ($parts['path'] ?? '/');
            if ($base !== '' && str_starts_with($path, $base)) {
                $path = substr($path, strlen($base));
            }
            $query = isset($parts['query']) && $parts['query'] !== '' ? $parts['query'] : null;

            return ['path' => self::normalize($path), 'query' => $query];
        }

        $query = parse_url('http://x'.$target, PHP_URL_QUERY);

        return ['path' => self::normalize($target), 'query' => is_string($query) && $query !== '' ? $query : null];
    }

    /** Percent-encodes each segment of a decoded path for a Location header. */
    public static function encode(string $path): string
    {
        return implode('/', array_map(rawurlencode(...), explode('/', $path)));
    }

    public static function compile(string $pattern): string
    {
        return '~^(?:'.str_replace('~', '\~', $pattern).')$~iu';
    }

    public static function isValidRegex(string $pattern): bool
    {
        if (trim($pattern) === '') {
            return false;
        }

        set_error_handler(static fn (): bool => true);
        try {
            return preg_match(self::compile($pattern), '') !== false;
        } finally {
            restore_error_handler();
        }
    }
}
