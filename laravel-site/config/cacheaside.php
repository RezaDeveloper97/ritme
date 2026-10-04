<?php

declare(strict_types=1);

/*
|--------------------------------------------------------------------------
| Cache-aside kernel (App\Support\Cache)
|--------------------------------------------------------------------------
|
| Hot reads go through App\Support\Cache\CacheAside (usually via a Cached*
| repository decorator). Keys look like  rt:{namespace}:v{version}:{key}.
|
| Invalidation is driver-agnostic (works on the cPanel `file` and `database`
| drivers, which have no tags): bumping a namespace increments its version,
| so every key built afterwards misses and the old entries simply expire.
|
|   php artisan cache:ns list           show namespaces, versions and TTLs
|   php artisan cache:ns bump blog      invalidate one namespace
|   php artisan cache:ns bump-all       invalidate every namespace
|
*/

return [

    /*
    | Cache store used for cached values AND namespace versions (one of the
    | stores in config/cache.php). null = the default store (CACHE_STORE).
    */
    'store' => env('CACHE_ASIDE_STORE'),

    /*
    | Prefix of every key the kernel writes. The store's own prefix
    | (cache.prefix) is still applied on top by Laravel.
    */
    'prefix' => 'rt',

    /*
    | TTL jitter as a fraction of the TTL: 0.1 = ±10 %. Spreads expiry of
    | entries written at the same moment so they don't all miss together.
    | 0 disables jitter. rememberForever() is never jittered.
    */
    'jitter' => 0.1,

    /*
    | Stampede protection: on a miss, only one process runs the loader while
    | the others wait (up to `wait` seconds) and then read the fresh value.
    | Used only when the store supports atomic locks (array, file, database,
    | redis do); otherwise, or when waiting times out, the loader runs
    | without a lock — a request never fails because of the lock.
    */
    'lock' => [
        'enabled' => (bool) env('CACHE_ASIDE_LOCK', true),
        'seconds' => 10, // lock lifetime (should exceed the slowest loader)
        'wait' => 3,     // max seconds a waiting process blocks before loading itself
    ],

    /*
    | Namespaces bumped on EVERY model change handled by a CacheBumpingObserver
    | (via NamespaceBumper), in addition to the model's own namespaces.
    | `pages` is the guest full-page cache (L1-07): any content change
    | must invalidate rendered pages.
    */
    'always_bump' => ['pages'],

    /*
    | Declared namespaces => default TTL in seconds (used when a caller passes
    | no TTL). Using an undeclared namespace throws, which catches typos.
    */
    'namespaces' => [
        'settings' => 86400,   // site settings (admin-edited, rarely changes)
        'seo' => 21600,        // page SEO records, redirects, JSON-LD fragments
        'pages' => 3600,       // guest full-page cache
        'blog' => 3600,        // magazine posts, categories, tags
        'faq' => 21600,        // FAQ groups and items
        'directory' => 3600,   // mother & child directory places
        'shop' => 1800,        // catalog, products, prices, stock
        'media' => 86400,      // media library / responsive image sets
        'sitemap' => 21600,    // generated sitemap documents
        'menu' => 86400,       // header / footer navigation
    ],

];
