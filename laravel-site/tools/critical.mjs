#!/usr/bin/env node
/**
 * critical — build-time critical CSS per template + per-template asset budget (L9-01).
 *
 *   node tools/critical.mjs                    # after `vite build`: generate critical CSS, then check the budget
 *   node tools/critical.mjs --budget           # budget check only (existing build + critical files)
 *   node tools/critical.mjs --budget --lab     # + throttled lab metrics (150 ms RTT, 1.6 Mbps, 4× CPU) for docs
 *   node tools/critical.mjs --no-budget        # generate only
 *   node tools/critical.mjs --base http://127.0.0.1:8000 --json /tmp/budget.json
 *
 * Runs in `npm run build` (vite build → build-sw → critical). Nothing here runs on the server: the output is shipped
 * in public/build/critical/ — `<template>.css` + `manifest.json` (stylesheet it was cut from, route patterns →
 * template, sha256 per file for the CSP, page modules to modulepreload). App\View\Components\Layout\Assets inlines the
 * template's file in a <style> (hash allowed by App\Http\Middleware\SecurityHeaders), preloads the full stylesheet and
 * applies it from the end of <body> (no JS, CSP-safe). No/stale manifest → the plain blocking stylesheet.
 *
 * How the cut is made: the built stylesheet is split into rules (strings/comments/nesting aware); each template's
 * sample URLs are loaded in headless system Chrome (CDP over Node's built-in WebSocket, no npm deps) at 390×844,
 * 1024×768 and 1440×900 with the full stylesheet; a style rule is kept when one of its selectors (user-action pseudo-classes and
 * pseudo-elements stripped) matches an element above the fold (1.25 viewports) or a display:none element whose parent
 * is above the fold (so hiding rules always apply), plus every child of an above-the-fold grid container (placement).
 * @font-face/@property/@layer statements are always kept,
 * @keyframes when referenced. Media/supports wrappers are kept around kept rules regardless of the viewport, so
 * every breakpoint variant of a fold element is present. Unmapped routes (search, join, cart, errors…) use
 * `default`, the union of all templates.
 *
 * Its own `php -S` (Laravel router, PAGE_CACHE_ENABLED=false so no page is cached without its critical CSS) on a free
 * port unless --base is given. If Chrome is missing or the site does not answer (fresh checkout without a database),
 * it warns, writes nothing (pages keep the blocking stylesheet) and exits 0; CRITICAL_STRICT=1 makes that an error.
 * CRITICAL_SKIP=1 skips the step. Exit codes: 0 ok · 1 budget exceeded or error.
 */
import { spawn } from 'node:child_process';
import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { createServer } from 'node:net';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { gzipSync } from 'node:zlib';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const BUILD_DIR = resolve(ROOT, 'public/build');
const OUT_DIR = resolve(BUILD_DIR, 'critical');
const STYLESHEET = 'resources/css/app.css';
const DEFAULT_CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const VIEWPORTS = [{ width: 390, height: 844, mobile: true }, { width: 1024, height: 768, mobile: false }, { width: 1440, height: 900, mobile: false }];
const FOLD = 1.25;
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

/**
 * Templates: sample URLs (or a link discovered on another page) and the route-name patterns (Str::is) they serve.
 * Order matters only for display. `default` (union) serves everything else.
 */
export const TEMPLATES = [
    { key: 'home', urls: ['/'], routes: ['home'] },
    { key: 'stage', urls: ['/cycle', '/ttc', '/pregnancy', '/postpartum', '/menopause', '/teen'], routes: ['stage.*'] },
    {
        key: 'page',
        urls: ['/services', '/plus', '/tools', '/about', '/social-responsibility', '/privacy', '/terms', '/faq', '/contact', '/directory/business'],
        routes: ['services', 'plus', 'tools', 'about', 'social-responsibility', 'privacy', 'terms', 'faq', 'contact', 'directory.business'],
    },
    { key: 'blog', urls: ['/blog'], routes: ['blog.index', 'blog.category', 'blog.tag', 'blog.author'] },
    { key: 'post', discover: { from: '/blog', pattern: /href="(?:https?:\/\/[^/"]+)?(\/blog\/(?!category\/|tag\/|author\/|feed)[^"/?#]+)"/ }, routes: ['blog.show'] },
    { key: 'shop', urls: ['/shop'], routes: ['shop.index'] },
    // L9-02: the category listing needs its own above-the-fold cut (sharing /shop's shifted the whole <main>, CLS 0.89).
    { key: 'category', discover: { from: '/shop', pattern: /href="(?:https?:\/\/[^/"]+)?(\/shop\/category\/[^"/?#]+)"/ }, routes: ['shop.category'] },
    { key: 'product', discover: { from: '/shop', pattern: /href="(?:https?:\/\/[^/"]+)?(\/shop\/product\/[^"/?#]+)"/ }, routes: ['shop.product'] },
    { key: 'directory', urls: ['/directory'], routes: ['directory.index', 'directory.city', 'directory.category'] },
    {
        key: 'misc',
        urls: ['/search', '/offline', '/directory/join', '/shop/cart'],
        routes: ['search', 'pwa.offline', 'directory.join', 'directory.join.done', 'directory.booked', 'shop.cart', 'shop.checkout', 'shop.order'],
    },
    { key: 'place', discover: { from: '/directory', pattern: /href="(?:https?:\/\/[^/"]+)?(\/directory\/place\/[^"/?#]+)"/ }, routes: ['directory.place'] },
];

