<?php

declare(strict_types=1);

namespace App\Providers;

use App\Http\Middleware\ThrottlePerRoute;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Http\Middleware\TrustProxies;
use Illuminate\Routing\Router;
use Illuminate\Support\ServiceProvider;

final class AppServiceProvider extends ServiceProvider
{
    public function register(): void
    {
        //
    }

    public function boot(): void
    {
        // Views receive DTOs; any lazy load or silently dropped attribute is a bug outside production.
        $strict = ! $this->app->isProduction();
        Model::preventLazyLoading($strict);
        Model::preventSilentlyDiscardingAttributes($strict);

        // L9-04: X-Forwarded-* only from configured proxies (none by default) — rate limits key on the real peer IP.
        $proxies = config('app.trusted_proxies', []);
        if (is_array($proxies) && $proxies !== []) {
            /** @var list<string> $proxies */
            TrustProxies::at($proxies === ['*'] ? '*' : $proxies);
        }

        // L9-04: `throttle:N,M` counts per route + client instead of one counter per IP shared by every route.
        $this->app->make(Router::class)->aliasMiddleware('throttle', ThrottlePerRoute::class);
    }
}
