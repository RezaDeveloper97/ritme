<?php

declare(strict_types=1);

use App\Http\Controllers\PlaceholderPageController;
use App\Http\Controllers\Seo\RobotsTxtController;
use App\Http\Controllers\Seo\SitemapController;
use App\Http\Controllers\Seo\SitemapIndexController;
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

Route::get('/', PlaceholderPageController::class)->name('home');                                   // L3-02

Route::name('stage.')->group(function (): void {
    Route::get('/cycle', PlaceholderPageController::class)->name('cycle');                         // L3-03
    Route::get('/ttc', PlaceholderPageController::class)->name('ttc');                             // L3-04
    Route::get('/pregnancy', PlaceholderPageController::class)->name('pregnancy');                 // L3-04
    Route::get('/postpartum', PlaceholderPageController::class)->name('postpartum');               // L3-05
    Route::get('/menopause', PlaceholderPageController::class)->name('menopause');                 // L3-05
    Route::get('/teen', PlaceholderPageController::class)->name('teen');                           // L3-05
});

Route::get('/services', PlaceholderPageController::class)->name('services');                       // L3-06
Route::get('/plus', PlaceholderPageController::class)->name('plus');                               // L3-06
Route::get('/tools', PlaceholderPageController::class)->name('tools');                             // L3-07
Route::get('/about', PlaceholderPageController::class)->name('about');                             // L3-08
Route::get('/social-responsibility', PlaceholderPageController::class)->name('social-responsibility'); // L3-08
Route::get('/privacy', PlaceholderPageController::class)->name('privacy');                         // L3-08
Route::get('/terms', PlaceholderPageController::class)->name('terms');                             // L3-08
Route::get('/faq', PlaceholderPageController::class)->name('faq');                                 // L3-09
Route::get('/contact', PlaceholderPageController::class)->name('contact');                         // L3-10

Route::prefix('blog')->name('blog.')->group(function (): void {
    Route::get('/', PlaceholderPageController::class)->name('index');                              // L4-02
    Route::get('/{slug}', PlaceholderPageController::class)->name('show');                         // L4-03
});

Route::prefix('directory')->name('directory.')->group(function (): void {
    Route::get('/', PlaceholderPageController::class)->name('index');                              // L5-02
    Route::get('/business', PlaceholderPageController::class)->name('business');                   // L5-05
    Route::get('/join', PlaceholderPageController::class)->name('join');                           // L5-05
    Route::get('/join/done', PlaceholderPageController::class)->name('join.done');                 // L5-05
    Route::get('/place/{slug}', PlaceholderPageController::class)->name('place');                  // L5-03
    Route::get('/booked/{code}', PlaceholderPageController::class)->where('code', '[A-Za-z0-9-]+')->name('booked'); // L5-04
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

// L1-02 shell preview for fidelity screenshots (tools/shot.mjs); never registered in production.
if (! app()->isProduction()) {
    Route::view('/_preview/layout/{variant}', 'layouts.preview')->whereIn('variant', ['dark', 'light'])->name('preview.layout');
    Route::view('/_components', 'previews.components')->name('preview.components'); // L3-01 component kit
}
