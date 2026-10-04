<?php

declare(strict_types=1);

namespace App\Http\Middleware;

use App\Domain\Seo\SeoManager;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CacheKey;
use Closure;
use Illuminate\Contracts\Auth\Factory as AuthFactory;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\Cookie\QueueingFactory as CookieJar;
use Illuminate\Foundation\Vite;
use Illuminate\Http\Request;
use Illuminate\Support\Str;
use Symfony\Component\HttpFoundation\BinaryFileResponse;
use Symfony\Component\HttpFoundation\Response;
use Symfony\Component\HttpFoundation\StreamedResponse;

/**
 * Guest full-page cache over the cache-aside `pages` namespace (bumped by every CacheBumpingObserver, so any
 * content / settings / SEO change invalidates rendered pages).
 *
 * Runs in the `web` group right after the session has started (priority: before SubstituteBindings), so a HIT
 * costs one cache read and no database query. Only guest GET/HEAD requests are served or stored, and only
 * 200 text/html responses without cookies, flashes or a `Cache-Control: no-store` opt-out are stored.
 *
 * Bypass (X-Page-Cache: BYPASS): admin / Livewire / Filament / excluded paths, transactional noindex routes
 * (SeoManager::NOINDEX_ROUTES + pagecache.except_routes), authenticated users or `remember_*` cookies, a
 * bypass cookie (cart), session flash data or validation errors, a request `Cache-Control: no-cache` /
 * `Pragma: no-cache`, query parameters outside the whitelist (tracking parameters are ignored).
 *
 * Keys include the Vite manifest hash, so a new build invalidates every page on its own.
 *
 * CSRF: cached forms keep working without a token endpoint or JS — before storing, the session's CSRF token
 * (and the per-request CSP nonce) are replaced by placeholders, and on every HIT they are swapped for the
 * current visitor's token / nonce. Pages that carry a token are flagged so HttpCacheHeaders never makes them
 * publicly cacheable.
 */
final class PageCache
{
    /** Request attribute: the request was eligible (HIT or MISS) — read by HttpCacheHeaders. */
    public const ATTR_ELIGIBLE = 'page_cache.eligible';

    /** Request attribute: the page contains the visitor's CSRF token (must stay private, keeps its cookie). */
    public const ATTR_CSRF = 'page_cache.csrf';

    public const HEADER = 'X-Page-Cache';

    private const CSRF_PLACEHOLDER = '__RT_PAGE_CACHE_CSRF__';

    private const NONCE_PLACEHOLDER = '__RT_PAGE_CACHE_NONCE__';

    /** Response headers kept in the cached payload. */
    private const STORED_HEADERS = ['content-type', 'content-language', 'x-robots-tag', 'link'];

    public function __construct(
        private readonly CacheAside $cache,
        private readonly Config $config,
        private readonly AuthFactory $auth,
        private readonly CookieJar $cookies,
        private readonly Vite $vite,
    ) {}

    public function handle(Request $request, Closure $next): Response
    {
        $query = $this->cacheableQuery($request);

        if ($query === null || ! $this->eligible($request)) {
            return $this->mark($next($request), 'BYPASS');
        }

        $request->attributes->set(self::ATTR_ELIGIBLE, true);
        // The Vite manifest hash is part of the key: a new build (deploy) never serves HTML pointing at deleted assets.
        $key = CacheKey::make('pages', 'html', (string) ($this->vite->manifestHash() ?? 'nobuild'), sha1($request->getSchemeAndHttpHost().$request->getBaseUrl().$request->getPathInfo().'?'.$query));

        $fresh = null;
        $ttl = $this->config->get('pagecache.ttl');

        /** @var array{status: int, headers: array<string, string>, content: string}|null $payload */
        $payload = $this->cache->remember($key, is_numeric($ttl) ? (int) $ttl : null, function () use ($request, $next, &$fresh): ?array {
            $fresh = $next($request);

            return $this->storable($request, $fresh) ? $this->payload($request, $fresh) : null;
        });

        if ($fresh instanceof Response) {
            if ($payload === null) {
                $request->attributes->remove(self::ATTR_ELIGIBLE);

                return $this->mark($fresh, 'BYPASS');
            }

            $request->attributes->set(self::ATTR_CSRF, str_contains($payload['content'], self::CSRF_PLACEHOLDER));

            return $this->mark($fresh, 'MISS');
        }

        if (! is_array($payload)) { // another process stored garbage or the store failed: render normally
            $request->attributes->remove(self::ATTR_ELIGIBLE);

            return $this->mark($next($request), 'BYPASS');
        }

        return $this->mark($this->restore($request, $payload), 'HIT');
    }

