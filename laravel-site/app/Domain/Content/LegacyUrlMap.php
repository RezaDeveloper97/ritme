<?php

declare(strict_types=1);

namespace App\Domain\Content;

/**
 * Old URLs of the static export and the WordPress install → route names of the new site (docs/AUDIT.md §7).
 *
 * Keys are the export's file names without `.html`; the same key serves `/<key>.html`, `/ritme-static/<key>.html`
 * and the WordPress slug `/<key>/`. Detail pages (article, place, product, booked, done) point to their list page,
 * never to demo content. DB-managed redirects come later (L7-03) and sit in front of this map.
 */
final class LegacyUrlMap
{
    /** Old file name / WordPress slug => new route name. */
    public const ROUTES = [
        'index' => 'home',
        'home' => 'home',
        'cycle' => 'stage.cycle',
        'ttc' => 'stage.ttc',
        'pregnancy' => 'stage.pregnancy',
        'postpartum' => 'stage.postpartum',
        'menopause' => 'stage.menopause',
        'teen' => 'stage.teen',
        'services' => 'services',
        'plus' => 'plus',
        'tools' => 'tools',
        'about' => 'about',
        'social-responsibility' => 'social-responsibility',
        'privacy' => 'privacy',
        'faq' => 'faq',
        'contact' => 'contact',
        'blog' => 'blog.index',
        'article' => 'blog.index',
        'directory' => 'directory.index',
        'directory-place' => 'directory.index',
        'directory-booked' => 'directory.index',
        'directory-business' => 'directory.business',
        'directory-join' => 'directory.join',
        'directory-join-done' => 'directory.business',
        'shop' => 'shop.index',
        'shop-list' => 'shop.index',
        'shop-product' => 'shop.index',
        'shop-cart' => 'shop.cart',
        'shop-checkout' => 'shop.cart',
        'shop-done' => 'shop.index',
    ];

    /** Prefix of the WordPress "method A" static copy. */
    public const STATIC_PREFIX = '/ritme-static';

    /** Path prefixes of old asset folders that are gone for good (410). */
    public const GONE_PREFIXES = ['/ritme-static/assets/', '/wp-content/uploads/ritme/'];

    /**
     * Route name for an old path (`/cycle.html`, `/ritme-static/cycle.html`, `/home`), case-insensitive; null when
     * the path is not a legacy URL. The path must already be slash-normalised (no trailing or duplicate slashes).
     */
    public static function routeFor(string $path): ?string
    {
        $path = strtolower($path);
        if (str_starts_with($path, self::STATIC_PREFIX.'/')) {
            $path = substr($path, strlen(self::STATIC_PREFIX));
            if (! str_ends_with($path, '.html')) {
                return null;
            }
        }

        $key = ltrim($path, '/');
        if (str_ends_with($key, '.html')) {
            $key = substr($key, 0, -5);
        }

        return str_contains($key, '/') ? null : (self::ROUTES[$key] ?? null);
    }

    public static function isGone(string $path): bool
    {
        $path = strtolower($path);
        foreach (self::GONE_PREFIXES as $prefix) {
            if (str_starts_with($path.'/', $prefix) || str_starts_with($path, $prefix)) {
                return true;
            }
        }

        return false;
    }
}
