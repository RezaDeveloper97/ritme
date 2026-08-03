<?php

namespace App\Providers;

use App\Services\HealthEngine\RecommendationRepository;
use App\Services\MessageSystem\Support\MessageContentRepository;
use Illuminate\Support\ServiceProvider;
use Laravel\Passport\Passport;

class AppServiceProvider extends ServiceProvider
{
    /**
     * Register any application services.
     */
    public function register(): void
    {
        // Singletons so their per-request lookup caches — and, for the
        // recommendations, the single query that loads the whole set — are
        // shared across every engine a request builds.
        $this->app->singleton(MessageContentRepository::class);
        $this->app->singleton(RecommendationRepository::class);
    }

    /**
     * Bootstrap any application services.
     */
    public function boot(): void
    {
        // Set token expiration to 1 year
        Passport::tokensExpireIn(now()->addYear());
        Passport::refreshTokensExpireIn(now()->addYear());
        Passport::personalAccessTokensExpireIn(now()->addYear());
    }
}
