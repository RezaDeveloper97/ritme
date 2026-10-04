// node --test tests/js — PWA (L8-02): build ids / update tiers, service-worker routing rules, the precache list and
// (when `npm run build` has produced it) the generated public/sw.js in a simulated worker scope.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import vm from 'node:vm';
import { buildStamp, compareBuildIds, updateTier } from '../../resources/js/sw/version.js';
import { classify, isCacheable, isNeverCached, pageKey } from '../../resources/js/sw/rules.js';
import { adminPrefixes, precacheFromManifest, resolveBuildId } from '../../tools/build-sw.mjs';

const ORIGIN = 'https://ritme.test';
const OLD = '20261001090000-aaaaaaa';
const NOW = '20261004120000-bbbbbbb';
const NEW = '20261005080000-ccccccc';

test('build ids compare by their UTC time stamp, never as plain strings', () => {
    assert.equal(buildStamp(NOW), '20261004120000');
    assert.equal(buildStamp('20261004120000'), '20261004120000');
    assert.equal(buildStamp('dev'), null);
    assert.equal(buildStamp('v1.2.3'), null);
    assert.equal(compareBuildIds(OLD, NOW), -1);
    assert.equal(compareBuildIds(NEW, NOW), 1);
    assert.equal(compareBuildIds('20261004120000-zzz', '20261004120000-aaa'), 0); // sha is informational
    assert.equal(compareBuildIds('dev', NOW), 0); // unknown never triggers anything
    assert.equal(compareBuildIds(NOW, null), 0);
    assert.equal(compareBuildIds('20261004120000', '9'), 0);
});

test('update tier: forced below the minimum, soft when a newer build is deployed, else none', () => {
    assert.equal(updateTier(NOW, { build_id: NOW, min_build_id: null }), 'none');
    assert.equal(updateTier(NOW, { build_id: NEW, min_build_id: null }), 'soft');
    assert.equal(updateTier(NOW, { build_id: NEW, min_build_id: OLD }), 'soft');
    assert.equal(updateTier(NOW, { build_id: NEW, min_build_id: NEW }), 'forced');
    assert.equal(updateTier(OLD, { build_id: NOW, min_build_id: NOW }), 'forced');
    assert.equal(updateTier(NOW, { build_id: NOW, min_build_id: NOW }), 'none');
    assert.equal(updateTier('dev', { build_id: NEW, min_build_id: NEW }), 'none');
    assert.equal(updateTier(NOW, null), 'none');
    assert.equal(updateTier(NOW, { build_id: 'garbage', min_build_id: 'garbage' }), 'none');
});

const req = (path, init = {}) => ({ method: 'GET', mode: 'no-cors', url: new URL(path, ORIGIN).href, headers: new Headers(), ...init });

test('routing: never touches admin, Livewire, POST, transactional, search, version or foreign requests', () => {
    const bypass = [
        '/admin', '/admin/settings/pwa', '/livewire/update', '/filament/exports/1', '/search?q=x', '/shop/cart', '/shop/checkout',
        '/shop/order/AB-12', '/directory/booked/X1', '/directory/join/done', '/newsletter/confirm/abc', '/pwa/version.json',
        '/sw.js', '/manifest.webmanifest', '/_preview/layout/dark', '/css/filament/filament/app.css',
    ];
    for (const path of bypass) {
        assert.equal(classify(req(path, { mode: 'navigate' }), ORIGIN), 'bypass', path);
    }
    assert.equal(classify(req('/blog/x/view', { method: 'POST' }), ORIGIN), 'bypass');
    assert.equal(classify(req('/contact', { method: 'POST', mode: 'navigate' }), ORIGIN), 'bypass');
    assert.equal(classify({ ...req('/'), url: 'https://cdn.example.com/x.js' }, ORIGIN), 'bypass');
    assert.equal(classify(req('/media/a.mp4', { headers: new Headers({ range: 'bytes=0-' }) }), ORIGIN), 'bypass');
    // a renamed admin path (ADMIN_PATH) is passed in at build time
    assert.equal(classify(req('/panel/login', { mode: 'navigate' }), ORIGIN, ['/panel']), 'bypass');
    assert.equal(isNeverCached('/ADMIN/x'), true);
    assert.equal(isNeverCached('/administrators-guide'), false); // prefix match is per path segment
});

test('routing: pages network-first, build assets and icons cache-first, media stale-while-revalidate', () => {
    assert.equal(classify(req('/', { mode: 'navigate' }), ORIGIN), 'page');
    assert.equal(classify(req('/blog/some-post', { mode: 'navigate' }), ORIGIN), 'page');
    assert.equal(classify(req('/offline', { mode: 'navigate' }), ORIGIN), 'page');
    assert.equal(classify(req('/build/assets/app-123.css'), ORIGIN), 'static');
    assert.equal(classify(req('/icons/icon-192.png'), ORIGIN), 'static');
    assert.equal(classify(req('/media/2026/10/a-768.webp'), ORIGIN), 'media');
    assert.equal(classify(req('/favicon.ico'), ORIGIN), 'bypass');
});

