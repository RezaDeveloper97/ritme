<?php

declare(strict_types=1);

use App\Http\Controllers\AboutController;
use App\Http\Controllers\Blog\BlogListingController;
use App\Http\Controllers\Blog\NewsletterController;
use App\Http\Controllers\Blog\PostViewController;
use App\Http\Controllers\Blog\ShowPostController;
use App\Http\Controllers\ContactController;
use App\Http\Controllers\Directory\ListPlacesController;
use App\Http\Controllers\FaqController;
use App\Http\Controllers\FeedController;
use App\Http\Controllers\HomeController;
use App\Http\Controllers\PlaceholderPageController;
use App\Http\Controllers\PlusController;
use App\Http\Controllers\PrivacyController;
use App\Http\Controllers\Pwa\ManifestController;
use App\Http\Controllers\Pwa\OfflineController;
use App\Http\Controllers\Pwa\VersionController;
use App\Http\Controllers\SearchController;
use App\Http\Controllers\Seo\RobotsTxtController;
use App\Http\Controllers\Seo\SitemapController;
use App\Http\Controllers\Seo\SitemapIndexController;
use App\Http\Controllers\ServicesController;
use App\Http\Controllers\SocialResponsibilityController;
use App\Http\Controllers\StagePageController;
use App\Http\Controllers\TermsController;
use App\Http\Controllers\ToolsController;
use App\Http\Middleware\PageCache;
use Illuminate\Cookie\Middleware\AddQueuedCookiesToResponse;
use Illuminate\Cookie\Middleware\EncryptCookies;
use Illuminate\Foundation\Http\Middleware\ValidateCsrfToken;
use Illuminate\Session\Middleware\StartSession;
use Illuminate\Support\Facades\Route;
use Illuminate\View\Middleware\ShareErrorsFromSession;

/*
 * Public pages — docs/AUDIT.md §7 (route names are the StaticPage registry values). No closures: route:cache must
 * work. Every page starts on PlaceholderPageController; its page task replaces that line with the real controller.
 * URL hygiene (trailing slash, duplicate slashes, case, *.html, WordPress slugs, https/host) is done by the global
 * App\Http\Middleware\CanonicalizeUrl before routing.
 */

Route::get('/', HomeController::class)->name('home');                                            // L3-02

Route::name('stage.')->group(function (): void {
    Route::get('/cycle', StagePageController::class)->name('cycle');                         // L3-03
    Route::get('/ttc', StagePageController::class)->name('ttc');                             // L3-04
    Route::get('/pregnancy', StagePageController::class)->name('pregnancy');                 // L3-04
    Route::get('/postpartum', StagePageController::class)->name('postpartum');               // L3-05
    Route::get('/menopause', StagePageController::class)->name('menopause');                 // L3-05
    Route::get('/teen', StagePageController::class)->name('teen');                           // L3-05
});

Route::get('/services', ServicesController::class)->name('services');                               // L3-06
Route::get('/plus', PlusController::class)->name('plus');                                       // L3-06
Route::get('/tools', ToolsController::class)->name('tools');                                     // L3-07
Route::get('/about', AboutController::class)->name('about');                                     // L3-08
Route::get('/social-responsibility', SocialResponsibilityController::class)->name('social-responsibility'); // L3-08
Route::get('/privacy', PrivacyController::class)->name('privacy');                               // L3-08
Route::get('/terms', TermsController::class)->name('terms');                                     // L3-08
Route::get('/faq', FaqController::class)->name('faq');                                           // L3-09
Route::get('/contact', [ContactController::class, 'show'])->name('contact');                       // L3-10
// Contact form (L3-10): PRG, rate limited per IP (no captcha); honeypot + time trap answered like a real send.
Route::post('/contact', [ContactController::class, 'store'])->middleware('throttle:contact')->name('contact.store');

// Site search (L4-04): noindex + never page-cached (route name `search`, SeoManager::NOINDEX_ROUTES), rate limited per IP.
Route::get('/search', SearchController::class)->middleware('throttle:'.SearchController::PER_MINUTE.',1')->name('search');

Route::prefix('blog')->name('blog.')->group(function (): void {
    Route::get('/', [BlogListingController::class, 'index'])->name('index');                       // L4-02
    Route::get('/category/{slug}', [BlogListingController::class, 'category'])->name('category');  // L4-02
    Route::get('/tag/{slug}', [BlogListingController::class, 'tag'])->name('tag');                 // L4-02
    Route::get('/author/{slug}', [BlogListingController::class, 'author'])->name('author');        // L4-02
    // RSS 2.0 (L4-04): before `/{slug}`; cached XML, no session or cookies (like the crawler files below).
    Route::get('/feed', FeedController::class)
        ->withoutMiddleware([EncryptCookies::class, AddQueuedCookiesToResponse::class, StartSession::class, ShareErrorsFromSession::class, ValidateCsrfToken::class])
        ->name('feed');
    Route::get('/{slug}', ShowPostController::class)->name('show');                                // L4-03
    // View beacon (L4-03b): counts page-cache hits too; no session, cookies or CSRF, rate limited per IP.
    Route::post('/{slug}/view', PostViewController::class)
        ->withoutMiddleware([EncryptCookies::class, AddQueuedCookiesToResponse::class, StartSession::class, ShareErrorsFromSession::class, ValidateCsrfToken::class])
        ->middleware('throttle:'.PostViewController::PER_MINUTE.',1')
        ->name('view');
});

