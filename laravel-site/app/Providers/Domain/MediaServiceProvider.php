<?php

declare(strict_types=1);

namespace App\Providers\Domain;

use App\Domain\Media\Contracts\MediaRepository;
use App\Domain\Media\Models\Media;
use App\Domain\Media\Observers\MediaObserver;
use App\Domain\Media\Repositories\CachedMediaRepository;
use App\Domain\Media\Repositories\EloquentMediaRepository;
use App\Domain\Media\Support\FormatSupport;
use App\Domain\Media\Support\ImageEncoder;
use App\Domain\Media\Support\ImageManagerFactory;
use App\Domain\Media\Support\VariantPlanner;
use App\Providers\DomainServiceProvider;
use Illuminate\Contracts\Foundation\Application;
use Intervention\Image\ImageManager;

final class MediaServiceProvider extends DomainServiceProvider
{
    protected array $repositories = [
        MediaRepository::class => [EloquentMediaRepository::class, CachedMediaRepository::class],
    ];

    protected array $observers = [
        Media::class => MediaObserver::class,
    ];

    /** @var array<class-string, class-string> */
    public array $singletons = [
        FormatSupport::class => FormatSupport::class, // caches driver capability checks
    ];

    public function register(): void
    {
        parent::register();

        $this->app->singleton(ImageManager::class, static fn (Application $app): ImageManager => ImageManagerFactory::make((string) $app['config']->get('media.driver', 'auto')));

        $this->app->bind(ImageEncoder::class, static fn (Application $app): ImageEncoder => new ImageEncoder((array) $app['config']->get('media.quality', [])));

        $this->app->bind(VariantPlanner::class, static fn (Application $app): VariantPlanner => new VariantPlanner(
            presets: (array) $app['config']->get('media.presets', []),
            defaultFormats: array_values((array) $app['config']->get('media.formats', [])),
            support: $app->make(FormatSupport::class),
            posterWidth: (int) $app['config']->get('media.poster_width', 1280),
        ));
    }
}
