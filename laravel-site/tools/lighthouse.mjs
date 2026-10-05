#!/usr/bin/env node
/**
 * lighthouse — Lighthouse sweep on one URL per template, production-like (L9-02).
 *
 *   node tools/lighthouse.mjs --all                          # every template × mobile + desktop × warm + cold
 *   node tools/lighthouse.mjs --only home,product --form mobile --mode warm
 *   node tools/lighthouse.mjs --all --html                   # + HTML reports (tmp dir, never committed)
 *   node tools/lighthouse.mjs --all --base http://127.0.0.1:8000   # against a running site (no own server/proxy)
 *   node tools/lighthouse.mjs --list                         # print the templates and exit
 *   options: --retries N (default 2) · --strict-seo (count demo-content noindex) · --throttling devtools (applied
 *   throttling instead of the default simulation; unreliable on loopback) · --out DIR · LH_PROXY_LATENCY=ms (delay
 *   every proxied response, to compare with a remote server; never used for the gate)
 *
 * Targets (tasks/README.md, binding; task L9-02): Performance ≥ 95 (mobile; desktop held to the same bar), SEO 100,
 * Best Practices 100, Accessibility ≥ 95, CLS < 0.1, LCP < 2.5 s, TBT < 150 ms. Any miss → exit 1.
 *
 * How the site runs (nothing permanent is changed — no .env edits, no `artisan optimize` cache files):
 *  - its own `php -S` on a free port with APP_ENV=production, APP_DEBUG=false, PAGE_CACHE_ENABLED=true and APP_URL
 *    set to the measured origin (env vars of the child process only), serving the existing production build
 *    (`npm run build` first — critical CSS + budget come from L9-01);
 *  - in front of it a tiny Node proxy that does what `public/.htaccess` does on Apache and `php -S` cannot: gzip
 *    level 6 / brotli for the same MIME types, the immutable/1-year Cache-Control on build, media and fonts, the sw.js /
 *    manifest rules, `Vary: Accept-Encoding`, no `X-Powered-By`. Without it every text response would be measured
 *    uncompressed and uncached, which no production request is.
 *
 * Modes (the L1-07 guest page cache):
 *  - warm: the URL is fetched twice before the run, so Lighthouse's navigation is a page-cache HIT (what visitors get);
 *  - cold: Lighthouse sends `Cache-Control: no-cache`, which PageCache treats as a bypass → full Laravel render.
 *  The browser cache is always empty (Lighthouse resets storage), throttling is Lighthouse's default simulation
 *  (mobile: Moto G Power, 150 ms RTT, 1.6 Mbps, 4× CPU; desktop preset: 40 ms, 10 Mbps, 1× CPU).
 *
 * Variance: a run that misses a target is re-run twice more and the median (by performance score) counts.
 *
 * Output (docs/qa/perf/<date>/): one trimmed JSON per run (`<template>-<form>-<mode>.json`: scores, metrics, page-cache
 * header, LCP element, every non-passing audit with up to 5 items), `summary.json`, `summary.md` (the table). Full
 * Lighthouse JSON (+ HTML with --html) goes to the OS tmp dir, path printed at the end — not committed.
 *
 * Lighthouse is not a project dependency: it is resolved from LIGHTHOUSE_DIR (a node_modules dir), ./node_modules or
 * the newest copy in the npx cache (~/.npm/_npx/<hash>/node_modules, e.g. after `npx lighthouse --version`).
 * Chrome: CHROME_PATH or the system Google Chrome. Every non-local request fails the run (no-externals rule).
 *
 * Exit codes: 0 all targets met (or a documented exception) · 1 a target missed, a run failed, or a tool error.
 */
