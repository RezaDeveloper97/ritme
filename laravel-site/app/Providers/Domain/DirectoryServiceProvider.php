<?php

declare(strict_types=1);

namespace App\Providers\Domain;

use App\Domain\Directory\Contracts\PlaceRepository;
use App\Domain\Directory\Contracts\TaxonomyRepository;
use App\Domain\Directory\Models\Amenity;
use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\District;
use App\Domain\Directory\Models\Landing;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Directory\Models\PlaceReview;
use App\Domain\Directory\Models\PlaceService;
use App\Domain\Directory\Observers\LandingObserver;
use App\Domain\Directory\Observers\PlaceObserver;
use App\Domain\Directory\Observers\PlaceReviewObserver;
use App\Domain\Directory\Observers\PlaceServiceObserver;
use App\Domain\Directory\Observers\TaxonomyObserver;
use App\Domain\Directory\Repositories\CachedPlaceRepository;
use App\Domain\Directory\Repositories\CachedTaxonomyRepository;
use App\Domain\Directory\Repositories\EloquentPlaceRepository;
use App\Domain\Directory\Repositories\EloquentTaxonomyRepository;
use App\Domain\Directory\Search\PlaceSearchProvider;
use App\Domain\Directory\Sitemap\LandingSitemapProvider;
use App\Domain\Directory\Sitemap\PlaceSitemapProvider;
use App\Domain\Media\Actions\FindMediaUsages;
use App\Domain\Search\Support\SearchRegistry;
use App\Domain\Seo\Sitemap\SitemapRegistry;
use App\Providers\DomainServiceProvider;
use Illuminate\Cache\RateLimiting\Limit;
use Illuminate\Database\Eloquent\Relations\Relation;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\RateLimiter;

final class DirectoryServiceProvider extends DomainServiceProvider
{
    /** Join form posts per IP (route middleware `throttle:directory-join`, L5-05); no captcha or external service. */
    public const JOIN_PER_10_MINUTES = 3;

    public const JOIN_PER_DAY = 10;

    protected array $repositories = [
        PlaceRepository::class => [EloquentPlaceRepository::class, CachedPlaceRepository::class],
        TaxonomyRepository::class => [EloquentTaxonomyRepository::class, CachedTaxonomyRepository::class],
    ];

    protected array $observers = [
        Place::class => PlaceObserver::class,
        PlaceService::class => PlaceServiceObserver::class,
        PlaceReview::class => PlaceReviewObserver::class,
        City::class => TaxonomyObserver::class,
        District::class => TaxonomyObserver::class,
        PlaceCategory::class => TaxonomyObserver::class,
        Amenity::class => TaxonomyObserver::class,
        Landing::class => LandingObserver::class,
    ];

    public function register(): void
    {
        parent::register();

        // `/sitemaps/directory-places.xml`, `/sitemaps/directory-landings.xml` (L1-06 registry).
        $this->app->tag([PlaceSitemapProvider::class, LandingSitemapProvider::class], SitemapRegistry::TAG);

        // Places in `/search` (L4-04 registry).
        $this->app->tag([PlaceSearchProvider::class], SearchRegistry::TAG);
    }

    public function boot(): void
    {
        parent::boot();

        // Stable morph name for seo_meta.seoable_type (class names may move).
        Relation::morphMap(['directory_place' => Place::class]);

        // Media in use must never be offered for bulk deletion (admin media library, L2-03).
        FindMediaUsages::column('directory_places', 'cover_media_id', 'تصویر کاور مجموعه', 'name');
        FindMediaUsages::column('directory_place_media', 'media_id', 'گالری مجموعه', 'place_id');
        FindMediaUsages::column('directory_join_request_media', 'media_id', 'عکس درخواست ثبت مجموعه', 'join_request_id');

        RateLimiter::for('directory-join', static function (Request $request): array {
            $ip = (string) $request->ip();

            return [
                Limit::perMinutes(10, self::JOIN_PER_10_MINUTES)->by('directory-join:m:'.$ip),
                Limit::perDay(self::JOIN_PER_DAY)->by('directory-join:d:'.$ip),
            ];
        });
    }
}
