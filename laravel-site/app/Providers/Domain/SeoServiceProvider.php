<?php

declare(strict_types=1);

namespace App\Providers\Domain;

use App\Domain\Blog\Models\Author as BlogAuthor;
use App\Domain\Blog\Models\Category as BlogCategory;
use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Models\Tag as BlogTag;
use App\Domain\Directory\Models\City as DirectoryCity;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceCategory as DirectoryCategory;
use App\Domain\Media\Support\MediaOgImageResolver;
use App\Domain\Seo\Contracts\OgImageResolver;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Indexing\IndexNow\IndexNowKey;
use App\Domain\Seo\Indexing\Observers\IndexingSettingObserver;
use App\Domain\Seo\Indexing\Observers\IndexNowObserver;
use App\Domain\Seo\Indexing\SettingsRobotsRules;
use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Seo\Observers\SeoMetaObserver;
use App\Domain\Seo\Redirects\Console\FlushRedirectStatsCommand;
use App\Domain\Seo\Redirects\Console\PurgeNotFoundLogsCommand;
use App\Domain\Seo\Redirects\Contracts\RedirectMapRepository;
use App\Domain\Seo\Redirects\Models\Redirect;
use App\Domain\Seo\Redirects\Observers\RedirectObserver;
use App\Domain\Seo\Redirects\Observers\SlugRedirectObserver;
use App\Domain\Seo\Redirects\Repositories\CachedRedirectMapRepository;
use App\Domain\Seo\Redirects\Repositories\EloquentRedirectMapRepository;
use App\Domain\Seo\Repositories\CachedSeoMetaRepository;
use App\Domain\Seo\Repositories\EloquentSeoMetaRepository;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\SeoManager;
use App\Domain\Seo\Sitemap\PagesSitemapProvider;
use App\Domain\Seo\Sitemap\RobotsRules;
use App\Domain\Seo\Sitemap\SitemapRegistry;
use App\Domain\Settings\Models\Setting;
use App\Domain\Shop\Catalog\Models\Category as ShopCategory;
use App\Domain\Shop\Catalog\Models\Product;
use App\Http\Controllers\Seo\IndexNowKeyController;
use App\Providers\DomainServiceProvider;
use Illuminate\Console\Scheduling\Schedule;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\Foundation\Application;
use Illuminate\Contracts\Routing\UrlGenerator;
use Illuminate\Routing\Router;
use Illuminate\Support\Facades\Route;

final class SeoServiceProvider extends DomainServiceProvider
{
    protected array $repositories = [
        SeoMetaRepository::class => [EloquentSeoMetaRepository::class, CachedSeoMetaRepository::class],
        RedirectMapRepository::class => [EloquentRedirectMapRepository::class, CachedRedirectMapRepository::class], // L7-03
    ];

    protected array $observers = [
        SeoMeta::class => SeoMetaObserver::class,
        Setting::class => IndexingSettingObserver::class, // seo settings feed robots/sitemaps: bump `sitemap` (L7-04)
        // IndexNow submissions on publish / update / delete (L7-04; no-op unless switched on, production only).
        Post::class => IndexNowObserver::class,
        Product::class => IndexNowObserver::class,
        Place::class => IndexNowObserver::class,
        // Redirect manager (L7-03): the cached map, and auto 301s when a taxonomy slug changes.
        Redirect::class => RedirectObserver::class,
        BlogCategory::class => SlugRedirectObserver::class,
        BlogTag::class => SlugRedirectObserver::class,
        BlogAuthor::class => SlugRedirectObserver::class,
        ShopCategory::class => SlugRedirectObserver::class,
        DirectoryCity::class => SlugRedirectObserver::class,
        DirectoryCategory::class => SlugRedirectObserver::class,
    ];

    /** @var array<class-string, class-string> */
    public array $singletons = [
        OgImageResolver::class => MediaOgImageResolver::class, // og variant of the media library (L2-01)
        RobotsRules::class => SettingsRobotsRules::class,      // admin-edited rules, built-in defaults when empty (L7-04)
        SitemapRegistry::class => SitemapRegistry::class,
    ];

    public function register(): void
    {
        parent::register();

        // One SeoManager per request: controller overrides must never leak into the next request
        // (in-process sub-requests of seo:audit, tests, queue workers).
        // Same for the JSON-LD graph pages add their nodes to.
        $this->app->scoped(SeoManager::class);
        $this->app->scoped(SchemaGraph::class);
        $this->app->rebinding('request', static function (Application $app): void {
            $app->forgetInstance(SeoManager::class);
            $app->forgetInstance(SchemaGraph::class);
        });

        // WebSite SearchAction (L4-04) once the `search` route exists: {site}/search?q={search_term_string}.
        $this->app->afterResolving(SchemaGraph::class, static function (SchemaGraph $graph, Application $app): void {
            if ($app->make(Router::class)->has('search')) {
                $path = $app->make(UrlGenerator::class)->route('search', [], false);
                $graph->searchUrlTemplate(rtrim((string) $app->make(Config::class)->get('app.url'), '/').$path.'?q={search_term_string}');
            }
        });

        // Sitemap files (L1-06). Other contexts tag their own providers the same way in their service provider.
        $this->app->tag([PagesSitemapProvider::class], SitemapRegistry::TAG);
        $this->registerRedirects();
    }

    /** Redirect manager (L7-03): stats flush every five minutes, 404 purge daily (cPanel cron runs schedule:run). */
    private function registerRedirects(): void
    {
        $this->commands([FlushRedirectStatsCommand::class, PurgeNotFoundLogsCommand::class]);
        $this->callAfterResolving(Schedule::class, static function (Schedule $schedule): void {
            $schedule->command('seo:flush-redirect-stats')->everyFiveMinutes()->withoutOverlapping(10);
            $schedule->command('seo:purge-404')->dailyAt('03:40')->withoutOverlapping(30);
        });
    }

    public function boot(): void
    {
        parent::boot();

        // IndexNow key file (L7-04): `/{key}.txt` at the site root, no session / cookies (like robots.txt). Kept here
        // so the Indexing feature stays in one place; part of `route:cache` like any other route.
        if (! $this->app->routesAreCached()) {
            Route::get('/{key}.txt', IndexNowKeyController::class)
                ->where('key', IndexNowKey::PATTERN)
                ->name(IndexNowKeyController::ROUTE);
        }
    }
}
