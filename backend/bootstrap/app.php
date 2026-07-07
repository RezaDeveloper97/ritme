<?php

use App\Http\Middleware\EnsureAdminActive;
use App\Http\Middleware\EnsureSuperAdmin;
use Illuminate\Foundation\Application;
use Illuminate\Foundation\Configuration\Exceptions;
use Illuminate\Foundation\Configuration\Middleware;
use Illuminate\Support\Facades\Route;

return Application::configure(basePath: dirname(__DIR__))
    ->withRouting(
        web: __DIR__.'/../routes/web.php',
        api: __DIR__.'/../routes/api.php',
        commands: __DIR__.'/../routes/console.php',
        health: '/up',
        then: function (): void {
            // The admin panel ships in the same image but is only mounted when
            // ADMIN_PANEL_ENABLED=true, so the public backend container never
            // exposes it. The dedicated `admin` container sets the flag.
            if (filter_var(env('ADMIN_PANEL_ENABLED', false), FILTER_VALIDATE_BOOLEAN)) {
                Route::middleware('web')
                    ->prefix('admin')
                    ->name('admin.')
                    ->group(base_path('routes/admin.php'));
            }
        },
    )
    ->withMiddleware(function (Middleware $middleware): void {
        $middleware->alias([
            'admin.active' => EnsureAdminActive::class,
            'admin.super' => EnsureSuperAdmin::class,
        ]);

        // Web guests hitting a guarded admin page are sent to the admin login.
        $middleware->redirectGuestsTo(fn () => route('admin.login'));
    })
    ->withExceptions(function (Exceptions $exceptions): void {
        //
    })->create();
