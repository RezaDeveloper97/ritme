<?php

declare(strict_types=1);

namespace App\Http\Middleware;

use Closure;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Foundation\Vite;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/**
 * Browser / proxy caching for the HTML pages the guest page cache handles (global middleware, so it sees the final
 * response including the session cookie added by the `web` group).
 *
 *  - weak ETag over the final body + 304 Not Modified on a matching If-None-Match
 *  - `Cache-Control: public, max-age=0, s-maxage=…, stale-while-revalidate=…` when the response sets no cookie
 *  - `private, no-cache` otherwise (browsers still revalidate with the ETag)
 *
 * A brand-new visitor (no session cookie in the request) on a page without a CSRF form and with an untouched session
 * gets no session / XSRF cookie at all, so most first views are cookie-free and publicly cacheable. A visitor who
 * already has a session keeps refreshing it (private). Every other response keeps Laravel's defaults.
 */
final class HttpCacheHeaders
{
    /** Session keys that StartSession writes for every visitor; anything else means the render used the session. */
    private const PRISTINE_SESSION_KEYS = ['_token', '_previous', '_flash'];

    public function __construct(
        private readonly Config $config,
        private readonly Vite $vite,
    ) {}

    public function handle(Request $request, Closure $next): Response
    {
        $response = $next($request);

        if (! $request->attributes->get(PageCache::ATTR_ELIGIBLE)
            || ! $request->isMethodCacheable()
            || $response->getStatusCode() !== 200
            || ! is_string($response->getContent())
        ) {
            return $response;
        }

        if ($this->canDropSessionCookies($request, $response)) {
            foreach ($response->headers->getCookies() as $cookie) {
                $response->headers->removeCookie($cookie->getName(), $cookie->getPath(), $cookie->getDomain());
            }
        }

        $sMaxAge = max(0, (int) $this->config->get('pagecache.http.s_maxage', 0));

        if ($response->headers->getCookies() === [] && $sMaxAge > 0) {
            $swr = max(0, (int) $this->config->get('pagecache.http.stale_while_revalidate', 0));
            $response->headers->set('Cache-Control', 'public, max-age=0, s-maxage='.$sMaxAge.($swr > 0 ? ', stale-while-revalidate='.$swr : ''));
        } else {
            $response->headers->set('Cache-Control', 'private, no-cache');
        }

        if ($this->config->get('pagecache.http.etag', true)) {
            $response->setEtag(hash('xxh128', $this->stableContent((string) $response->getContent())), true);
            $response->isNotModified($request); // turns the response into an empty 304 on a match
        }

        return $response;
    }

    /**
     * The body without the per-request CSP nonce, so the same page keeps the same ETag (the CSRF token stays in:
     * a page with another visitor's token must not be revalidated as unchanged).
     */
    private function stableContent(string $content): string
    {
        $nonce = $this->vite->cspNonce();

        return is_string($nonce) && $nonce !== '' ? str_replace(' nonce="'.$nonce.'"', '', $content) : $content;
    }

    private function canDropSessionCookies(Request $request, Response $response): bool
    {
        $sessionCookie = (string) $this->config->get('session.cookie');

        if ($request->attributes->get(PageCache::ATTR_CSRF) || $request->cookies->has($sessionCookie) || ! $request->hasSession()) {
            return false;
        }

        foreach ($response->headers->getCookies() as $cookie) {
            if (! in_array($cookie->getName(), [$sessionCookie, 'XSRF-TOKEN'], true)) {
                return false; // the page set a cookie of its own: keep everything
            }
        }

        $session = $request->session();
        if ((array) $session->get('_flash.new', []) !== [] || (array) $session->get('_flash.old', []) !== []) {
            return false;
        }

        return array_diff(array_keys($session->all()), self::PRISTINE_SESSION_KEYS) === [];
    }
}
