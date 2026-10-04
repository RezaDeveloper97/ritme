<?php

declare(strict_types=1);

namespace App\Providers\Domain;

use App\Domain\Media\Support\MediaOgImageResolver;
use App\Domain\Seo\Contracts\OgImageResolver;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Seo\Observers\SeoMetaObserver;
use App\Domain\Seo\Repositories\CachedSeoMetaRepository;
use App\Domain\Seo\Repositories\EloquentSeoMetaRepository;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\SeoManager;
use App\Domain\Seo\Sitemap\DefaultRobotsRules;
use App\Domain\Seo\Sitemap\PagesSitemapProvider;
use App\Domain\Seo\Sitemap\RobotsRules;
use App\Domain\Seo\Sitemap\SitemapRegistry;
use App\Providers\DomainServiceProvider;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\Foundation\Application;
use Illuminate\Contracts\Routing\UrlGenerator;
use Illuminate\Routing\Router;

final class SeoServiceProvider extends DomainServiceProvider
{
    protected array $repositories = [
        SeoMetaRepository::class => [EloquentSeoMetaRepository::class, CachedSeoMetaRepository::class],
    ];

    protected array $observers = [
        SeoMeta::class => SeoMetaObserver::class,
    ];

    /** @var array<class-string, class-string> */
    public array $singletons = [
        OgImageResolver::class => MediaOgImageResolver::class, // og variant of the media library (L2-01)
        RobotsRules::class => DefaultRobotsRules::class,       // admin-edited rules in L7-04
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
    }
}
