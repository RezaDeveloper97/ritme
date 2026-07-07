<?php

namespace App\Providers;

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
        // Singleton so its per-request lookup cache is shared across all engines.
        $this->app->singleton(MessageContentRepository::class);
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