test('cacheable responses: complete 200s only — no redirects, no-store or admin (unsafe-eval CSP)', () => {
    const html = (init = {}, headers = {}) => new Response('<p>x</p>', { status: 200, headers: { 'content-type': 'text/html', ...headers }, ...init });
    assert.equal(isCacheable(html()), true);
    assert.equal(isCacheable(html({}, { 'cache-control': 'private, no-cache' })), true);
    assert.equal(isCacheable(html({}, { 'cache-control': 'no-store, max-age=0' })), false);
    assert.equal(isCacheable(html({}, { 'content-security-policy': "script-src 'self' 'unsafe-inline' 'unsafe-eval'" })), false);
    assert.equal(isCacheable(html({ status: 404 })), false);
    assert.equal(isCacheable({ status: 200, redirected: true, type: 'basic', headers: new Headers() }), false);
    assert.equal(isCacheable({ status: 0, type: 'opaque', headers: new Headers() }), false);
    assert.equal(isCacheable(null), false);
});

test('page cache keys drop tracking parameters and fragments', () => {
    assert.equal(pageKey(`${ORIGIN}/?source=pwa`), `${ORIGIN}/`);
    assert.equal(pageKey(`${ORIGIN}/blog?utm_source=x&page=2#top`), `${ORIGIN}/blog?page=2`);
    assert.equal(pageKey(`${ORIGIN}/tools?calc=ovulation`), `${ORIGIN}/tools?calc=ovulation`);
});

test('precache list: hashed public assets, /offline and icons — never admin entries, maps or JSON', () => {
    const list = precacheFromManifest({
        'resources/js/app.js': { file: 'assets/app-1.js', src: 'resources/js/app.js', isEntry: true },
        'resources/css/app.css': { file: 'assets/app-2.css', src: 'resources/css/app.css', assets: ['assets/vazir-3.woff2'] },
        'resources/js/modules/menu.js': { file: 'assets/menu-4.js', css: ['assets/menu-5.css'] },
        'resources/css/filament/admin/theme.css': { file: 'assets/theme-6.css', src: 'resources/css/filament/admin/theme.css' },
        'resources/svg/sprite.svg': { file: 'assets/sprite-7.svg', src: 'resources/svg/sprite.svg' },
        'x.map': { file: 'assets/app-1.js.map' },
    }, ['icon-192.png', 'apple-touch-icon.png']);

    assert.deepEqual(list, [
        '/build/assets/app-1.js', '/build/assets/app-2.css', '/build/assets/menu-4.js', '/build/assets/menu-5.css',
        '/build/assets/sprite-7.svg', '/build/assets/vazir-3.woff2', '/offline', '/icons/apple-touch-icon.png', '/icons/icon-192.png',
    ]);
});

test('build id and admin prefixes from the environment', () => {
    assert.match(resolveBuildId({}, new Date('2026-10-04T12:30:45Z')), /^20261004123045-[0-9a-z]+$/);
    assert.equal(resolveBuildId({ BUILD_ID: NOW }), NOW);
    assert.match(resolveBuildId({ BUILD_ID: 'nope' }), /^\d{14}-/);
    assert.deepEqual(adminPrefixes({ ADMIN_PATH: 'admin' }), []);
    assert.deepEqual(adminPrefixes({ ADMIN_PATH: '/panel/' }), ['/panel']);
});

/* ---------------------------------------------------------------- generated worker, simulated */

const SW_FILE = new URL('../../public/sw.js', import.meta.url);

