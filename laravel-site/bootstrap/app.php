<?php

declare(strict_types=1);

use App\Http\Middleware\CanonicalizeUrl;
use Illuminate\Foundation\Application;
use Illuminate\Foundation\Configuration\Exceptions;
use Illuminate\Foundation\Configuration\Middleware;

return Application::configure(basePath: dirname(__DIR__))
    ->withRouting(
        web: __DIR__.'/../routes/web.php',
        commands: __DIR__.'/../routes/console.php',
        health: '/up',
    )
    ->withMiddleware(function (Middleware $middleware): void {
        // Global, after TrustProxies and maintenance mode, before routing: 301s to the one canonical URL + 410s.
        $middleware->append(CanonicalizeUrl::class);
    })
    ->withExceptions(function (Exceptions $exceptions): void {
        //
    })->create();