/**
 * Budget per template (gzip level 6 — Apache mod_deflate's default — for text; woff2/raster as shipped), measured at
 * 390×844 on first load with an empty cache, until the network is idle (lazy below-the-fold images excluded).
 * Task targets: HTML < 40 KB, CSS < 25 KB, JS < 15 KB, ≤ 2 preloaded fonts; CLS < 0.1; LCP < 2.5 s (lab only).
 * css = external stylesheet + inline critical CSS. font: the 12 shipped subsets are kept as designed — every page
 * renders Vazirmatn 400–800 + Lalezar, and ASCII punctuation («» . : ·) needs each weight's Latin file (exception
 * justified in docs/PERFORMANCE.md). Per-template overrides below.
 */
export const BUDGET = {
    html: 40 * 1024,
    css: 25 * 1024,
    js: 15 * 1024,
    font: 256 * 1024,
    image: 200 * 1024,
    requests: 26,
    preloadFonts: 2,
    renderBlocking: 0,
    cls: 0.1,
    lcp: 2500,
};
export const BUDGET_OVERRIDES = {
    // Empty since L9-02 (product chip row now `flex-nowrap` + overflow; lab CLS 0.001).
};

/* ------------------------------------------------------------------------------------------------ *
 * CSS: split minified CSS into a rule tree and serialise a subset
 * ------------------------------------------------------------------------------------------------ */

const GROUP_AT_RULES = new Set(['media', 'supports', 'layer', 'container', 'scope', 'starting-style', 'document']);

/** Index just past the token at `i` when it is a string or comment, else -1. */
function skipToken(text, i) {
    const c = text[i];
    if (c === '"' || c === "'") {
        let j = i + 1;
        while (j < text.length && text[j] !== c) j += text[j] === '\\' ? 2 : 1;
        return j + 1;
    }
    if (c === '/' && text[i + 1] === '*') {
        const end = text.indexOf('*/', i + 2);
        return end === -1 ? text.length : end + 2;
    }
    return -1;
}

/** @returns {Array<{kind:'rule',selector:string,body:string}|{kind:'group',prelude:string,children:any[]}|{kind:'atomic',name:string,text:string}|{kind:'statement',text:string}>} */
export function parseCss(text) {
    const nodes = [];
    let i = 0;
    while (i < text.length) {
        while (i < text.length && /\s/.test(text[i])) i++;
        if (text.startsWith('/*', i)) { i = skipToken(text, i); continue; }
        if (i >= text.length) break;
        // prelude up to a top-level "{" or ";"
        let j = i, depth = 0;
        while (j < text.length) {
            const skip = skipToken(text, j);
            if (skip !== -1) { j = skip; continue; }
            const c = text[j];
            if (c === '(' || c === '[') depth++;
            else if (c === ')' || c === ']') depth--;
            else if (depth === 0 && (c === '{' || c === ';')) break;
            j++;
        }
        const prelude = text.slice(i, j).trim();
        if (j >= text.length) { if (prelude) nodes.push({ kind: 'statement', text: prelude }); break; }
        if (text[j] === ';') { nodes.push({ kind: 'statement', text: `${prelude};` }); i = j + 1; continue; }
        // balanced block
        let k = j + 1; depth = 1;
        while (k < text.length && depth > 0) {
            const skip = skipToken(text, k);
            if (skip !== -1) { k = skip; continue; }
            if (text[k] === '{') depth++;
            else if (text[k] === '}') depth--;
            k++;
        }
        const body = text.slice(j + 1, k - 1);
        if (prelude.startsWith('@')) {
            const name = /^@([\w-]+)/.exec(prelude)?.[1]?.toLowerCase() ?? '';
            if (GROUP_AT_RULES.has(name)) nodes.push({ kind: 'group', prelude, children: parseCss(body) });
            else nodes.push({ kind: 'atomic', name, prelude, text: `${prelude}{${body}}` });
        } else {
            nodes.push({ kind: 'rule', selector: prelude, body });
        }
        i = k;
    }
    return nodes;
}

