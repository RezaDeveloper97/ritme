<?php

declare(strict_types=1);

namespace App\Providers;

use Illuminate\Contracts\Foundation\Application;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Facades\View;
use Illuminate\Support\ServiceProvider;

/**
 * Base provider for a bounded context. Subclasses only declare arrays:
 *
 *  - $repositories: Contract => Eloquent impl, or Contract => [Eloquent impl, Cached decorator].
 *    The decorator receives the Eloquent impl through a constructor parameter named `$inner`.
 *  - $observers:    Model => Observer (or list of observers).
 *  - $composers:    view name/pattern => View composer class (or list).
 *
 * Plain service bindings use Laravel's own `$bindings` / `$singletons` properties.
 */
abstract class DomainServiceProvider extends ServiceProvider
{
    /** @var array<class-string, class-string|array{0: class-string, 1: class-string}> */
    protected array $repositories = [];

    /** @var array<class-string, class-string|list<class-string>> */
    protected array $observers = [];

    /** @var array<string, class-string|list<class-string>> */
    protected array $composers = [];

    public function register(): void
    {
        foreach ($this->repositories as $contract => $implementation) {
            [$eloquent, $cached] = is_array($implementation)
                ? [$implementation[0], $implementation[1]]
                : [$implementation, null];

            $this->app->singleton($contract, static function (Application $app) use ($eloquent, $cached): object {
                $inner = $app->make($eloquent);

                return $cached === null ? $inner : $app->make($cached, ['inner' => $inner]);
            });
        }
    }

    public function boot(): void
    {
        foreach ($this->observers as $model => $observers) {
            /** @var class-string<Model> $model */
            $model::observe($observers);
        }

        foreach ($this->composers as $views => $composers) {
            foreach ((array) $composers as $composer) {
                View::composer($views, $composer);
            }
        }
    }
}
