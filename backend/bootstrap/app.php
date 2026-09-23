<?php

use App\Http\Middleware\EnsureAdminActive;
use App\Http\Middleware\EnsureSuperAdmin;
use App\Http\Middleware\SetLocale;
use App\Http\Middleware\TestClock;
use Illuminate\Auth\AuthenticationException;
use Illuminate\Foundation\Application;
use Illuminate\Foundation\Configuration\Exceptions;
use Illuminate\Foundation\Configuration\Middleware;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Route;
use Lcobucci\JWT\Validation\RequiredConstraintsViolated;
use League\OAuth2\Server\Exception\OAuthServerException;

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
                // `setlocale:default` keeps the panel previewing the default
                // language whatever the admin's browser asks for.
                Route::middleware(['web', 'setlocale:default'])
                    ->prefix('admin')
                    ->name('admin.')
                    ->group(base_path('routes/admin.php'));
            }
        },
    )
    ->withMiddleware(function (Middleware $middleware): void {
        // In production every request arrives through the nginx `proxy`
        // container; the backend port is bound to 127.0.0.1 and is not
        // reachable from outside, so there is no untrusted path that could
        // forge these headers. Without this, TLS terminating at the proxy is
        // invisible to Laravel: isSecure() stays false and the admin panel
        // emits http:// asset and redirect URLs on an https page.
        $middleware->trustProxies(
            at: '*',
            headers: Request::HEADER_X_FORWARDED_FOR
                | Request::HEADER_X_FORWARDED_HOST
                | Request::HEADER_X_FORWARDED_PORT
                | Request::HEADER_X_FORWARDED_PROTO,
        );

        // Contract/test-only fixed clock (X-Test-Now). Inert unless APP_ENV is
        // local/testing/contract AND TEST_CLOCK_ENABLED=true — never in prod.
        $middleware->prepend(TestClock::class);

        // API requests resolve their locale against the languages table before
        // a controller runs, so nothing downstream has to guess.
        $middleware->appendToGroup('api', SetLocale::class);

        $middleware->alias([
            'admin.active' => EnsureAdminActive::class,
            'admin.super' => EnsureSuperAdmin::class,
            'setlocale' => SetLocale::class,
            'swagger.auth' => \App\Http\Middleware\SwaggerBasicAuth::class,
        ]);

        // Web guests hitting a guarded admin page are sent to the admin login.
        // API requests (Accept: application/json OR /api/*) must still get a
        // 401 rather than a redirect, so only redirect non-API web requests.
        $middleware->redirectGuestsTo(
            fn ($request) => $request->is('api/*') ? null : route('admin.login')
        );
    })
    ->withExceptions(function (Exceptions $exceptions): void {
        // Passport swallows the reason a bearer token was rejected: its guard
        // reports the OAuthServerException, blanks the Authorization header and
        // lets the request fall through to a generic AuthenticationException.
        // Catch the reason while it is being reported and park it on the
        // request, so the 401 below can say *why*. Returning nothing keeps the
        // default reporting.
        $exceptions->report(function (OAuthServerException $e): void {
            if (! app()->bound('request')) {
                return;
            }

            $violations = $e->getPrevious() instanceof RequiredConstraintsViolated
                ? $e->getPrevious()->violations()
                : [];

            $code = match (true) {
                $e->getHint() === 'Access token has been revoked' => 'token_revoked',
                // Expired is only trusted when it is the sole violation, i.e.
                // the signature checked out and the token really was ours.
                count($violations) === 1 && $violations[0]->getMessage() === 'The token is expired' => 'token_expired',
                default => 'unauthenticated',
            };

            request()->attributes->set('auth_error_code', $code);
        });

        // API 401s are always JSON with a stable `error_code` the client can
        // branch on: token_expired | token_revoked | unauthenticated. Only a
        // real sign-out reason should make the app drop its stored session.
        $exceptions->render(function (AuthenticationException $e, Request $request) {
            if (! $request->is('api/*') && ! $request->expectsJson()) {
                return null;
            }

            return response()->json([
                'message' => $e->getMessage(),
                'error_code' => $request->attributes->get('auth_error_code', 'unauthenticated'),
            ], 401);
        });
    })->create();