    /**
     * Normalised query string for the key ('' when none), or null when the query makes the page uncacheable.
     */
    private function cacheableQuery(Request $request): ?string
    {
        $whitelist = (array) $this->config->get('pagecache.query_whitelist', []);
        $kept = [];

        foreach ($request->query->all() as $name => $value) {
            $name = (string) $name;

            if (Str::is(SeoManager::TRACKING_PARAMS, $name)) {
                continue;
            }

            if (! in_array($name, $whitelist, true) || ! is_scalar($value)) {
                return null;
            }

            $kept[$name] = (string) $value;
        }

        ksort($kept);

        return http_build_query($kept);
    }

    private function eligible(Request $request): bool
    {
        if (! $this->config->get('pagecache.enabled', true) || ! $request->isMethodCacheable() || ! $request->hasSession()) {
            return false;
        }

        if ($request->is(...$this->exceptPaths())) {
            return false;
        }

        $route = $request->route();
        $routeName = is_object($route) ? $route->getName() : null;
        if ($routeName !== null && Str::is([...SeoManager::NOINDEX_ROUTES, ...(array) $this->config->get('pagecache.except_routes', [])], $routeName)) {
            return false;
        }

        $cacheControl = strtolower((string) $request->headers->get('Cache-Control', '').','.$request->headers->get('Pragma', ''));
        if (str_contains($cacheControl, 'no-cache') || str_contains($cacheControl, 'no-store')) {
            return false;
        }

        foreach (array_keys($request->cookies->all()) as $cookie) {
            if (str_starts_with((string) $cookie, 'remember_') || in_array($cookie, (array) $this->config->get('pagecache.bypass_cookies', []), true)) {
                return false;
            }
        }

        return ! $this->hasFlash($request) && ! $this->auth->guard()->check();
    }

    /**
     * @return list<string>
     */
    private function exceptPaths(): array
    {
        return [
            ...array_values(array_map('strval', (array) $this->config->get('pagecache.except_paths', []))),
            ...AdminPaths::patterns($this->config),
        ];
    }

    private function hasFlash(Request $request): bool
    {
        $session = $request->session();

        return $session->has('errors')
            || (array) $session->get('_flash.old', []) !== []
            || (array) $session->get('_flash.new', []) !== [];
    }

    private function storable(Request $request, Response $response): bool
    {
        if ($response->getStatusCode() !== 200
            || $response instanceof StreamedResponse
            || $response instanceof BinaryFileResponse
            || ! str_starts_with(strtolower((string) $response->headers->get('Content-Type', '')), 'text/html')
            || $response->headers->getCookies() !== []
            || $this->cookies->getQueuedCookies() !== []
            || $response->headers->hasCacheControlDirective('no-store')
            || ! is_string($response->getContent())
        ) {
            return false;
        }

        // The render itself may have flashed, logged someone in or touched a bypass cookie.
        return ! $this->hasFlash($request) && ! $this->auth->guard()->check();
    }

    /**
     * @return array{status: int, headers: array<string, string>, content: string}
     */
    private function payload(Request $request, Response $response): array
    {
        $headers = [];
        foreach (self::STORED_HEADERS as $name) {
            $value = $response->headers->get($name);
            if ($value !== null) {
                $headers[$name] = $value;
            }
        }

        $replace = [];
        $token = $this->csrfToken($request);
        if ($token !== null) {
            $replace[$token] = self::CSRF_PLACEHOLDER;
        }
        $nonce = $this->vite->cspNonce();
        if (is_string($nonce) && $nonce !== '') {
            $replace['nonce="'.$nonce.'"'] = 'nonce="'.self::NONCE_PLACEHOLDER.'"';
        }

        return [
            'status' => $response->getStatusCode(),
            'headers' => $headers,
            'content' => strtr((string) $response->getContent(), $replace),
        ];
    }

    /**
     * @param  array{status: int, headers: array<string, string>, content: string}  $payload
     */
    private function restore(Request $request, array $payload): Response
    {
        $content = $payload['content'];
        $hasCsrf = str_contains($content, self::CSRF_PLACEHOLDER);

        $nonce = $this->vite->cspNonce();
        $content = strtr($content, [
            self::CSRF_PLACEHOLDER => $this->csrfToken($request) ?? '',
            'nonce="'.self::NONCE_PLACEHOLDER.'"' => is_string($nonce) && $nonce !== '' ? 'nonce="'.$nonce.'"' : '',
        ]);

        $request->attributes->set(self::ATTR_CSRF, $hasCsrf);

        return new Response($content, $payload['status'], $payload['headers']);
    }

    private function csrfToken(Request $request): ?string
    {
        $token = $request->hasSession() ? $request->session()->token() : null;

        return is_string($token) && $token !== '' ? $token : null;
    }

    private function mark(Response $response, string $state): Response
    {
        $response->headers->set(self::HEADER, $state);

        return $response;
    }
}
