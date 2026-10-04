<?php

declare(strict_types=1);

namespace App\Http\Middleware;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\LegacyUrlMap;
use Closure;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Routing\Router;
use Illuminate\Routing\UrlGenerator;
use Symfony\Component\HttpFoundation\Response;

/**
 * One canonical URL per resource, reached in a single 301 hop (global middleware, runs before routing so it also
 * covers URLs that would 404):
 *
 *   - old asset folders (`/ritme-static/assets/*`, `/wp-content/uploads/ritme/*`) → 410 (any method)
 *   - GET/HEAD only: duplicate slashes collapsed, trailing slash removed, `*.html` / `/ritme-static/*.html` /
 *     WordPress slugs → new route (LegacyUrlMap), uppercase → lowercase for static page paths only (content slugs
 *     may be Persian or mixed case and are left alone), and — when enabled — the scheme + host of APP_URL
 *     (https in production).
 *
 * The query string is kept. Host/scheme redirection is opt-in (`app.canonical_redirect` = true, meant for the
 * production .env; off in local, tests and CLI audits whose APP_URL differs from the request origin). It relies on
 * Request::isSecure(), so behind a TLS-terminating proxy configure trusted proxies first.
 */
final class CanonicalizeUrl
{
    public function __construct(
        private readonly Config $config,
        private readonly Router $router,
        private readonly UrlGenerator $url,
    ) {}

    public function handle(Request $request, Closure $next): Response
    {
        $rawPath = $request->getPathInfo();

        if (LegacyUrlMap::isGone($rawPath)) {
            abort(410);
        }

        if (! in_array($request->getMethod(), ['GET', 'HEAD'], true)) {
            return $next($request);
        }

        $path = $this->normalizePath($rawPath);
        $origin = $request->getSchemeAndHttpHost();
        $targetOrigin = $this->redirectsHost() ? $this->canonicalOrigin($request) : $origin;

        if ($path === $rawPath && $targetOrigin === $origin) {
            return $next($request);
        }

        $query = $request->getQueryString();
        $target = $targetOrigin.$request->getBaseUrl().$path.($query === null ? '' : '?'.$query);

        return new RedirectResponse($target, 301);
    }

    private function normalizePath(string $path): string
    {
        $path = preg_replace('#/{2,}#', '/', $path) ?? $path;
        if ($path !== '/') {
            $path = rtrim($path, '/');
        }
        if ($path === '') {
            return '/';
        }

        $legacy = LegacyUrlMap::routeFor($path);
        if ($legacy !== null) {
            return $this->routePath($legacy) ?? $path;
        }

        $lower = strtolower($path);
        if ($lower !== $path && $this->isStaticPath($lower)) {
            return $lower;
        }

        return $path;
    }

    /** Path (relative to the app's base URL) of a named route, or null when it is not registered. */
    private function routePath(string $routeName): ?string
    {
        if (! $this->router->has($routeName)) {
            return null;
        }

        $path = (string) parse_url($this->url->route($routeName, [], false), PHP_URL_PATH);

        return $path === '' ? '/' : $path;
    }

    private function isStaticPath(string $path): bool
    {
        foreach (StaticPage::cases() as $page) {
            if ($page->path() === $path) {
                return true;
            }
        }

        return false;
    }

    private function redirectsHost(): bool
    {
        return (bool) $this->config->get('app.canonical_redirect', false);
    }

    /**
     * Scheme, host and port of APP_URL (production APP_URL is https, so this is the https redirect); falls back to
     * https and the request host for the parts APP_URL lacks.
     */
    private function canonicalOrigin(Request $request): string
    {
        $base = parse_url((string) $this->config->get('app.url')) ?: [];
        $scheme = isset($base['scheme']) ? strtolower($base['scheme']) : 'https';
        $host = isset($base['host']) ? strtolower($base['host']) : $request->getHost();
        $port = isset($base['port']) ? ':'.$base['port'] : '';

        return $scheme.'://'.$host.$port;
    }
}
