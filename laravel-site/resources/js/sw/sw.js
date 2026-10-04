/**
 * Service worker SOURCE (L8-02). `tools/build-sw.mjs` bundles it after `vite build` into public/sw.js, replacing
 * the three build constants below — public/sw.js is generated, git-ignored and must never be edited by hand.
 *
 * Caches (all prefixed `ritme-`; anything else of ours is deleted on activate):
 *  - shell-<build> : precache — hashed CSS/JS/fonts/sprite of this build, /offline, icons (cache first),
 *  - pages-<build> : visited HTML pages, network first with a 3 s timeout, LRU-capped,
 *  - media-v1      : /media/* images, stale-while-revalidate, LRU-capped at 60 entries.
 * Never cached: see rules.js (admin, Livewire, POST, cart/checkout/order/booked, search, no-store …).
 *
 * Updates: a new worker installs and WAITS; the page shows the soft toast and posts SKIP_WAITING when the visitor
 * accepts (or the forced screen does it). Nothing here calls skipWaiting() on its own.
 */
import { classify, isCacheable, isHtml, pageKey } from './rules.js';

/* global __BUILD_ID__, __PRECACHE__, __NEVER_CACHE_EXTRA__ */
const BUILD_ID = __BUILD_ID__;
const PRECACHE = __PRECACHE__;
const EXTRA_NEVER = __NEVER_CACHE_EXTRA__;

const OFFLINE_URL = '/offline';
const NETWORK_TIMEOUT_MS = 3000;
const PAGE_LIMIT = 40;
const MEDIA_LIMIT = 60;

const SHELL = `ritme-shell-${BUILD_ID}`;
const PAGES = `ritme-pages-${BUILD_ID}`;
const MEDIA = 'ritme-media-v1';
const CURRENT = [SHELL, PAGES, MEDIA];

self.addEventListener('install', (event) => {
    event.waitUntil(
        caches.open(SHELL).then((cache) => cache.addAll(PRECACHE.map((url) => new Request(url, { cache: 'reload' })))),
    );
});

self.addEventListener('activate', (event) => {
    event.waitUntil(
        caches
            .keys()
            .then((keys) => Promise.all(keys.filter((key) => key.startsWith('ritme-') && !CURRENT.includes(key)).map((key) => caches.delete(key))))
            .then(() => self.clients.claim()),
    );
});

self.addEventListener('message', (event) => {
    const type = event.data && event.data.type;
    if (type === 'SKIP_WAITING') {
        self.skipWaiting();
    } else if (type === 'GET_BUILD_ID' && event.ports[0]) {
        event.ports[0].postMessage({ build_id: BUILD_ID });
    }
});

self.addEventListener('fetch', (event) => {
    const kind = classify(event.request, self.location.origin, EXTRA_NEVER);

    if (kind === 'page') {
        event.respondWith(page(event));
    } else if (kind === 'static') {
        event.respondWith(cacheFirst(event.request));
    } else if (kind === 'media') {
        event.respondWith(staleWhileRevalidate(event));
    }
});

/** Oldest-first eviction: Cache keys keep insertion order and every write re-inserts its entry. */
async function trim(cache, limit) {
    const keys = await cache.keys();
    await Promise.all(keys.slice(0, Math.max(0, keys.length - limit)).map((key) => cache.delete(key)));
}

async function store(cacheName, key, response, limit) {
    const cache = await caches.open(cacheName);
    await cache.delete(key);
    await cache.put(key, response);
    await trim(cache, limit);
}

async function page(event) {
    const key = pageKey(event.request.url);
    const network = fetch(event.request);

    // Clone synchronously on arrival (before the page reads the body), store in the background.
    event.waitUntil(
        network
            .then((response) => {
                if (isCacheable(response) && isHtml(response)) {
                    return store(PAGES, key, response.clone(), PAGE_LIMIT);
                }
                return undefined;
            })
            .catch(() => undefined),
    );

    let timer;
    const timeout = new Promise((resolve) => {
        timer = setTimeout(() => resolve(null), NETWORK_TIMEOUT_MS);
    });

    try {
        const first = await Promise.race([network, timeout]);
        if (first) return first;

        const cached = await caches.match(key, { cacheName: PAGES });
        return cached || (await network);
    } catch {
        return (await caches.match(key, { cacheName: PAGES })) || (await caches.match(OFFLINE_URL, { cacheName: SHELL })) || Response.error();
    } finally {
        clearTimeout(timer);
    }
}

async function cacheFirst(request) {
    // Icons are requested with a `?v=` content hash but precached without it; build assets are hashed by name.
    const ignoreSearch = new URL(request.url).pathname.startsWith('/icons/');
    const cached = await caches.match(request, { ignoreSearch });
    if (cached) return cached;

    const response = await fetch(request);
    if (isCacheable(response)) {
        const copy = response.clone();
        caches.open(SHELL).then((cache) => cache.put(request, copy));
    }

    return response;
}

async function staleWhileRevalidate(event) {
    const request = event.request;
    const cached = await caches.match(request, { cacheName: MEDIA });
    const network = fetch(request).then((response) => {
        if (isCacheable(response)) {
            const copy = response.clone();
            event.waitUntil(store(MEDIA, request, copy, MEDIA_LIMIT));
        }
        return response;
    });

    if (cached) {
        event.waitUntil(network.catch(() => undefined));
        return cached;
    }

    return network;
}
