<?php

declare(strict_types=1);

use App\Http\Middleware\ApplyRedirects;
use App\Http\Middleware\CanonicalizeUrl;
use App\Http\Middleware\HttpCacheHeaders;
use App\Http\Middleware\MinifyHtml;
use App\Http\Middleware\PageCache;
use App\Http\Middleware\SecurityHeaders;
use Illuminate\Foundation\Application;
use Illuminate\Foundation\Configuration\Exceptions;
use Illuminate\Foundation\Configuration\Middleware;
use Illuminate\Routing\Middleware\SubstituteBindings;

return Application::configure(basePath: dirname(__DIR__))
    ->withRouting(
        web: __DIR__.'/../routes/web.php',
        commands: __DIR__.'/../routes/console.php',
        health: '/up',
    )
    ->withMiddleware(function (Middleware $middleware): void {
        // Outermost: CSP nonce before anything renders, security headers on every response (redirects, errors too).
        $middleware->prepend(SecurityHeaders::class);
        // Admin redirects (L7-03): wraps CanonicalizeUrl — wins over legacy 301s, otherwise only consulted on a 404.
        $middleware->append(ApplyRedirects::class);
        // Global, after TrustProxies and maintenance mode, before routing: 301s to the one canonical URL + 410s.
        $middleware->append(CanonicalizeUrl::class);
        // Cache-Control + weak ETag/304 for page-cached HTML; global so it sees the session cookie of the web group.
        $middleware->append(HttpCacheHeaders::class);

        // Guest full-page cache right after the session starts (before route-model binding queries), then the
        // minifier inside it so pages are cached minified. See config/pagecache.php.
        // The header cart badge (L6-02/L6-04) reads this plain integer cookie from JS.
        $middleware->encryptCookies(except: ['ritme_cart_count']);

        $middleware->web(append: [PageCache::class, MinifyHtml::class]);
        $middleware->prependToPriorityList(SubstituteBindings::class, PageCache::class);
        $middleware->appendToPriorityList(PageCache::class, MinifyHtml::class);
    })
    ->withExceptions(function (Exceptions $exceptions): void {
        //
    })->create();
