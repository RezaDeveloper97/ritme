/**
 * Service-worker routing rules (L8-02) — pure functions, unit-tested by `node --test tests/js`.
 *
 * The worker only ever touches same-origin GETs it recognises; everything else goes straight to the network
 * untouched ("bypass"): admin + Livewire + Filament, transactional and personal pages (cart, checkout, orders,
 * bookings, `…/done`), search, newsletter token pages, the version endpoint and the worker itself.
 */

/** Path prefixes never served from or written to a cache (matched as `/x` or `/x/…`). */
export const NEVER_CACHE = [
    '/admin',
    '/livewire',
    '/filament',
    '/css/filament',
    '/js/filament',
    '/fonts/filament',
    '/search',
    '/shop/cart',
    '/shop/checkout',
    '/shop/order',
    '/shop/done',
    '/directory/booked',
    '/newsletter',
    '/pwa',
    '/sw.js',
    '/manifest.webmanifest',
    '/storage',
    '/up',
    '/_preview',
    '/_components',
];

/** Query parameters that never make a different page (mirrors SeoManager::TRACKING_PARAMS). */
const TRACKING = [/^utm_/, /^(gclid|gbraid|wbraid|dclid|fbclid|msclkid|yclid|twclid|igshid|srsltid|_ga|_gl|mc_cid|mc_eid|ref|source)$/];

const underPrefix = (path, prefix) => path === prefix || path.startsWith(`${prefix}/`);

/** True for paths the worker must never cache. `extra` = more prefixes (the configured admin path). */
export function isNeverCached(path, extra = []) {
    const lower = path.toLowerCase();
    if (lower.endsWith('/done') || lower.includes('/done/')) return true;

    return [...NEVER_CACHE, ...extra].some((prefix) => underPrefix(lower, prefix.toLowerCase()));
}

/**
 * What the worker does with a request:
 *  - "page"   — navigation: network first (3 s), then the cached page, then /offline,
 *  - "static" — hashed build assets and icons: cache first (immutable),
 *  - "media"  — /media/*: stale-while-revalidate, capped,
 *  - "bypass" — not handled at all.
 *
 * @param {{method: string, url: string, mode?: string, headers?: {has(name: string): boolean}}} request
 * @param {string} origin  the worker's own origin
 * @param {string[]} extra additional never-cache prefixes
 */
export function classify(request, origin, extra = []) {
    if (request.method !== 'GET') return 'bypass';

    let url;
    try {
        url = new URL(request.url);
    } catch {
        return 'bypass';
    }
    if (url.origin !== origin || isNeverCached(url.pathname, extra)) return 'bypass';
    if (request.headers?.has('range')) return 'bypass'; // media streaming: let the browser talk to the server

    if (request.mode === 'navigate') return 'page';
    if (url.pathname.startsWith('/build/') || url.pathname.startsWith('/icons/')) return 'static';
    if (url.pathname.startsWith('/media/')) return 'media';

    return 'bypass';
}

/**
 * May this response be stored? Only complete same-origin 200s that are not redirects, not `no-store` and not an
 * admin response (Laravel sends admin/Livewire pages with the relaxed CSP containing 'unsafe-eval' — a second
 * guard should the admin path differ from the build-time list).
 */
export function isCacheable(response) {
    if (!response || response.status !== 200 || response.redirected) return false;
    if (response.type !== 'basic' && response.type !== 'default') return false;

    const cacheControl = response.headers.get('cache-control') || '';
    if (/\bno-store\b/i.test(cacheControl)) return false;

    const csp = response.headers.get('content-security-policy') || '';

    return !csp.includes("'unsafe-eval'");
}

/** Navigation responses are cached only when they are HTML documents. */
export function isHtml(response) {
    return (response.headers.get('content-type') || '').toLowerCase().includes('text/html');
}

/** Cache key for a page: tracking parameters dropped, so `/?source=pwa` and `/` share one entry. */
export function pageKey(href) {
    const url = new URL(href);
    url.hash = '';
    for (const name of [...url.searchParams.keys()]) {
        if (TRACKING.some((pattern) => pattern.test(name))) url.searchParams.delete(name);
    }
    url.search = url.searchParams.toString() ? `?${url.searchParams.toString()}` : '';

    return url.href;
}
