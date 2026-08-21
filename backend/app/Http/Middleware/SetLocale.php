<?php

namespace App\Http\Middleware;

use App\Services\Language\LanguageRegistry;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/**
 * Pins the request's locale to a language the app actually ships.
 *
 * Precedence: an explicit `?locale=` (browsers forbid JS from overriding
 * Accept-Language, so the web client passes it as a query param), then the
 * Accept-Language header, then the default language. Anything unrecognised
 * falls back rather than 404s — an old client asking for a locale that has
 * since been removed still gets readable content.
 *
 * Because it sets app()->setLocale(), `__()`, `Translatable::pick()` and
 * `$model->localized()` all resolve to the same locale without every call site
 * threading it through by hand.
 *
 * `setlocale:default` ignores the request entirely and pins the default
 * language. The admin panel uses that: it is a Persian-only interface, and its
 * content lists must preview the default language regardless of what
 * Accept-Language the admin's browser happens to send.
 */
class SetLocale
{
    public function __construct(private readonly LanguageRegistry $registry) {}

    public function handle(Request $request, Closure $next, ?string $mode = null): Response
    {
        if ($mode === 'default') {
            app()->setLocale($this->registry->defaultCode());

            return $next($request);
        }

        $requested = $request->query('locale') ?? $request->header('Accept-Language');

        app()->setLocale($this->registry->resolve(is_string($requested) ? $requested : null));

        return $next($request);
    }
}