import { spawn } from 'node:child_process';
import { existsSync, mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { createServer as createHttpServer, request as httpRequest } from 'node:http';
import { createServer } from 'node:net';
import { homedir, tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { brotliCompressSync, constants as zlibConstants, gzipSync } from 'node:zlib';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const DEFAULT_CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const LATENCY = Number(process.env.LH_PROXY_LATENCY ?? 0);
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

export const TARGETS = {
    performance: 95,
    accessibility: 95,
    'best-practices': 100,
    seo: 100,
    cls: 0.1,
    lcp: 2500,
    tbt: 150,
};

/**
 * One URL per template (task scope). `discover` takes the first matching link of another page, so the sweep follows
 * whatever content the database has. `allow` lists audits that fail on purpose: a reason, or {reason, when(audit,
 * ctx)} when only a specific failure is acceptable. They are left out of that category's score (shown with a `*`),
 * every other audit still counts.
 */

/** Only a `<meta name="robots" content="noindex…">` (not robots.txt, not X-Robots-Tag) — the demo-content rule. */
const DEMO_NOINDEX = {
    reason: 'sample/demo content is noindex by design (L6-01 demo products, L7-05b lists without real content); passes once real content is published — `--strict-seo` counts it',
    when: (audit) => (audit.details?.items ?? []).length > 0
        && audit.details.items.every((i) => /name="robots"[^>]*noindex/i.test(i.source?.snippet ?? '')),
};
export const TEMPLATES = [
    { key: 'home', url: '/' },
    { key: 'stage', url: '/cycle' },
    { key: 'tools', url: '/tools' },
    { key: 'faq', url: '/faq' },
    { key: 'contact', url: '/contact' },
    { key: 'blog', url: '/blog', allow: { 'is-crawlable': DEMO_NOINDEX } },
    { key: 'article', discover: { from: '/blog', pattern: /href="(?:https?:\/\/[^/"]+)?(\/blog\/(?!category\/|tag\/|author\/|feed)[^"/?#]+)"/ }, allow: { 'is-crawlable': DEMO_NOINDEX } },
    { key: 'directory', url: '/directory', allow: { 'is-crawlable': DEMO_NOINDEX } },
    { key: 'place', discover: { from: '/directory', pattern: /href="(?:https?:\/\/[^/"]+)?(\/directory\/place\/[^"/?#]+)"/ }, allow: { 'is-crawlable': DEMO_NOINDEX } },
    { key: 'shop', url: '/shop', allow: { 'is-crawlable': DEMO_NOINDEX } },
    { key: 'category', discover: { from: '/shop', pattern: /href="(?:https?:\/\/[^/"]+)?(\/shop\/category\/[^"/?#]+)"/ }, allow: { 'is-crawlable': DEMO_NOINDEX } },
    { key: 'product', discover: { from: '/shop', pattern: /href="(?:https?:\/\/[^/"]+)?(\/shop\/product\/[^"/?#]+)"/ }, allow: { 'is-crawlable': DEMO_NOINDEX } },
    {
        key: 'cart',
        url: '/shop/cart',
        allow: { 'is-crawlable': 'cart is noindex by design (SeoManager forced rule: search/cart/checkout/done)' },
    },
    {
        key: '404',
        url: '/lighthouse-missing-page',
        status: 404,
        allow: {
            'http-status-code': 'the 404 template must answer 404',
            'is-crawlable': 'error pages are noindex by design',
            'errors-in-console': {
                reason: 'the only console error is the 404 of the document itself',
                when: (audit, ctx) => (audit.details?.items ?? []).every((i) => (i.sourceLocation?.url ?? i.url ?? '').endsWith(ctx.path)),
            },
        },
    },
];

/* ------------------------------------------------------------------------------------------------ *
 * CLI
 * ------------------------------------------------------------------------------------------------ */

function parseArgs(argv) {
    const args = { all: false, only: null, forms: ['mobile', 'desktop'], modes: ['warm', 'cold'], base: null, out: join(ROOT, 'docs/qa/perf'), html: false, list: false, retries: 2, throttling: 'simulate', strictSeo: false };
    for (let i = 0; i < argv.length; i++) {
        const a = argv[i];
        const next = () => argv[++i];
        if (a === '--all') args.all = true;
        else if (a === '--only') args.only = next().split(',').map((s) => s.trim()).filter(Boolean);
        else if (a === '--form') args.forms = next().split(',').map((s) => s.trim());
        else if (a === '--mode') args.modes = next().split(',').map((s) => s.trim());
        else if (a === '--base') args.base = next().replace(/\/$/, '');
        else if (a === '--out') args.out = resolve(next());
        else if (a === '--html') args.html = true;
        else if (a === '--retries') args.retries = Number(next());
        else if (a === '--list') args.list = true;
        else if (a === '--throttling') args.throttling = next();
        else if (a === '--strict-seo') args.strictSeo = true;
        else if (a === '-h' || a === '--help') { console.log(readFileSync(fileURLToPath(import.meta.url), 'utf8').split('*/')[0]); process.exit(0); }
        else throw new Error(`unknown argument ${a}`);
    }
    for (const f of args.forms) if (!['mobile', 'desktop'].includes(f)) throw new Error(`--form: ${f}`);
    for (const m of args.modes) if (!['warm', 'cold'].includes(m)) throw new Error(`--mode: ${m}`);
    if (!args.all && !args.only && !args.list) throw new Error('pass --all or --only <template,…> (see --list)');
    return args;
}

/* ------------------------------------------------------------------------------------------------ *
 * Lighthouse + Chrome resolution
 * ------------------------------------------------------------------------------------------------ */

function lighthouseDirs() {
    const dirs = [];
    if (process.env.LIGHTHOUSE_DIR) dirs.push(resolve(process.env.LIGHTHOUSE_DIR));
    dirs.push(join(ROOT, 'node_modules'));
    const npx = join(homedir(), '.npm/_npx');
    if (existsSync(npx)) for (const h of readdirSync(npx)) dirs.push(join(npx, h, 'node_modules'));
    return dirs
        .filter((d) => existsSync(join(d, 'lighthouse/package.json')) && existsSync(join(d, 'chrome-launcher/package.json')))
        .map((d) => ({ dir: d, version: JSON.parse(readFileSync(join(d, 'lighthouse/package.json'), 'utf8')).version }));
}

const cmpVersion = (a, b) => {
    const pa = a.split('.').map(Number), pb = b.split('.').map(Number);
    for (let i = 0; i < 3; i++) if ((pa[i] ?? 0) !== (pb[i] ?? 0)) return (pa[i] ?? 0) - (pb[i] ?? 0);
    return 0;
};

async function loadLighthouse() {
    const found = lighthouseDirs();
    if (!found.length) throw new Error('Lighthouse not found: run `npx lighthouse --version` once (npm cache) or set LIGHTHOUSE_DIR');
    // An explicit LIGHTHOUSE_DIR / local node_modules wins, otherwise the newest npx copy.
    const pick = process.env.LIGHTHOUSE_DIR || existsSync(join(ROOT, 'node_modules/lighthouse')) ? found[0] : found.sort((a, b) => cmpVersion(b.version, a.version))[0];
    const imp = (p) => import(pathToFileURL(join(pick.dir, p)).href);
    const [{ default: lighthouse }, { default: desktopConfig }, chromeLauncher] = await Promise.all([
        imp('lighthouse/core/index.js'),
        imp('lighthouse/core/config/desktop-config.js'),
        imp('chrome-launcher/dist/index.js'),
    ]);
    return { lighthouse, desktopConfig, chromeLauncher, version: pick.version, dir: pick.dir };
}

/* ------------------------------------------------------------------------------------------------ *
 * Site: php -S (production-like env) + .htaccess-equivalent proxy
 * ------------------------------------------------------------------------------------------------ */

function freePort() {
    return new Promise((res, rej) => {
        const srv = createServer();
        srv.unref();
        srv.on('error', rej);
        srv.listen(0, '127.0.0.1', () => { const { port } = srv.address(); srv.close(() => res(port)); });
    });
}

const COMPRESSIBLE = /^(text\/(html|plain|css|xml|javascript)|application\/(javascript|json|ld\+json|xml|rss\+xml|atom\+xml|manifest\+json)|image\/svg\+xml|font\/(ttf|otf))\b/i;
const MIME = { '.woff2': 'font/woff2', '.avif': 'image/avif', '.webp': 'image/webp', '.svg': 'image/svg+xml', '.webmanifest': 'application/manifest+json', '.js': 'text/javascript', '.mjs': 'text/javascript' };

/** Headers `public/.htaccess` (mod_headers / mod_expires / mod_mime) would set for this path. */
function htaccessHeaders(path, headers) {
    const ext = (path.match(/\.[a-z0-9]+$/i) || [''])[0].toLowerCase();
    if (MIME[ext] && !/charset/.test(headers['content-type'] ?? '')) {
        headers['content-type'] = MIME[ext] + (/^\.(svg|js|mjs|webmanifest)$/.test(ext) ? '; charset=utf-8' : '');
    }
    const year = 'public, max-age=31536000';
    if (/^\/(build|media)\//.test(path) || /\.(woff2?|ttf|otf)$/i.test(path)) headers['cache-control'] = `${year}, immutable`;
    else if (path === '/sw.js') headers['cache-control'] = 'public, max-age=0, must-revalidate';
    else if (path === '/manifest.webmanifest' && !headers['cache-control']) headers['cache-control'] = 'public, max-age=3600';
    else if (!headers['cache-control'] && /\.(avif|webp|jpe?g|png|gif|svg|css|js)$/i.test(path)) headers['cache-control'] = year;
    if (/\.(html|css|js|mjs|json|svg|xml|txt|webmanifest)$/i.test(path)) headers.vary = headers.vary ? `${headers.vary}, Accept-Encoding` : 'Accept-Encoding';
    delete headers['x-powered-by'];
    return headers;
}

function startProxy(target, port) {
    const server = createHttpServer((req, res) => {
        const up = httpRequest({ host: '127.0.0.1', port: target, method: req.method, path: req.url, headers: req.headers }, (ur) => {
            const chunks = [];
            ur.on('data', (c) => chunks.push(c));
            ur.on('end', () => {
                let body = Buffer.concat(chunks);
                const path = new URL(req.url, 'http://x').pathname;
                const headers = htaccessHeaders(path, { ...ur.headers });
                delete headers['content-length'];
                delete headers['transfer-encoding'];
                delete headers.connection;
                const accept = String(req.headers['accept-encoding'] ?? '');
                if (body.length > 0 && !headers['content-encoding'] && COMPRESSIBLE.test(headers['content-type'] ?? '') && req.method !== 'HEAD') {
                    if (/\bbr\b/.test(accept)) {
                        body = brotliCompressSync(body, { params: { [zlibConstants.BROTLI_PARAM_QUALITY]: 5 } });
                        headers['content-encoding'] = 'br';
                    } else if (/\bgzip\b/.test(accept)) {
                        body = gzipSync(body, { level: 6 });
                        headers['content-encoding'] = 'gzip';
                    }
                    if (headers['content-encoding'] && !/accept-encoding/i.test(headers.vary ?? '')) headers.vary = headers.vary ? `${headers.vary}, Accept-Encoding` : 'Accept-Encoding';
                }
                if (req.method !== 'HEAD') headers['content-length'] = String(body.length);
                const send = () => { res.writeHead(ur.statusCode ?? 502, headers); res.end(req.method === 'HEAD' ? undefined : body); };
                if (LATENCY > 0) setTimeout(send, LATENCY); else send();
            });
        });
        up.on('error', (e) => { res.writeHead(502); res.end(String(e)); });
        req.pipe(up);
    });
    return new Promise((res) => server.listen(port, '127.0.0.1', () => res(server)));
}

async function startSite() {
    const [phpPort, port] = [await freePort(), await freePort()];
    const base = `http://127.0.0.1:${port}`;
    const router = existsSync(join(ROOT, 'server.php')) ? join(ROOT, 'server.php') : join(ROOT, 'vendor/laravel/framework/src/Illuminate/Foundation/resources/server.php');
    const env = { ...process.env, APP_ENV: 'production', APP_DEBUG: 'false', APP_URL: base, PAGE_CACHE_ENABLED: 'true', LOG_LEVEL: 'warning' };
    // Cached pages from an older build/markup must not be measured (same step as every deploy, L1-07).
    await new Promise((res, rej) => {
        const bump = spawn('php', ['artisan', 'cache:ns', 'bump', 'pages', '--no-interaction'], { cwd: ROOT, env, stdio: 'ignore' });
        bump.on('exit', (code) => (code === 0 ? res() : rej(new Error(`php artisan cache:ns bump pages → exit ${code}`))));
    });
    const proc = spawn('php', ['-S', `127.0.0.1:${phpPort}`, router], {
        cwd: join(ROOT, 'public'), detached: true, stdio: ['ignore', 'ignore', 'pipe'],
        env: { ...env, PHP_CLI_SERVER_WORKERS: '6' },
    });
    let stderr = '';
    proc.stderr.on('data', (d) => { stderr = (stderr + d).slice(-4000); });
    const proxy = await startProxy(phpPort, port);
    const stop = async () => {
        proxy.close();
        if (proc.exitCode === null) {
            try { process.kill(-proc.pid, 'SIGTERM'); } catch { /* gone */ }
            for (let i = 0; i < 40 && proc.exitCode === null; i++) await sleep(50);
            if (proc.exitCode === null) { try { process.kill(-proc.pid, 'SIGKILL'); } catch { /* gone */ } }
        }
    };
    let up = false;
    for (let i = 0; i < 100 && !up; i++) {
        if (proc.exitCode !== null) break;
        try { up = (await fetch(`${base}/`, { signal: AbortSignal.timeout(5000) })).ok; } catch { /* not up yet */ }
        if (!up) await sleep(200);
    }
    if (!up) { await stop(); throw new Error(`site did not answer 200 at ${base}/ ${stderr.slice(-400)}`); }
    return { base, stop };
}

async function resolveUrl(base, t) {
    if (t.url) return t.url;
    const html = await (await fetch(base + t.discover.from, { signal: AbortSignal.timeout(15000) })).text();
    const m = t.discover.pattern.exec(html);
    return m ? m[1] : null;
}

async function warm(url) {
    for (let i = 0; i < 2; i++) {
        const r = await fetch(url, { redirect: 'manual', signal: AbortSignal.timeout(20000) });
        await r.arrayBuffer();
    }
}

/* ------------------------------------------------------------------------------------------------ *
 * Scoring
 * ------------------------------------------------------------------------------------------------ */

const SKIP_MODES = new Set(['notApplicable', 'manual', 'informative', 'error']);

/** Category score in 0–100; audits in `allow` are dropped from the weighted mean. */
function categoryScore(lhr, id, allow) {
    const cat = lhr.categories[id];
    if (!cat) return null;
    let sum = 0, weight = 0, adjusted = false;
    for (const ref of cat.auditRefs) {
        const a = lhr.audits[ref.id];
        if (!ref.weight || !a || SKIP_MODES.has(a.scoreDisplayMode) || a.score === null) continue;
        if (allow[ref.id] && a.score < 1) { adjusted = true; continue; }
        sum += ref.weight * a.score;
        weight += ref.weight;
    }
    if (!adjusted) return { score: Math.round((cat.score ?? 0) * 100), adjusted: false };
    return { score: weight ? Math.round((sum / weight) * 100) : 100, adjusted: true, raw: Math.round((cat.score ?? 0) * 100) };
}

function trimItems(details) {
    const items = details?.items ?? [];
    return items.slice(0, 5).map((it) => {
        const out = {};
        for (const [k, v] of Object.entries(it)) {
            if (v && typeof v === 'object') {
                if (v.type === 'node') out[k] = { selector: v.selector, snippet: String(v.snippet ?? '').slice(0, 200), label: v.nodeLabel };
                else if (v.type === 'source-location') out[k] = `${v.url}:${v.line}`;
                else if (Array.isArray(v.items) || v.type === 'subitems') continue;
                else out[k] = v.value ?? v.text ?? undefined;
            } else out[k] = typeof v === 'string' ? v.slice(0, 200) : v;
        }
        return out;
    });
}

function summarise(lhr, ctx) {
    const allow = {};
    for (const [id, rule] of Object.entries(ctx.template.allow ?? {})) {
        const audit = lhr.audits[id];
        if (!audit || (audit.score ?? 1) >= 1) continue;
        if (ctx.strictSeo && rule === DEMO_NOINDEX) continue;
        if (typeof rule === 'string') allow[id] = rule;
        else if (rule.when(audit, ctx)) allow[id] = rule.reason;
    }
    const cats = {};
    for (const id of ['performance', 'accessibility', 'best-practices', 'seo']) cats[id] = categoryScore(lhr, id, allow);
    const num = (id) => lhr.audits[id]?.numericValue ?? null;
    const metrics = {
        fcp: num('first-contentful-paint'),
        lcp: num('largest-contentful-paint'),
        tbt: num('total-blocking-time'),
        cls: num('cumulative-layout-shift'),
        si: num('speed-index'),
        ttfb: num('server-response-time'),
    };
    const reqs = lhr.audits['network-requests']?.details?.items ?? [];
    const main = reqs.find((r) => r.resourceType === 'Document') ?? null;
    const origin = new URL(ctx.base).origin;
    const external = reqs.filter((r) => /^https?:/.test(r.url) && new URL(r.url).origin !== origin && !/^https?:\/\/(127\.0\.0\.1|localhost)(:|\/)/.test(r.url)).map((r) => r.url);
    const lcpEl = lhr.audits['largest-contentful-paint-element']?.details?.items?.[0]?.items?.[0]?.node
        ?? lhr.audits['lcp-breakdown-insight']?.details?.items?.find?.((i) => i.type === 'node') ?? null;
    const failing = [];
    for (const [id, a] of Object.entries(lhr.audits)) {
        if (a.score === null || a.score >= 1 || SKIP_MODES.has(a.scoreDisplayMode)) continue;
        if (a.scoreDisplayMode === 'metricSavings' && a.score >= 0.9) continue;
        failing.push({ id, title: a.title, score: a.score, displayValue: a.displayValue, allowed: allow[id] ?? undefined, items: trimItems(a.details) });
    }
    const fails = [];
    const t = TARGETS;
    if (cats.performance && cats.performance.score < t.performance) fails.push(`perf ${cats.performance.score} < ${t.performance}`);
    if (cats.accessibility && cats.accessibility.score < t.accessibility) fails.push(`a11y ${cats.accessibility.score} < ${t.accessibility}`);
    if (cats['best-practices'] && cats['best-practices'].score < t['best-practices']) fails.push(`bp ${cats['best-practices'].score} < ${t['best-practices']}`);
    if (cats.seo && cats.seo.score < t.seo) fails.push(`seo ${cats.seo.score} < ${t.seo}`);
    if (metrics.cls !== null && metrics.cls >= t.cls) fails.push(`CLS ${metrics.cls.toFixed(3)} ≥ ${t.cls}`);
    if (metrics.lcp !== null && metrics.lcp >= t.lcp) fails.push(`LCP ${Math.round(metrics.lcp)} ms ≥ ${t.lcp}`);
    if (metrics.tbt !== null && metrics.tbt >= t.tbt) fails.push(`TBT ${Math.round(metrics.tbt)} ms ≥ ${t.tbt}`);
    if (external.length) fails.push(`${external.length} external request(s)`);
    if (lhr.runtimeError) fails.push(`runtime error ${lhr.runtimeError.code}`);
    const expected = ctx.template.status ?? 200;
    if (main && main.statusCode !== expected) fails.push(`HTTP ${main.statusCode} ≠ ${expected}`);
    return {
        template: ctx.template.key,
        url: ctx.path,
        formFactor: ctx.form,
        mode: ctx.mode,
        lighthouseVersion: lhr.lighthouseVersion,
        fetchTime: lhr.fetchTime,
        status: main?.statusCode ?? null,
        pageCache: ctx.pageCache ?? null,
        scores: cats,
        metrics,
        requests: reqs.length,
        transferKB: Math.round(reqs.reduce((s, r) => s + (r.transferSize ?? 0), 0) / 102.4) / 10,
        lcpElement: lcpEl ? { selector: lcpEl.selector, snippet: String(lcpEl.snippet ?? '').slice(0, 200) } : null,
        external,
        failing,
        fails,
        pass: fails.length === 0,
    };
}

/* ------------------------------------------------------------------------------------------------ *
 * Runner
 * ------------------------------------------------------------------------------------------------ */

async function runOnce(lh, chrome, ctx, html) {
    const flags = { port: chrome.port, output: html ? ['json', 'html'] : ['json'], logLevel: 'error', onlyCategories: ['performance', 'accessibility', 'best-practices', 'seo'] };
    if (ctx.template.status && ctx.template.status >= 400) flags.ignoreStatusCode = true;
    flags.throttlingMethod = ctx.throttling;
    if (ctx.mode === 'cold') flags.extraHeaders = { 'Cache-Control': 'no-cache' };
    const config = ctx.form === 'desktop' ? lh.desktopConfig : undefined;
    // The page-cache state of the measured document, read with the same request headers right after the run would
    // change it — so read it via a HEAD with the same headers first (warm: must already be HIT).
    const probe = await fetch(ctx.base + ctx.path, { method: 'GET', redirect: 'manual', headers: ctx.mode === 'cold' ? { 'Cache-Control': 'no-cache' } : {}, signal: AbortSignal.timeout(20000) });
    await probe.arrayBuffer();
    const pageCache = probe.headers.get('x-page-cache');
    const result = await lh.lighthouse(ctx.base + ctx.path, flags, config);
    const reports = Array.isArray(result.report) ? result.report : [result.report];
    return { lhr: result.lhr, json: reports[0], html: html ? reports[1] : null, pageCache };
}

async function main() {
    const args = parseArgs(process.argv.slice(2));
    if (args.list) { for (const t of TEMPLATES) console.log(`${t.key.padEnd(10)} ${t.url ?? `(first link on ${t.discover.from})`}`); return 0; }
    const templates = TEMPLATES.filter((t) => !args.only || args.only.includes(t.key));
    if (args.only) for (const k of args.only) if (!TEMPLATES.some((t) => t.key === k)) throw new Error(`unknown template ${k}`);

    const lh = await loadLighthouse();
    console.log(`Lighthouse ${lh.version} (${lh.dir})`);
    const chromePath = process.env.CHROME_PATH || (existsSync(DEFAULT_CHROME) ? DEFAULT_CHROME : undefined);
    const site = args.base ? { base: args.base, stop: async () => {} } : await startSite();
    const chrome = await lh.chromeLauncher.launch({ chromePath, chromeFlags: ['--headless=new', '--no-first-run', '--disable-extensions', '--no-default-browser-check'] });

    const date = new Date().toISOString().slice(0, 10);
    const outDir = join(args.out, date);
    const fullDir = join(tmpdir(), 'ritme-lighthouse', date);
    mkdirSync(outDir, { recursive: true });
    mkdirSync(fullDir, { recursive: true });
    const results = [];
    let failed = false;
    try {
        console.log(`site ${site.base} · ${templates.length} templates × ${args.forms.join('+')} × ${args.modes.join('+')}`);
        for (const t of templates) {
            const path = await resolveUrl(site.base, t);
            if (!path) { console.log(`✗ ${t.key}: no URL (nothing to discover on ${t.discover.from})`); failed = true; results.push({ template: t.key, pass: false, fails: ['no URL'] }); continue; }
            for (const form of args.forms) {
                for (const mode of args.modes) {
                    const ctx = { template: t, path, form, mode, base: site.base, throttling: args.throttling, strictSeo: args.strictSeo };
                    if (mode === 'warm') await warm(site.base + path);
                    const runs = [];
                    let best = null;
                    for (let attempt = 0; attempt <= args.retries; attempt++) {
                        const r = await runOnce(lh, chrome, ctx, args.html);
                        const s = summarise(r.lhr, { ...ctx, pageCache: r.pageCache });
                        runs.push({ r, s });
                        if (attempt === 0 && s.pass) break;
                        if (attempt === 0 && args.retries > 0) process.stdout.write(`  … ${t.key} ${form} ${mode} missed (${s.fails.join('; ')}), re-running\n`);
                    }
                    // median by performance score (one run → that run)
                    runs.sort((a, b) => (a.s.scores.performance?.score ?? 0) - (b.s.scores.performance?.score ?? 0));
                    best = runs[Math.floor(runs.length / 2)];
                    const s = best.s;
                    s.runs = runs.map((x) => x.s.scores.performance?.score ?? null);
                    const name = `${t.key}-${form}-${mode}`;
                    writeFileSync(join(outDir, `${name}.json`), JSON.stringify(s, null, 2) + '\n');
                    writeFileSync(join(fullDir, `${name}.json`), best.r.json);
                    if (best.r.html) writeFileSync(join(fullDir, `${name}.html`), best.r.html);
                    results.push(s);
                    if (!s.pass) failed = true;
                    const sc = s.scores;
                    const fmt = (c) => (c ? `${c.score}${c.adjusted ? '*' : ''}` : '–');
                    console.log(`${s.pass ? '✓' : '✗'} ${name.padEnd(24)} P ${fmt(sc.performance).padStart(3)} A ${fmt(sc.accessibility).padStart(3)} BP ${fmt(sc['best-practices']).padStart(3)} SEO ${fmt(sc.seo).padStart(4)}  LCP ${(s.metrics.lcp / 1000).toFixed(2)} s  CLS ${s.metrics.cls.toFixed(3)}  TBT ${Math.round(s.metrics.tbt)} ms  cache ${s.pageCache ?? '–'}${s.fails.length ? `  ← ${s.fails.join('; ')}` : ''}`);
                }
            }
        }
    } finally {
        try { await chrome.kill(); } catch { /* gone */ }
        await site.stop();
    }

    const summary = { date, lighthouse: lh.version, base: args.base ? site.base : 'own php -S (APP_ENV=production, page cache on) + .htaccess proxy', targets: TARGETS, results };
    writeFileSync(join(outDir, 'summary.json'), JSON.stringify(summary, null, 2) + '\n');
    writeFileSync(join(outDir, 'summary.md'), markdown(summary));
    console.log(`\n${results.filter((r) => r.pass).length}/${results.length} runs meet every target → ${outDir}/summary.md`);
    console.log(`full Lighthouse reports: ${fullDir}`);
    return failed ? 1 : 0;
}

function markdown({ date, lighthouse, targets, results }) {
    const fmt = (c) => (c ? `${c.score}${c.adjusted ? '*' : ''}` : '–');
    const lines = [
        `# Lighthouse sweep — ${date}`,
        '',
        `Lighthouse ${lighthouse}, generated by \`node tools/lighthouse.mjs\`. Targets: Performance ≥ ${targets.performance}, A11y ≥ ${targets.accessibility}, BP ${targets['best-practices']}, SEO ${targets.seo}, CLS < ${targets.cls}, LCP < ${targets.lcp / 1000} s, TBT < ${targets.tbt} ms.`,
        '`*` = category score without an audit that fails on purpose (listed below). warm = page-cache HIT, cold = full render (`Cache-Control: no-cache`).',
        '',
        '| Template | URL | Form | Mode | Perf | A11y | BP | SEO | FCP | LCP | TBT | CLS | Req | KB | OK |',
        '|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|',
    ];
    for (const r of results) {
        if (!r.scores) { lines.push(`| ${r.template} | – | – | – | – | – | – | – | – | – | – | – | – | – | ✗ ${r.fails.join('; ')} |`); continue; }
        const m = r.metrics;
        lines.push(`| ${r.template} | \`${r.url}\` | ${r.formFactor} | ${r.mode} | ${fmt(r.scores.performance)} | ${fmt(r.scores.accessibility)} | ${fmt(r.scores['best-practices'])} | ${fmt(r.scores.seo)} | ${(m.fcp / 1000).toFixed(2)} s | ${(m.lcp / 1000).toFixed(2)} s | ${Math.round(m.tbt)} ms | ${m.cls.toFixed(3)} | ${r.requests} | ${r.transferKB} | ${r.pass ? '✓' : `✗ ${r.fails.join('; ')}`} |`);
    }
    const allowed = new Map();
    for (const r of results) for (const f of r.failing ?? []) if (f.allowed) allowed.set(`${r.template}: \`${f.id}\``, f.allowed);
    if (allowed.size) {
        lines.push('', '## Allowed failures', '');
        for (const [k, v] of allowed) lines.push(`- ${k} — ${v}`);
    }
    return lines.join('\n') + '\n';
}

main().then((code) => process.exit(code), (e) => { console.error(e?.stack ?? String(e)); process.exit(1); });