/** Top-level comma split of a selector list. */
export function splitSelectors(list) {
    const out = [];
    let depth = 0, start = 0;
    for (let i = 0; i < list.length; i++) {
        const skip = skipToken(list, i);
        if (skip !== -1) { i = skip - 1; continue; }
        const c = list[i];
        if (c === '(' || c === '[') depth++;
        else if (c === ')' || c === ']') depth--;
        else if (c === ',' && depth === 0) { out.push(list.slice(start, i).trim()); start = i + 1; }
    }
    out.push(list.slice(start).trim());
    return out.filter(Boolean);
}

/** Every distinct selector of every style rule in the tree. */
export function collectSelectors(nodes, into = new Set()) {
    for (const n of nodes) {
        if (n.kind === 'rule') splitSelectors(n.selector).forEach((s) => into.add(s));
        else if (n.kind === 'group') collectSelectors(n.children, into);
    }
    return into;
}

/** Serialise the subset of `nodes` whose style rules have a selector in `keep`. */
export function serialiseSubset(nodes, keep) {
    const pass = (list) => {
        let out = '';
        for (const n of list) {
            if (n.kind === 'rule') {
                if (splitSelectors(n.selector).some((s) => keep.has(s))) out += `${n.selector}{${n.body}}`;
            } else if (n.kind === 'group') {
                const inner = pass(n.children);
                if (inner !== '' || /^@layer\b/i.test(n.prelude) && n.children.length === 0) out += `${n.prelude}{${inner}}`;
            } else if (n.kind === 'atomic') {
                if (n.name !== 'keyframes' && !n.name.endsWith('-keyframes')) out += n.text;
                else out += `\u0000KF:${n.prelude.replace(/^@[\w-]+\s*/, '')}\u0000${n.text}\u0000`;
            } else {
                out += n.text;
            }
        }
        return out;
    };
    let css = pass(nodes);
    // keyframes only when their name is used by a kept rule
    const used = css.replace(/\u0000KF:[^\u0000]*\u0000[^\u0000]*\u0000/g, '');
    css = css.replace(/\u0000KF:([^\u0000]*)\u0000([^\u0000]*)\u0000/g, (_, name, text) => (used.includes(name.trim()) ? text : ''));
    return css;
}

/* ------------------------------------------------------------------------------------------------ *
 * Page-side probes (evaluated in Chrome)
 * ------------------------------------------------------------------------------------------------ */

/** Returns indices of selectors that match an above-the-fold element, plus the data-module names on the page. */
const FOLD_PROBE = `(selectors, fold) => {
  const limit = innerHeight * fold;
  const set = new Set([document.documentElement, document.body]);
  for (const el of document.body.querySelectorAll('*')) {
    const cs = getComputedStyle(el);
    if (cs.display === 'none') { if (set.has(el.parentElement)) set.add(el); continue; }
    if (el.getClientRects().length === 0) continue;
    const r = el.getBoundingClientRect();
    if (r.bottom >= 0 && r.top <= limit) set.add(el);
  }
  // Grid items below the fold still place the items above it (explicit rows/columns, subgrid): keep every child
  // of an above-the-fold grid container (the child itself, not its subtree).
  for (const el of [...set]) {
    if (el.children.length && /grid/.test(getComputedStyle(el).display)) for (const c of el.children) set.add(c);
  }
  const DYN = /::?(?:before|after|placeholder|marker|selection|file-selector-button|backdrop|first-line|first-letter|-webkit-[\\w-]+|-moz-[\\w-]+|hover|focus-visible|focus-within|focus|active|visited|target|checked|open|popover-open|placeholder-shown|autofill|user-invalid|invalid|valid|disabled|enabled|indeterminate|default|required|optional|read-only|read-write|modal)(?![\\w-])(?:\\([^)]*\\))?/g;
  const hit = [];
  selectors.forEach((sel, i) => {
    let s = sel.replace(DYN, '').trim();
    if (s === '' || /[>+~]$/.test(s)) s = s === '' ? '*' : s + ' *';
    try {
      for (const el of document.querySelectorAll(s)) { if (set.has(el)) { hit.push(i); return; } }
    } catch (e) { hit.push(i); }
  });
  const modules = [...new Set([...document.querySelectorAll('[data-module]')].flatMap((el) => el.dataset.module.split(/\\s+/)).filter(Boolean))];
  return { hit, modules, fold: set.size };
}`;

