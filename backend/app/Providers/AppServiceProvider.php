<?php

namespace App\Providers;

use App\Services\HealthEngine\RecommendationRepository;
use App\Services\Language\LanguageRegistry;
use App\Services\MessageSystem\Support\MessageContentRepository;
use DateInterval;
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

        // The locale list is read by admin forms, API resolvers and content
        // fallbacks many times per request; one instance keeps its memo warm.
        $this->app->singleton(LanguageRegistry::class);
    }

    /**
     * Bootstrap any application services.
     */
    public function boot(): void
    {
        // Sessions last a fixed number of days (365 by default). An interval,
        // not a boot-time date, so a long-lived worker can't drift the value.
        $lifetime = new DateInterval('P'.config('passport.token_lifetime_days', 365).'D');

        Passport::tokensExpireIn($lifetime);
        Passport::refreshTokensExpireIn($lifetime);
        Passport::personalAccessTokensExpireIn($lifetime);
    }
}
