<?php

declare(strict_types=1);

/*
|--------------------------------------------------------------------------
| Performance core (L1-07): guest full-page cache, HTTP caching, security headers
|--------------------------------------------------------------------------
|
| App\Http\Middleware\PageCache      guest GET/HEAD 200 HTML → cache-aside `pages` namespace (X-Page-Cache)
| App\Http\Middleware\HttpCacheHeaders Cache-Control + weak ETag / 304 for the pages PageCache handles
| App\Http\Middleware\SecurityHeaders strict CSP (no external origins), HSTS, nosniff, referrer, permissions
|
| The page cache lives in the cache-aside `pages` namespace (TTL in config/cacheaside.php). Every
| CacheBumpingObserver bumps `pages` (cacheaside.always_bump), so content, settings and SEO changes
| invalidate rendered pages; `php artisan cache:ns bump pages` clears them by hand.
|
*/

return [

    'enabled' => (bool) env('PAGE_CACHE_ENABLED', true),

    /*
    | Seconds a rendered page is kept. null = the `pages` namespace TTL (cacheaside.namespaces.pages).
    */
    'ttl' => env('PAGE_CACHE_TTL') === null ? null : (int) env('PAGE_CACHE_TTL'),

    /*
    | Query parameters that make a different page and are part of the cache key. Tracking parameters
    | (App\Domain\Seo\SeoManager::TRACKING_PARAMS: utm_*, gclid, fbclid …) are dropped from the key.
    | A request with any other parameter (search, filters …) is not cached at all.
    */
    'query_whitelist' => ['page'],

    /*
    | Paths (Request::is patterns, no leading slash) never cached. The Filament admin path
    | (filament.admin.path) is always added.
    */
    'except_paths' => [
        'livewire/*',
        'filament/*',
        'storage/*',
        'up',
        '_preview/*',
    ],

    /*
    | Route names (Str::is patterns) never cached, in addition to the transactional noindex routes of
    | App\Domain\Seo\SeoManager::NOINDEX_ROUTES (search, cart, checkout, order, done, booked).
    */
    'except_routes' => [],

    /*
    | A request carrying one of these cookies is never served from / written to the cache (e.g. the
    | shop cart cookie of L6-04, so a filled mini-cart is never cached or hidden). `remember_*` login
    | cookies always bypass.
    */
    'bypass_cookies' => ['ritme_cart'],

    /*
    | App\Http\Middleware\MinifyHtml (web group, inside PageCache, never on admin paths). Measured on the layout
    | pages: raw HTML −16 %, gzip −5 %.
    */
    'minify' => (bool) env('PAGE_CACHE_MINIFY', true),

    /*
    | HTTP caching for pages the page cache handles (HttpCacheHeaders).
    |
    | Browsers always revalidate (max-age=0) with a weak ETag → 304. `public, s-maxage` (for a shared cache /
    | reverse proxy) is sent only when the response sets no cookie; otherwise `private, no-cache`.
    | s_maxage = 0 disables the public variant.
    */
    'http' => [
        'etag' => true,
        's_maxage' => (int) env('HTTP_CACHE_S_MAXAGE', 300),
        'stale_while_revalidate' => (int) env('HTTP_CACHE_SWR', 60),
    ],

    /*
    | SecurityHeaders. The public site needs no external origin at all (CLAUDE.md "No external requests").
    | The Filament admin (and Livewire / Filament endpoints) gets a relaxed policy because Livewire + Alpine
    | need inline scripts, eval and inline styles — still no external origin.
    */
    'security' => [
        'enabled' => (bool) env('SECURITY_HEADERS', true),

        // null = on in production (only ever sent over https).
        'hsts' => env('SECURITY_HSTS') === null ? null : (bool) env('SECURITY_HSTS'),
        'hsts_max_age' => 31536000,
        'hsts_include_subdomains' => (bool) env('SECURITY_HSTS_SUBDOMAINS', false),

        // Per-request nonce in `script-src` (+ on Vite's tags). Off: the site has no inline JS (CLAUDE.md: JS only as
        // data-module ES modules; JSON-LD is a data block CSP does not apply to), so `script-src 'self'` is enough.
        // Turn on only for an inline bootstrap; page-cached HTML gets the current nonce swapped in on every HIT.
        'nonce' => (bool) env('SECURITY_CSP_NONCE', false),

        // Request::is patterns that use the admin policy; filament.admin.path is always added.
        'admin_paths' => ['livewire/*', 'filament/*'],

        'permissions_policy' => 'accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()',
    ],

];
