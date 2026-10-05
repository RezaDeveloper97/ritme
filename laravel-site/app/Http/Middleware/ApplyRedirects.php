<?php

declare(strict_types=1);

namespace App\Http\Middleware;

use App\Domain\Content\LegacyUrlMap;
use App\Domain\Seo\Redirects\Actions\RecordNotFound;
use App\Domain\Seo\Redirects\Actions\RecordRedirectHit;
use App\Domain\Seo\Redirects\Contracts\RedirectMapRepository;
use App\Domain\Seo\Redirects\Data\RedirectMatch;
use App\Domain\Seo\Redirects\Support\RedirectPath;
use Closure;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Routing\Router;
use Illuminate\Routing\UrlGenerator;
use Symfony\Component\HttpFoundation\Response;
use Throwable;

/**
 * Admin redirects (L7-03), global, registered right before CanonicalizeUrl so it wraps it:
 *
 *   - non-canonical or legacy URLs (the ones CanonicalizeUrl / LegacyUrlMap would 301 or 410): an admin redirect for
 *     the normalised path wins, in one hop;
 *   - otherwise the request runs normally and the map is consulted only when the response is a 404.
 *
 * Live pages never read the map (zero cost); a lookup is one cache read (CachedRedirectMapRepository). An unmatched
 * 404 is noted by RecordNotFound (cache only, flushed by the scheduler). GET/HEAD only; admin paths are skipped.
 */
final class ApplyRedirects
{
    public function __construct(
        private readonly RedirectMapRepository $redirects,
        private readonly RecordRedirectHit $recordHit,
        private readonly RecordNotFound $recordNotFound,
        private readonly Config $config,
        private readonly Router $router,
        private readonly UrlGenerator $url,
    ) {}

    public function handle(Request $request, Closure $next): Response
    {
        if (! in_array($request->getMethod(), ['GET', 'HEAD'], true) || AdminPaths::matches($request, $this->config)) {
            return $next($request);
        }

        $raw = $request->getPathInfo();
        $path = RedirectPath::normalize($raw);

        if ($this->isNonCanonical($raw, $path)) {
            $redirect = $this->lookup($request, $path);
            if ($redirect !== null) {
                return $redirect;
            }
        }

        $response = $next($request);
        if ($response->getStatusCode() !== 404) {
            return $response;
        }

        $redirect = $this->lookup($request, $path);
        if ($redirect !== null) {
            return $redirect;
        }

        try {
            $this->recordNotFound->handle($path, $request->headers->get('referer'), $request->userAgent());
        } catch (Throwable $e) {
            report($e); // the monitor must never break a 404 page
        }

        return $response;
    }

    /** URLs CanonicalizeUrl would 301 / 410 (uncollapsed slashes, trailing slash, `*.html`, WordPress slugs). */
    private function isNonCanonical(string $raw, string $path): bool
    {
        if (rawurldecode($raw) !== $path || LegacyUrlMap::isGone($raw)) {
            return true;
        }

        $legacy = LegacyUrlMap::routeFor($path);
        if ($legacy === null || ! $this->router->has($legacy)) {
            return false;
        }

        // `/cycle` is both a WordPress slug and the live route: only a different route path is a legacy URL.
        return (string) parse_url($this->url->route($legacy, [], false), PHP_URL_PATH) !== $path;
    }

    private function lookup(Request $request, string $path): ?Response
    {
        $match = $this->redirects->map()->match($path);
        if ($match === null) {
            return null;
        }

        if (! $match->code->isGone() && ($match->target === null || $this->pointsToItself($match, $path))) {
            return null; // a regex that resolves to the requested path again: never loop
        }

        $this->recordHit->handle($match->id);

        if ($match->code->isGone()) {
            abort(410);
        }

        return new RedirectResponse($this->targetUrl($request, (string) $match->target), $match->code->value);
    }

    private function pointsToItself(RedirectMatch $match, string $path): bool
    {
        $local = RedirectPath::local((string) $match->target, (string) $this->config->get('app.url', ''));

        return $local !== null && RedirectPath::key($local['path']) === RedirectPath::key($path);
    }

    /**
     * Absolute URL of the target; the request's query string is kept unless the target has its own.
     */
    private function targetUrl(Request $request, string $target): string
    {
        $query = $request->getQueryString();

        if (str_starts_with($target, '/') && ! str_starts_with($target, '//')) {
            [$path, $targetQuery] = array_pad(explode('?', $target, 2), 2, null);
            $url = $request->getSchemeAndHttpHost().$request->getBaseUrl().RedirectPath::encode((string) $path);
            $query = $targetQuery ?? $query;
        } else {
            $url = $target;
            if (str_contains($target, '?')) {
                $query = null;
            }
        }

        return $url.($query === null || $query === '' ? '' : '?'.$query);
    }
}