/** A minimal ServiceWorkerGlobalScope: listeners, Cache Storage in memory, a scripted fetch. */
function workerScope(fetchImpl) {
    const listeners = {};
    const stores = new Map();
    const keyOf = (input) => (typeof input === 'string' ? new URL(input, ORIGIN).href : input.url);
    const open = async (name) => {
        if (!stores.has(name)) stores.set(name, new Map());
        const store = stores.get(name);
        return {
            match: async (key) => store.get(keyOf(key))?.clone(),
            put: async (key, response) => void store.set(keyOf(key), response),
            delete: async (key) => store.delete(keyOf(key)),
            keys: async () => [...store.keys()].map((url) => ({ url })),
            addAll: async (requests) => {
                for (const request of requests) {
                    const response = await fetchImpl(request);
                    if (!response.ok) throw new TypeError(`precache failed: ${request.url}`);
                    store.set(keyOf(request), response);
                }
            },
        };
    };
    const caches = {
        open,
        keys: async () => [...stores.keys()],
        delete: async (name) => stores.delete(name),
        match: async (key, options = {}) => {
            const names = options.cacheName ? [options.cacheName] : [...stores.keys()];
            for (const name of names) {
                const hit = stores.get(name)?.get(keyOf(key));
                if (hit) return hit.clone();
            }
            return undefined;
        },
    };
    class WorkerRequest extends Request {
        constructor(input, init) {
            super(typeof input === 'string' ? new URL(input, ORIGIN) : input, init);
        }
    }
    const self = {
        location: new URL(ORIGIN),
        addEventListener: (type, fn) => {
            listeners[type] = fn;
        },
        skipWaiting: () => {
            self.skipped = true;
        },
        clients: { claim: async () => undefined },
    };
    const context = vm.createContext({ self, caches, fetch: fetchImpl, Request: WorkerRequest, Response, Headers, URL, setTimeout, clearTimeout, Promise, console });
    vm.runInContext(readFileSync(SW_FILE, 'utf8'), context);

    const lifecycle = async (type) => {
        const pending = [];
        listeners[type]({ waitUntil: (promise) => pending.push(promise) });
        await Promise.all(pending);
    };
    const dispatchFetch = async (request) => {
        let response;
        const pending = [];
        listeners.fetch({ request, respondWith: (promise) => (response = promise), waitUntil: (promise) => pending.push(promise) });
        const result = response === undefined ? undefined : await response;
        await Promise.all(pending);
        return result;
    };

    return { self, caches, stores, listeners, lifecycle, dispatchFetch };
}

const htmlResponse = (body, headers = {}) => new Response(body, { status: 200, headers: { 'content-type': 'text/html; charset=utf-8', ...headers } });

test('generated sw.js: precaches, serves cached pages and /offline when offline, never caches admin', { skip: !existsSync(SW_FILE) && 'run npm run build first' }, async () => {
    let online = true;
    const fetchImpl = async (input) => {
        if (!online) throw new TypeError('Failed to fetch');
        const url = new URL(typeof input === 'string' ? input : input.url, ORIGIN);
        if (url.pathname === '/admin') return htmlResponse('<p>admin</p>', { 'content-security-policy': "script-src 'self' 'unsafe-inline' 'unsafe-eval'" });
        if (url.pathname === '/offline') return htmlResponse('<h1>offline</h1>');
        if (url.pathname.startsWith('/build/') || url.pathname.startsWith('/icons/')) return new Response('asset', { status: 200 });
        return htmlResponse(`<h1>${url.pathname}</h1>`);
    };
    const sw = workerScope(fetchImpl);

    await sw.lifecycle('install');
    const [shell] = [...sw.stores.keys()];
    assert.match(shell, /^ritme-shell-\d{14}-/);
    assert.ok(sw.stores.get(shell).has(`${ORIGIN}/offline`));
    assert.equal(sw.self.skipped, undefined, 'a new worker waits for the visitor (soft tier)');

    // stale caches of an older build are removed on activate
    await sw.caches.open('ritme-pages-20200101000000-old');
    await sw.lifecycle('activate');
    assert.equal(sw.stores.has('ritme-pages-20200101000000-old'), false);

    const nav = (path) => ({ method: 'GET', mode: 'navigate', url: `${ORIGIN}${path}`, headers: new Headers() });

    assert.equal(await (await sw.dispatchFetch(nav('/blog/a'))).text(), '<h1>/blog/a</h1>');
    assert.equal(await sw.dispatchFetch(nav('/admin')), undefined, 'admin is not intercepted');
    assert.equal(await sw.dispatchFetch({ ...nav('/contact'), method: 'POST' }), undefined);

    online = false;
    assert.equal(await (await sw.dispatchFetch(nav('/blog/a'))).text(), '<h1>/blog/a</h1>', 'visited page from cache');
    assert.equal(await (await sw.dispatchFetch(nav('/never-seen'))).text(), '<h1>offline</h1>', 'offline fallback');
    const asset = [...sw.stores.get(shell).keys()].find((url) => url.includes('/build/'));
    assert.equal(await (await sw.dispatchFetch({ ...nav(new URL(asset).pathname), mode: 'no-cors' })).text(), 'asset', 'precached asset');

    const cachedUrls = [...sw.stores.values()].flatMap((store) => [...store.keys()]);
    assert.ok(!cachedUrls.some((url) => url.includes('/admin')));

    sw.listeners.message({ data: { type: 'SKIP_WAITING' }, ports: [] });
    assert.equal(sw.self.skipped, true);
});