const METRICS_PROBE = `(() => {
  window.__lab = { cls: 0, lcp: 0, fcp: 0, csp: [] };
  document.addEventListener('securitypolicyviolation', (e) => window.__lab.csp.push(e.violatedDirective + ' ' + (e.blockedURI || 'inline')), true);
  try {
    new PerformanceObserver((l) => { for (const e of l.getEntries()) if (!e.hadRecentInput) window.__lab.cls += e.value; }).observe({ type: 'layout-shift', buffered: true });
    new PerformanceObserver((l) => { const e = l.getEntries(); window.__lab.lcp = e[e.length - 1].startTime; }).observe({ type: 'largest-contentful-paint', buffered: true });
    new PerformanceObserver((l) => { for (const e of l.getEntries()) if (e.name === 'first-contentful-paint') window.__lab.fcp = e.startTime; }).observe({ type: 'paint', buffered: true });
  } catch (e) {}
})();`;

const PAGE_FACTS = `(() => ({
  lab: window.__lab,
  preloadFonts: document.querySelectorAll('head link[rel=preload][as=font]').length,
  renderBlocking: document.querySelectorAll('head link[rel=stylesheet]:not([media=print])').length
    + [...document.querySelectorAll('head script[src]')].filter((s) => !s.async && !s.defer && s.type !== 'module').length,
  inlineCss: [...document.querySelectorAll('style')].reduce((n, s) => n + s.textContent.length, 0),
  h1: document.querySelectorAll('h1').length,
}))()`;

/* ------------------------------------------------------------------------------------------------ *
 * Chrome + CDP
 * ------------------------------------------------------------------------------------------------ */

export async function launchChrome() {
    const bin = process.env.CHROME_PATH || DEFAULT_CHROME;
    if (!existsSync(bin)) throw new Error(`Chrome not found at ${bin} (set CHROME_PATH)`);
    const profile = mkdtempSync(join(tmpdir(), 'ritme-critical-'));
    const proc = spawn(bin, [
        '--headless=new', '--remote-debugging-port=0', `--user-data-dir=${profile}`,
        '--no-first-run', '--no-default-browser-check', '--disable-extensions', '--disable-sync',
        '--disable-background-networking', '--disable-component-update', '--disable-default-apps',
        '--disable-domain-reliability', '--metrics-recording-only', '--no-pings', '--mute-audio',
        '--hide-scrollbars', '--force-device-scale-factor=1', '--force-color-profile=srgb', 'about:blank',
    ], { stdio: ['ignore', 'ignore', 'pipe'] });
    let stderr = '';
    proc.stderr.on('data', (d) => { stderr = (stderr + d).slice(-4000); });
    const portFile = join(profile, 'DevToolsActivePort');
    for (let i = 0; i < 200 && !existsSync(portFile); i++) {
        if (proc.exitCode !== null) throw new Error(`Chrome exited: ${stderr.slice(-500)}`);
        await sleep(50);
    }
    if (!existsSync(portFile)) { proc.kill('SIGKILL'); throw new Error('Chrome did not open a DevTools port'); }
    const [port, path] = readFileSync(portFile, 'utf8').trim().split('\n');
    const close = async () => {
        proc.kill('SIGTERM');
        for (let i = 0; i < 40 && proc.exitCode === null; i++) await sleep(50);
        if (proc.exitCode === null) proc.kill('SIGKILL');
        rmSync(profile, { recursive: true, force: true });
    };
    return { wsUrl: `ws://127.0.0.1:${port}${path}`, close };
}

export class Cdp {
    static connect(url) {
        return new Promise((res, rej) => {
            const ws = new WebSocket(url);
            ws.addEventListener('open', () => res(new Cdp(ws)));
            ws.addEventListener('error', () => rej(new Error(`CDP connect failed: ${url}`)));
        });
    }

    constructor(ws) {
        this.ws = ws; this.id = 0; this.pending = new Map(); this.handlers = new Set();
        ws.addEventListener('message', (ev) => {
            const msg = JSON.parse(typeof ev.data === 'string' ? ev.data : Buffer.from(ev.data).toString());
            if (msg.id && this.pending.has(msg.id)) {
                const { res, rej } = this.pending.get(msg.id);
                this.pending.delete(msg.id);
                msg.error ? rej(new Error(`${msg.error.message} ${msg.error.data ?? ''}`)) : res(msg.result);
            } else if (msg.method) {
                for (const h of this.handlers) h(msg);
            }
        });
    }

    send(method, params = {}, sessionId) {
        const id = ++this.id;
        this.ws.send(JSON.stringify({ id, method, params, ...(sessionId ? { sessionId } : {}) }));
        return new Promise((res, rej) => this.pending.set(id, { res, rej }));
    }

    on(fn) { this.handlers.add(fn); return () => this.handlers.delete(fn); }

    close() { this.ws.close(); }
}

const isLocal = (url) => url.startsWith('data:') || url.startsWith('blob:') || url.startsWith('about:')
    || /^https?:\/\/(127\.0\.0\.1|localhost|\[::1\])(:\d+)?\//.test(url);

/**
 * Opens a fresh tab (own browser context → empty cache), runs `fn(s, state)` and closes it. The state tracks requests
 * until network idle; non-local requests are failed and recorded.
 */