// Newsletter (L4-02): double opt-in. POST is rate limited (no captcha); token pages are never page-cached; the
// unsubscribe POST is also the RFC 8058 one-click target, authorised by the token instead of CSRF.
Route::prefix('newsletter')->name('newsletter.')->group(function (): void {
    Route::post('/', [NewsletterController::class, 'store'])->middleware('throttle:newsletter')->name('store');
    Route::withoutMiddleware([PageCache::class])->group(function (): void {
        Route::get('/confirm/{token}', [NewsletterController::class, 'confirm'])->where('token', '[A-Za-z0-9]{1,64}')->name('confirm');
        Route::get('/unsubscribe/{token}', [NewsletterController::class, 'unsubscribeForm'])->where('token', '[A-Za-z0-9]{1,64}')->name('unsubscribe');
        Route::post('/unsubscribe/{token}', [NewsletterController::class, 'unsubscribe'])->where('token', '[A-Za-z0-9]{1,64}')
            ->withoutMiddleware([ValidateCsrfToken::class])->middleware('throttle:newsletter')->name('unsubscribe.store');
    });
});

Route::prefix('directory')->name('directory.')->group(function (): void {
    Route::get('/', ListPlacesController::class)->name('index');                                   // L5-02
    Route::get('/business', PlaceholderPageController::class)->name('business');                   // L5-05
    Route::get('/join', PlaceholderPageController::class)->name('join');                           // L5-05
    Route::get('/join/done', PlaceholderPageController::class)->name('join.done');                 // L5-05
    Route::get('/place/{slug}', PlaceholderPageController::class)->name('place');                  // L5-03
    Route::get('/booked/{code}', PlaceholderPageController::class)->where('code', '[A-Za-z0-9-]+')->name('booked'); // L5-04
    // City / city × category landings (L5-02): after the fixed paths above, so `business`, `join` … never match a city.
    Route::get('/{city}', ListPlacesController::class)->name('city');                              // L5-02
    Route::get('/{city}/{category}', ListPlacesController::class)->name('category');               // L5-02
});

Route::prefix('shop')->name('shop.')->group(function (): void {
    Route::get('/', PlaceholderPageController::class)->name('index');                              // L6-02
    Route::get('/category/{slug}', PlaceholderPageController::class)->name('category');            // L6-02
    Route::get('/product/{slug}', PlaceholderPageController::class)->name('product');              // L6-03
    Route::get('/cart', PlaceholderPageController::class)->name('cart');                           // L6-04
    Route::get('/checkout', PlaceholderPageController::class)->name('checkout');                   // L6-05
    Route::get('/order/{code}', PlaceholderPageController::class)->where('code', '[A-Za-z0-9-]+')->name('order'); // L6-05
});

// Crawler files (L1-06): cached documents, no session or cookies so they stay cacheable by proxies and the server.
Route::withoutMiddleware([EncryptCookies::class, AddQueuedCookiesToResponse::class, StartSession::class, ShareErrorsFromSession::class, ValidateCsrfToken::class])
    ->group(function (): void {
        Route::get('/robots.txt', RobotsTxtController::class)->name('robots');
        Route::get('/sitemap.xml', SitemapIndexController::class)->name('sitemap.index');
        Route::get('/sitemaps/{file}.xml', SitemapController::class)->where('file', '[a-z][a-z0-9-]*')->name('sitemap.file');
        Route::permanentRedirect('/sitemap_index.xml', '/sitemap.xml'); // old WordPress (Yoast) index
        Route::permanentRedirect('/wp-sitemap.xml', '/sitemap.xml');    // old WordPress core index
    });

// Web app manifest (L8-01): cached JSON, no session or cookies (like the crawler files above).
Route::get('/manifest.webmanifest', ManifestController::class)
    ->withoutMiddleware([EncryptCookies::class, AddQueuedCookiesToResponse::class, StartSession::class, ShareErrorsFromSession::class, ValidateCsrfToken::class])
    ->name('pwa.manifest');

// Service worker (L8-02): public/sw.js is a generated static file (tools/build-sw.mjs). Two-tier update source of
// truth — never cached, no session or cookies; /offline is the precached fallback page (noindex).
Route::get('/pwa/version.json', VersionController::class)
    ->withoutMiddleware([EncryptCookies::class, AddQueuedCookiesToResponse::class, StartSession::class, ShareErrorsFromSession::class, ValidateCsrfToken::class])
    ->name('pwa.version');
Route::get('/offline', OfflineController::class)->name('pwa.offline');

// L1-02 shell preview for fidelity screenshots (tools/shot.mjs); never registered in production.
if (! app()->isProduction()) {
    Route::view('/_preview/layout/{variant}', 'layouts.preview')->whereIn('variant', ['dark', 'light'])->name('preview.layout');
    Route::view('/_components', 'previews.components')->name('preview.components'); // L3-01 component kit
}
