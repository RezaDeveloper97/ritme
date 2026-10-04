<?php

declare(strict_types=1);

namespace App\Providers\Domain;

use App\Domain\Seo\Contracts\OgImageResolver;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Seo\Observers\SeoMetaObserver;
use App\Domain\Seo\Repositories\CachedSeoMetaRepository;
use App\Domain\Seo\Repositories\EloquentSeoMetaRepository;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\SeoManager;
use App\Domain\Seo\Support\NullOgImageResolver;
use App\Providers\DomainServiceProvider;
use Illuminate\Contracts\Foundation\Application;

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
        OgImageResolver::class => NullOgImageResolver::class, // replaced by the media library (L2)
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
    }
}