export async function withTab(cdp, fn) {
    const { browserContextId } = await cdp.send('Target.createBrowserContext', { disposeOnDetach: true });
    const { targetId } = await cdp.send('Target.createTarget', { url: 'about:blank', browserContextId });
    const { sessionId } = await cdp.send('Target.attachToTarget', { targetId, flatten: true });
    const s = (m, p) => cdp.send(m, p, sessionId);
    const state = { inflight: new Set(), last: Date.now(), loaded: false, requests: new Map(), external: [], status: null };
    const off = cdp.on((msg) => {
        if (msg.sessionId !== sessionId) return;
        const p = msg.params;
        switch (msg.method) {
            case 'Fetch.requestPaused':
                if (isLocal(p.request.url)) s('Fetch.continueRequest', { requestId: p.requestId }).catch(() => {});
                else { state.external.push(p.request.url); s('Fetch.failRequest', { requestId: p.requestId, errorReason: 'BlockedByClient' }).catch(() => {}); }
                break;
            case 'Network.requestWillBeSent':
                if (!p.request.url.startsWith('data:')) {
                    state.inflight.add(p.requestId);
                    state.requests.set(p.requestId, { url: p.request.url, type: p.type, status: 0, mime: '', done: false });
                }
                state.last = Date.now();
                break;
            case 'Network.responseReceived': {
                const r = state.requests.get(p.requestId);
                if (r) { r.status = p.response.status; r.mime = p.response.mimeType; r.type = p.type; }
                if (p.type === 'Document' && state.status === null) state.status = p.response.status;
                break;
            }
            case 'Network.loadingFinished':
            case 'Network.loadingFailed': {
                const r = state.requests.get(p.requestId);
                if (r) r.done = msg.method === 'Network.loadingFinished';
                state.inflight.delete(p.requestId);
                state.last = Date.now();
                break;
            }
            case 'Page.loadEventFired': state.loaded = true; break;
        }
    });
    state.idle = async (quiet = 500, max = 20000) => {
        const start = Date.now();
        while (Date.now() - start < max) {
            if (state.loaded && state.inflight.size === 0 && Date.now() - state.last >= quiet) return;
            await sleep(80);
        }
    };
    state.navigate = async (url) => {
        state.loaded = false; state.status = null;
        const nav = await s('Page.navigate', { url });
        if (nav.errorText) throw new Error(`${nav.errorText} for ${url}`);
        await state.idle();
        await s('Runtime.evaluate', { expression: 'document.fonts ? document.fonts.ready.then(() => 1) : 1', awaitPromise: true });
    };
    try {
        await s('Fetch.enable', { patterns: [{ urlPattern: '*' }] });
        await Promise.all([s('Network.enable'), s('Page.enable'), s('Runtime.enable')]);
        await s('Network.setCacheDisabled', { cacheDisabled: true });
        await s('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value: 'light' }] });
        return await fn(s, state);
    } finally {
        off();
        await cdp.send('Target.closeTarget', { targetId }).catch(() => {});
        await cdp.send('Target.disposeBrowserContext', { browserContextId }).catch(() => {});
    }
}

/** Run `jobs` (async fns) with at most `n` at a time. */
async function pool(jobs, n) {
    const results = new Array(jobs.length);
    let next = 0;
    await Promise.all(Array.from({ length: Math.min(n, jobs.length) }, async () => {
        while (next < jobs.length) {
            const i = next++;
            results[i] = await jobs[i]();
        }
    }));
    return results;
}

/* ------------------------------------------------------------------------------------------------ *
 * Server
 * ------------------------------------------------------------------------------------------------ */

function freePort() {
    return new Promise((res, rej) => {
        const srv = createServer();
        srv.unref();
        srv.on('error', rej);
        srv.listen(0, '127.0.0.1', () => { const { port } = srv.address(); srv.close(() => res(port)); });
    });
}

async function startServer() {
    const port = await freePort();
    const router = existsSync(join(ROOT, 'server.php')) ? join(ROOT, 'server.php') : join(ROOT, 'vendor/laravel/framework/src/Illuminate/Foundation/resources/server.php');
    const proc = spawn('php', ['-S', `127.0.0.1:${port}`, router], {
        cwd: join(ROOT, 'public'), detached: true, stdio: ['ignore', 'ignore', 'pipe'],
        env: { ...process.env, PHP_CLI_SERVER_WORKERS: '6', PAGE_CACHE_ENABLED: 'false' },
    });
    let stderr = '';
    proc.stderr.on('data', (d) => { stderr = (stderr + d).slice(-4000); });
    const base = `http://127.0.0.1:${port}`;
    let up = false;
    for (let i = 0; i < 100 && !up; i++) {
        if (proc.exitCode !== null) throw new Error(`php -S exited: ${stderr.slice(-400)}`);
        try { up = (await fetch(`${base}/`, { signal: AbortSignal.timeout(3000) })).ok; } catch { /* not up yet */ }
        if (!up) await sleep(200);
    }
    const stop = async () => {
        if (proc.exitCode === null) {
            try { process.kill(-proc.pid, 'SIGTERM'); } catch { /* gone */ }
            for (let i = 0; i < 40 && proc.exitCode === null; i++) await sleep(50);
            if (proc.exitCode === null) { try { process.kill(-proc.pid, 'SIGKILL'); } catch { /* gone */ } }
        }
    };
    if (!up) { await stop(); throw new Error(`site did not answer 200 at ${base}/ ${stderr.slice(-300)}`); }
    return { base, stop };
}

async function resolveUrls(base, template) {
    if (template.urls) return template.urls;
    try {
        const html = await (await fetch(base + template.discover.from, { signal: AbortSignal.timeout(10000) })).text();
        const m = template.discover.pattern.exec(html);
        return m ? [m[1]] : [];
    } catch {
        return [];
    }
}

async function answers(url) {
    try {
        const r = await fetch(url, { redirect: 'manual', signal: AbortSignal.timeout(10000) });
        await r.arrayBuffer();
        return r.status === 200;
    } catch {
        return false;
    }
}

/* ------------------------------------------------------------------------------------------------ *
 * Generate
 * ------------------------------------------------------------------------------------------------ */

function viteManifest() {
    const file = join(BUILD_DIR, 'manifest.json');
    if (!existsSync(file)) throw new Error('public/build/manifest.json missing — run `vite build` first');
    return JSON.parse(readFileSync(file, 'utf8'));
}

const sha256 = (text) => createHash('sha256').update(text).digest('base64');
const gz = (buf) => gzipSync(buf, { level: 6 }).length;

async function generate(cdp, base) {
    const manifest = viteManifest();
    const entry = manifest[STYLESHEET];
    if (!entry?.file) throw new Error(`${STYLESHEET} missing from the Vite manifest`);
    const cssText = readFileSync(join(BUILD_DIR, entry.file), 'utf8');
    const tree = parseCss(cssText);
    const selectors = [...collectSelectors(tree)];

    rmSync(OUT_DIR, { recursive: true, force: true }); // pages must render with the plain stylesheet while probing

    const jobs = [];
    const plan = [];
    for (const t of TEMPLATES) {
        const urls = [];
        for (const u of await resolveUrls(base, t)) {
            if (await answers(base + u)) urls.push(u); else console.warn(`! critical: ${t.key} sample ${u} does not answer 200 — skipped`);
        }
        plan.push({ template: t, urls });
        for (const u of urls) {
            for (const vp of VIEWPORTS) {
                jobs.push(() => withTab(cdp, async (s, state) => {
                    await s('Emulation.setDeviceMetricsOverride', { width: vp.width, height: vp.height, deviceScaleFactor: 1, mobile: vp.mobile });
                    await state.navigate(base + u);
                    await sleep(100);
                    const { result, exceptionDetails } = await s('Runtime.evaluate', {
                        expression: `(${FOLD_PROBE})(${JSON.stringify(selectors)}, ${FOLD})`, returnByValue: true,
                    });
                    if (exceptionDetails) throw new Error(`probe failed on ${u}: ${exceptionDetails.text}`);
                    return { key: t.key, url: u, ...result.value };
                }));
            }
        }
    }
    const results = await pool(jobs, 6);

    const out = { version: 1, stylesheet: entry.file, generated_at: new Date().toISOString(), default: 'default', routes: {}, templates: {} };
    mkdirSync(OUT_DIR, { recursive: true });
    const union = new Set();
    const emit = (key, keep, modules) => {
        const css = serialiseSubset(tree, keep).replace(/<\/style/gi, '<\\/style');
        writeFileSync(join(OUT_DIR, `${key}.css`), css);
        out.templates[key] = { file: `${key}.css`, sha256: sha256(css), bytes: Buffer.byteLength(css), gzip: gz(Buffer.from(css)), modules };
    };
    let allModules = null;
    for (const { template, urls } of plan) {
        const mine = results.filter((r) => r.key === template.key);
        if (urls.length === 0 || mine.length === 0) continue;
        const keep = new Set();
        mine.forEach((r) => r.hit.forEach((i) => { keep.add(selectors[i]); union.add(selectors[i]); }));
        // modulepreload only modules present on every sample page of the template
        const modules = mine.map((r) => r.modules).reduce((a, b) => a.filter((m) => b.includes(m)));
        allModules = allModules === null ? modules : allModules.filter((m) => modules.includes(m));
        emit(template.key, keep, modules);
        for (const pattern of template.routes) out.routes[pattern] = template.key;
    }
    if (union.size === 0) throw new Error('no template could be probed');
    emit('default', union, allModules ?? []);
    writeFileSync(join(OUT_DIR, 'manifest.json'), `${JSON.stringify(out, null, 2)}\n`);

    const full = gz(Buffer.from(cssText));
    console.log(`critical CSS from ${entry.file} (${(full / 1024).toFixed(1)} KB gz, ${selectors.length} selectors):`);
    for (const [key, t] of Object.entries(out.templates)) {
        console.log(`  ${key.padEnd(10)} ${(t.bytes / 1024).toFixed(1).padStart(5)} KB raw  ${(t.gzip / 1024).toFixed(1).padStart(5)} KB gz  modules: ${t.modules.join(' ') || '—'}`);
    }
    return out;
}

/* ------------------------------------------------------------------------------------------------ *
 * Budget
 * ------------------------------------------------------------------------------------------------ */

function category(r) {
    const path = r.url.replace(/[?#].*$/, '');
    if (r.type === 'Document') return 'html';
    if (r.type === 'Stylesheet' || /\.css$/.test(path)) return 'css';
    if (r.type === 'Script' || /\.m?js$/.test(path)) return 'js';
    if (r.type === 'Font' || /\.woff2?$/.test(path)) return 'font';
    if (r.type === 'Image' || /\.(svg|avif|webp|png|jpe?g|gif|ico)$/.test(path)) return 'image';
    return 'other';
}

const TEXTUAL = /^(text\/|application\/(javascript|json|manifest\+json|xml)|image\/svg)/;

async function measure(cdp, base, url, lab) {
    return withTab(cdp, async (s, state) => {
        await s('Emulation.setDeviceMetricsOverride', { width: 390, height: 844, deviceScaleFactor: 1, mobile: true });
        if (lab) {
            await s('Network.emulateNetworkConditions', { offline: false, latency: 150, downloadThroughput: 1.6 * 1024 * 1024 / 8, uploadThroughput: 750 * 1024 / 8 });
            await s('Emulation.setCPUThrottlingRate', { rate: 4 });
        }
        await s('Page.addScriptToEvaluateOnNewDocument', { source: METRICS_PROBE });
        await state.navigate(base + url);
        await sleep(lab ? 1500 : 400);
        await state.idle(400);
        const facts = (await s('Runtime.evaluate', { expression: PAGE_FACTS, returnByValue: true })).result.value;
        const totals = { html: 0, css: 0, js: 0, font: 0, image: 0, other: 0 };
        const files = [];
        for (const [id, r] of state.requests) {
            if (!r.done) continue;
            let size = 0;
            try {
                const body = await s('Network.getResponseBody', { requestId: id });
                const buf = Buffer.from(body.body, body.base64Encoded ? 'base64' : 'utf8');
                size = TEXTUAL.test(r.mime) ? gz(buf) : buf.length;
            } catch { /* body gone (e.g. 204) */ }
            const cat = category(r);
            totals[cat] += size;
            files.push({ cat, url: r.url.replace(base, ''), size });
        }
        return { url, status: state.status, requests: [...state.requests.values()].filter((r) => r.done).length, totals, files, external: state.external, ...facts };
    });
}

function checkBudget(key, m, critical, lab) {
    const b = { ...BUDGET, ...(BUDGET_OVERRIDES[key] ?? {}) };
    const inline = critical?.templates?.[key]?.gzip ?? critical?.templates?.default?.gzip ?? 0;
    const row = {
        key, url: m.url, html: m.totals.html, inline, css: m.totals.css + inline, js: m.totals.js, font: m.totals.font,
        image: m.totals.image, requests: m.requests, preloadFonts: m.preloadFonts, renderBlocking: m.renderBlocking,
        cls: m.lab?.cls ?? 0, lcp: m.lab?.lcp ?? 0, fcp: m.lab?.fcp ?? 0, fonts: m.files.filter((f) => f.cat === 'font').map((f) => f.url.replace(/^.*\//, '')),
    };
    const fails = [];
    for (const k of ['html', 'css', 'js', 'font', 'image', 'requests', 'preloadFonts', 'renderBlocking']) {
        if (row[k] > b[k]) fails.push(`${k} ${row[k]} > ${b[k]}`);
    }
    if (row.cls >= b.cls) fails.push(`cls ${row.cls.toFixed(3)} ≥ ${b.cls}`);
    if (lab && row.lcp >= b.lcp) fails.push(`lcp ${Math.round(row.lcp)} ms ≥ ${b.lcp}`);
    if (critical && !m.inlineCss) fails.push('critical CSS not inlined (stale page cache? `php artisan cache:ns bump pages`)');
    if (m.lab?.csp?.length) fails.push(`CSP violations: ${m.lab.csp.join(', ')}`);
    if (m.external.length) fails.push(`external requests: ${m.external.join(', ')}`);
    if (m.status !== 200) fails.push(`status ${m.status}`);
    return { ...row, fails };
}

async function budget(cdp, base, lab) {
    const critFile = join(OUT_DIR, 'manifest.json');
    const critical = existsSync(critFile) ? JSON.parse(readFileSync(critFile, 'utf8')) : null;
    const rows = [];
    for (const t of TEMPLATES) {
        const urls = await resolveUrls(base, t);
        if (!urls[0] || !(await answers(base + urls[0]))) { console.warn(`! budget: ${t.key} has no answering sample — skipped`); continue; }
        rows.push(checkBudget(t.key, await measure(cdp, base, urls[0], lab), critical, lab));
    }
    const kb = (n) => (n / 1024).toFixed(1);
    console.log(`\nbudget (390×844, empty cache, gzip -6${lab ? ', lab 150 ms / 1.6 Mbps / 4× CPU' : ''}):`);
    console.log('  template   html  css(inl)    js  font image  req pre blk   cls' + (lab ? '   fcp   lcp' : ''));
    for (const r of rows) {
        console.log(`  ${r.key.padEnd(9)} ${kb(r.html).padStart(5)} ${kb(r.css).padStart(5)}(${kb(r.inline)}) ${kb(r.js).padStart(5)} ${kb(r.font).padStart(5)} ${kb(r.image).padStart(5)} ${String(r.requests).padStart(4)} ${String(r.preloadFonts).padStart(3)} ${String(r.renderBlocking).padStart(3)} ${r.cls.toFixed(3)}`
            + (lab ? ` ${String(Math.round(r.fcp)).padStart(5)} ${String(Math.round(r.lcp)).padStart(5)}` : '')
            + (r.fails.length ? `  ✘ ${r.fails.join('; ')}` : '  ✔'));
    }
    return rows;
}

/* ------------------------------------------------------------------------------------------------ *
 * Main
 * ------------------------------------------------------------------------------------------------ */

function parseArgs(argv) {
    const o = { generate: true, budget: true, lab: false, base: null, json: null };
    for (let i = 0; i < argv.length; i++) {
        const a = argv[i];
        if (a === '--budget') o.generate = false;
        else if (a === '--no-budget') o.budget = false;
        else if (a === '--lab') o.lab = true;
        else if (a === '--base') o.base = argv[++i].replace(/\/+$/, '');
        else if (a === '--json') o.json = argv[++i];
        else if (a === '--help' || a === '-h') o.help = true;
        else throw new Error(`unknown argument ${a}`);
    }
    return o;
}

async function main(argv) {
    const opts = parseArgs(argv);
    if (opts.help) { console.log(readFileSync(fileURLToPath(import.meta.url), 'utf8').split('*/')[0]); return 0; }
    if (process.env.CRITICAL_SKIP === '1') { console.warn('! critical: skipped (CRITICAL_SKIP=1) — pages use the blocking stylesheet'); return 0; }
    if (existsSync(join(ROOT, 'public/hot'))) { console.warn('! critical: public/hot exists (vite dev server) — skipped'); return 0; }

    let server = null, chrome = null, cdp = null;
    try {
        try {
            server = opts.base ? { base: opts.base, stop: async () => {} } : await startServer();
            chrome = await launchChrome();
            cdp = await Cdp.connect(chrome.wsUrl);
        } catch (e) {
            if (opts.generate && !opts.base) rmSync(OUT_DIR, { recursive: true, force: true });
            const msg = `critical: ${e.message}`;
            if (process.env.CRITICAL_STRICT === '1' || !opts.generate) { console.error(`✘ ${msg}`); return 1; }
            console.warn(`! ${msg} — no critical CSS (pages use the blocking stylesheet)`);
            return 0;
        }
        if (opts.generate) await generate(cdp, server.base);
        if (!opts.budget) return 0;
        const rows = await budget(cdp, server.base, opts.lab);
        if (opts.json) writeFileSync(resolve(opts.json), `${JSON.stringify({ lab: opts.lab, budget: BUDGET, overrides: BUDGET_OVERRIDES, rows }, null, 2)}\n`);
        const failed = rows.filter((r) => r.fails.length);
        if (failed.length) { console.error(`✘ budget exceeded on ${failed.map((r) => r.key).join(', ')} (docs/PERFORMANCE.md)`); return 1; }
        console.log('✔ budget met on every template');
        return 0;
    } finally {
        cdp?.close();
        await chrome?.close();
        await server?.stop();
    }
}

if (import.meta.url === pathToFileURL(process.argv[1] ?? '').href) {
    main(process.argv.slice(2)).then((code) => process.exit(code), (e) => { console.error(`✘ critical: ${e.stack ?? e.message}`); process.exit(1); });
}
