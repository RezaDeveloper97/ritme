#!/usr/bin/env node
/**
 * pwa-check — repeatable PWA verification of the production build (L8-03).
 *
 *   node tools/pwa-check.mjs                 # existing production build, own `php -S` (Laravel router) on a free port
 *   node tools/pwa-check.mjs --build         # run `npm run build` first
 *   node tools/pwa-check.mjs --base http://127.0.0.1:8000 [--out docs/qa/L8] [--keep-min]
 *
 * Drives headless system Chrome over the DevTools protocol (Node ≥ 22 built-in WebSocket, no npm deps; CDP plumbing
 * like tools/shot.mjs). Chrome: /Applications/Google Chrome.app (CHROME_PATH overrides). Viewport 390×844 mobile.
 *
 * Checks (each ✔ / ✘ / ! with evidence; any ✘ → exit 1):
 *  build      production build present (no public/hot), public/sw.js stamped with public/build/build-id.json
 *  http       /manifest.webmanifest, /pwa/version.json (no-store, deployed build id), /sw.js, /offline (noindex)
 *  manifest   Page.getAppManifest without errors + required members; Page.getInstallabilityErrors empty
 *  sw         /sw.js registers, activates and controls the page
 *  precache   ritme-shell-<build> holds exactly the URLs tools/build-sw.mjs derives from the Vite manifest
 *  offline    (page + worker targets emulated offline) visited page from the page cache, unvisited → /offline,
 *             admin + cart navigations NOT answered by the worker (never-cache routes)
 *  never      online admin/cart/version.json responses not from the worker and absent from every ritme-* cache
 *  soft       a newer deployed build id (running bundle's id rewritten older on the wire) → toast «نسخه جدید آماده است»;
 *             a redeployed (byte-changed) /sw.js → waiting worker → toast → SKIP_WAITING + one reload (worker channel)
 *  forced     PwaSettings.min_build_id raised to the deployed build (artisan tinker) → blocking screen that Escape and
 *             Back cannot leave; min_build_id restored to its previous value afterwards (also on failure)
 *  install    beforeinstallprompt → own install banner on a later page view (headless Chrome may not fire it: !)
 *  external   no request to a non-local origin from any page or worker (Network domain, all targets)
 *  csp        no CSP violation (securitypolicyviolation events + console/Log reports) on any page
 *
 * Screenshots (390 wide, git-ignored PNGs) and pwa-check.json go to --out (default docs/qa/L8).
 * Exit codes: 0 all checks pass (warnings allowed) · 1 a check failed or the tool errored.
 */
import { execFileSync, spawn } from 'node:child_process';
import { existsSync, mkdirSync, mkdtempSync, readdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { createServer } from 'node:net';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { adminPrefixes, precacheFromManifest } from './build-sw.mjs';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const DEFAULT_CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const WIDTH = 390;
const HEIGHT = 844;
const FAKE_OLD_BUILD = '20000101000000-pwacheck';

const VISITED = '/cycle';
const UNVISITED = '/about';
const ADMIN = `${adminPrefixes()[0] ?? '/admin'}/login`;
const CART = '/shop/cart';

const TEXT = {
    soft: 'نسخه جدید آماده است',
    forced: 'این نسخه دیگر پشتیبانی نمی‌شود',
    install: 'ریتمی را روی صفحه اصلی داشته باش',
};

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

/* ------------------------------------------------------------------------------------------------ *
 * Arguments + results
 * ------------------------------------------------------------------------------------------------ */

function parseArgs(argv) {
    const opts = { out: 'docs/qa/L8', base: null, build: false, keepMin: false };
    for (let i = 0; i < argv.length; i++) {
        const a = argv[i];
        const val = () => {
            const v = argv[++i];
            if (v === undefined) throw new Error(`${a} needs a value`);
            return v;
        };
        switch (a) {
            case '--help': case '-h': opts.help = true; break;
            case '--build': opts.build = true; break;
            case '--base': opts.base = val().replace(/\/+$/, ''); break;
            case '--out': opts.out = val(); break;
            case '--keep-min': opts.keepMin = true; break;
            default: throw new Error(`unknown argument ${a}`);
        }
    }
    if (opts.base && !isLocalUrl(opts.base)) throw new Error(`--base must be a local origin: ${opts.base}`);
    return opts;
}

export function isLocalUrl(url) {
    let u;
    try { u = new URL(url); } catch { return false; }
    if (['file:', 'data:', 'blob:', 'about:', 'chrome-error:', 'devtools:', 'chrome-extension:'].includes(u.protocol)) return true;
    if (!['http:', 'https:', 'ws:', 'wss:'].includes(u.protocol)) return false;
    return ['127.0.0.1', 'localhost', '[::1]'].includes(u.hostname);
}

const results = [];
function check(id, status, title, evidence = []) {
    results.push({ id, status, title, evidence: [].concat(evidence) });
    const mark = { pass: '✔', fail: '✘', warn: '!' }[status];
    console.log(`${mark} ${id.padEnd(10)} ${title}`);
    for (const e of [].concat(evidence)) console.log(`    ${e}`);
}
const expect = (id, ok, title, evidence) => check(id, ok ? 'pass' : 'fail', title, evidence);

/* ------------------------------------------------------------------------------------------------ *
 * Laravel: server + settings
 * ------------------------------------------------------------------------------------------------ */

function freePort() {
    return new Promise((res, rej) => {
        const srv = createServer();
        srv.unref();
        srv.on('error', rej);
        srv.listen(0, '127.0.0.1', () => {
            const { port } = srv.address();
            srv.close(() => res(port));
        });
    });
}

/**
 * `php -S` on a free port with Laravel's own router (what `php artisan serve` runs), wrapped by a throw-away router in
 * a temp dir: while the flag file holds a token, /sw.js is answered byte-changed (public/sw.js + a comment) — a "new
 * service worker deploy" for the update-channel check without touching the build or the repo.
 */
async function startServer() {
    const port = await freePort();
    const dir = mkdtempSync(join(tmpdir(), 'ritme-pwa-srv-'));
    const flag = join(dir, 'sw-flag');
    writeFileSync(flag, '');
    const laravelRouter = existsSync(join(ROOT, 'server.php')) ? join(ROOT, 'server.php') : join(ROOT, 'vendor/laravel/framework/src/Illuminate/Foundation/resources/server.php');
    const router = join(dir, 'router.php');
    writeFileSync(router, `<?php
$uri = urldecode(parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH) ?? '');
$token = trim((string) @file_get_contents(${JSON.stringify(flag)}));
if ($uri === '/sw.js' && $token !== '') {
    header('Content-Type: text/javascript; charset=utf-8');
    header('Cache-Control: no-cache');
    readfile(getcwd().'/sw.js');
    echo "\n/* pwa-check redeploy ".$token." */\n";
    return true;
}
return require ${JSON.stringify(laravelRouter)};
`);
    const proc = spawn('php', ['-S', `127.0.0.1:${port}`, router], {
        cwd: join(ROOT, 'public'), detached: true, stdio: ['ignore', 'ignore', 'pipe'], env: { ...process.env, PHP_CLI_SERVER_WORKERS: '4' },
    });
    let stderr = '';
    proc.stderr.on('data', (d) => { stderr = (stderr + d).slice(-4000); });
    const base = `http://127.0.0.1:${port}`;
    let up = false;
    for (let i = 0; i < 100 && !up; i++) {
        if (proc.exitCode !== null) throw new Error(`php -S exited: ${stderr.slice(-400)}`);
        try {
            const r = await fetch(`${base}/pwa/version.json`, { signal: AbortSignal.timeout(2000) });
            up = r.ok;
        } catch { /* not up yet */ }
        if (!up) await sleep(200);
    }
    if (!up) throw new Error(`site did not answer at ${base}: ${stderr.slice(-400)}`);
    const stop = async () => {
        if (proc.exitCode === null) {
            try { process.kill(-proc.pid, 'SIGTERM'); } catch { /* gone */ }
            for (let i = 0; i < 40 && proc.exitCode === null; i++) await sleep(50);
            if (proc.exitCode === null) { try { process.kill(-proc.pid, 'SIGKILL'); } catch { /* gone */ } }
        }
        rmSync(dir, { recursive: true, force: true });
    };
    const redeploySw = (token) => writeFileSync(flag, token ?? '');
    return { base, stop, redeploySw };
}

const PHP_SETTINGS = '$r = app(\\App\\Domain\\Settings\\Contracts\\SettingsRepository::class); $g = \\App\\Domain\\Settings\\Enums\\SettingGroup::Pwa;';

function tinker(code) {
    return execFileSync('php', ['artisan', 'tinker', '--execute', code], { cwd: ROOT, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim();
}

function readMinBuildId() {
    const out = tinker(`${PHP_SETTINGS} echo json_encode(['min' => $r->group($g)->toArray()['min_build_id'] ?? null]);`);
    return JSON.parse(out.slice(out.indexOf('{'))).min;
}

function writeMinBuildId(value) {
    const php = value === null ? 'null' : `'${String(value).replace(/[^0-9a-z-]/gi, '')}'`;
    tinker(`${PHP_SETTINGS} app(\\App\\Domain\\Settings\\Actions\\UpdateSettings::class)->handle($g, ['min_build_id' => ${php}]); echo 'ok';`);
}

/* ------------------------------------------------------------------------------------------------ *
 * Chrome + CDP (browser-level connection, flat sessions, auto-attach to pages AND service workers)
 * ------------------------------------------------------------------------------------------------ */

async function launchChrome() {
    const bin = process.env.CHROME_PATH || DEFAULT_CHROME;
    if (!existsSync(bin)) throw new Error(`Chrome not found at ${bin} (set CHROME_PATH)`);
    const profile = mkdtempSync(join(tmpdir(), 'ritme-pwa-'));
    const proc = spawn(bin, [
        '--headless=new', '--remote-debugging-port=0', `--user-data-dir=${profile}`,
        '--no-first-run', '--no-default-browser-check', '--disable-extensions', '--disable-sync',
        '--disable-background-networking', '--disable-component-update', '--disable-default-apps',
        '--disable-domain-reliability', '--metrics-recording-only', '--no-pings', '--mute-audio',
        '--hide-scrollbars', '--force-device-scale-factor=1', '--force-color-profile=srgb', 'about:blank',
    ], { stdio: ['ignore', 'ignore', 'pipe'] });
    let stderr = '';
    proc.stderr.on('data', (d) => { stderr += d; });
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

class Cdp {
    static connect(url) {
        return new Promise((res, rej) => {
            const ws = new WebSocket(url);
            ws.addEventListener('open', () => res(new Cdp(ws)));
            ws.addEventListener('error', () => rej(new Error(`CDP connect failed: ${url}`)));
        });
    }

    constructor(ws) {
        this.ws = ws; this.id = 0; this.pending = new Map(); this.handlers = [];
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

    on(fn) { this.handlers.push(fn); return () => { this.handlers = this.handlers.filter((h) => h !== fn); }; }

    close() { this.ws.close(); }
}

/** Injected into every page before its scripts: reports CSP violations and install-prompt events to the tool. */
const PAGE_PROBE = `(() => {
  document.addEventListener('securitypolicyviolation', (e) => {
    console.error('[pwa-check:csp] ' + e.violatedDirective + ' ' + (e.blockedURI || '') + ' ' + (e.sourceFile || '') + ':' + (e.lineNumber || 0));
  }, true);
  addEventListener('beforeinstallprompt', () => { try { sessionStorage.setItem('pwa-check:bip', '1'); } catch (e) {} });
})();`;

/**
 * One browser, many contexts. Every attached target (page or service worker) is instrumented before it runs:
 * Network (external requests, document responses), Runtime/Log (CSP reports, errors), optional offline emulation
 * and optional response rewriting (Fetch) per browser context.
 */
class Harness {
    constructor(cdp) {
        this.cdp = cdp;
        this.sessions = new Map(); // sessionId → { type, url, contextId, targetId }
        this.contexts = new Map(); // contextId → { offline, rewrite: (url, body) => body|null }
        this.external = new Set();
        this.csp = [];
        this.errors = [];
        this.docs = new Map(); // sessionId → last Document response
        this.paused = [];
        this.waiters = [];
        this.navEvents = new Map(); // sessionId → count of Page.loadEventFired
        cdp.on((msg) => this.onEvent(msg));
    }

    async init() {
        await this.cdp.send('Target.setAutoAttach', { autoAttach: true, waitForDebuggerOnStart: true, flatten: true });
        await this.cdp.send('Target.setDiscoverTargets', { discover: true });
    }

    ctx(contextId) {
        if (!this.contexts.has(contextId)) this.contexts.set(contextId, { offline: false, rewrite: null });
        return this.contexts.get(contextId);
    }

    onEvent(msg) {
        const p = msg.params ?? {};
        if (msg.method === 'Target.attachedToTarget') {
            this.attach(p).catch((e) => this.errors.push(`attach ${p.targetInfo?.type}: ${e.message}`));
            return;
        }
        if (msg.method === 'Target.detachedFromTarget') { this.sessions.delete(p.sessionId); return; }
        const sid = msg.sessionId;
        if (!sid) return;
        const info = this.sessions.get(sid);
        switch (msg.method) {
            case 'Network.requestWillBeSent':
                if (!isLocalUrl(p.request.url)) this.external.add(`${info?.type ?? '?'}: ${p.request.url}`);
                break;
            case 'Network.responseReceived':
                if (p.type === 'Document') {
                    this.docs.set(sid, { url: p.response.url, status: p.response.status, fromServiceWorker: !!p.response.fromServiceWorker });
                }
                break;
            case 'Runtime.consoleAPICalled': {
                const text = p.args.map((x) => x.value ?? x.description ?? '').join(' ');
                if (text.startsWith('[pwa-check:csp]')) this.csp.push(`${info?.url ?? ''} ${text.slice(16)}`);
                else if (p.type === 'error') this.errors.push(`${info?.type}: ${text}`);
                break;
            }
            case 'Log.entryAdded':
                if (/Content.Security.Policy/i.test(p.entry.text)) this.csp.push(`${p.entry.url ?? info?.url ?? ''} ${p.entry.text}`);
                break;
            case 'Runtime.exceptionThrown':
                this.errors.push(`${info?.type} uncaught: ${p.exceptionDetails.exception?.description ?? p.exceptionDetails.text}`);
                break;
            case 'Page.loadEventFired':
                this.navEvents.set(sid, (this.navEvents.get(sid) ?? 0) + 1);
                break;
            case 'Page.frameNavigated':
                if (!p.frame.parentId && info) info.url = p.frame.url;
                break;
            case 'Fetch.requestPaused':
                this.onPaused(sid, p).catch(() => {});
                break;
        }
        for (const w of [...this.waiters]) w(msg);
    }

    async attach(p) {
        const { sessionId, targetInfo } = p;
        const s = (m, params) => this.cdp.send(m, params, sessionId);
        const contextId = targetInfo.browserContextId;
        if (/^(chrome|chrome-extension|chrome-untrusted|devtools):/.test(targetInfo.url) || !['page', 'service_worker', 'worker', 'shared_worker'].includes(targetInfo.type)) {
            // Chrome's own UI / component extensions: not the site under test.
            if (p.waitingForDebugger) await s('Runtime.runIfWaitingForDebugger').catch(() => {});
            return;
        }
        this.sessions.set(sessionId, { type: targetInfo.type, url: targetInfo.url, contextId, targetId: targetInfo.targetId });
        const ctx = this.ctx(contextId);
        try {
            if (['page', 'service_worker', 'worker', 'shared_worker'].includes(targetInfo.type)) {
                await s('Network.enable').catch(() => {});
                await s('Runtime.enable').catch(() => {});
                await s('Log.enable').catch(() => {});
                if (ctx.offline) await this.applyOffline(sessionId, true);
                await this.applyFetch(sessionId, targetInfo.type, ctx);
            }
            if (targetInfo.type === 'page') {
                await s('Page.enable');
                await s('Page.addScriptToEvaluateOnNewDocument', { source: PAGE_PROBE });
                await s('Emulation.setDeviceMetricsOverride', { width: WIDTH, height: HEIGHT, deviceScaleFactor: 1, mobile: true });
                await s('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-reduced-motion', value: 'reduce' }, { name: 'prefers-color-scheme', value: 'light' }] });
            }
        } finally {
            if (p.waitingForDebugger) await s('Runtime.runIfWaitingForDebugger').catch(() => {});
        }
        for (const w of [...this.waiters]) w({ method: 'pwa-check.attached', params: p });
    }

    async applyOffline(sessionId, offline) {
        await this.cdp.send('Network.emulateNetworkConditions', {
            offline, latency: 0, downloadThroughput: -1, uploadThroughput: -1,
        }, sessionId).catch(() => {});
    }

    async applyFetch(sessionId, type, ctx) {
        const patterns = [];
        if (type === 'page' && ctx.rewrite) patterns.push({ urlPattern: '*/build/assets/*.js', requestStage: 'Response' });
        if (patterns.length) await this.cdp.send('Fetch.enable', { patterns }, sessionId).catch(() => {});
        else await this.cdp.send('Fetch.disable', {}, sessionId).catch(() => {});
    }

    async onPaused(sessionId, p) {
        const info = this.sessions.get(sessionId);
        const ctx = info ? this.ctx(info.contextId) : null;
        const s = (m, params) => this.cdp.send(m, params, sessionId);
        const url = p.request.url;
        const fn = ctx?.rewrite ?? null;
        if (!fn || p.responseStatusCode !== 200) { await s('Fetch.continueRequest', { requestId: p.requestId }); return; }
        const { body, base64Encoded } = await s('Fetch.getResponseBody', { requestId: p.requestId });
        const text = base64Encoded ? Buffer.from(body, 'base64').toString('utf8') : body;
        const next = fn(url, text);
        if (next === null || next === text) { await s('Fetch.continueRequest', { requestId: p.requestId }); return; }
        this.paused.push({ type: info?.type, url });
        await s('Fetch.fulfillRequest', {
            requestId: p.requestId,
            responseCode: 200,
            responseHeaders: (p.responseHeaders ?? []).filter((h) => !/^(content-length|content-encoding|etag)$/i.test(h.name)),
            body: Buffer.from(next, 'utf8').toString('base64'),
        });
    }

    async setOffline(contextId, offline) {
        this.ctx(contextId).offline = offline;
        await Promise.all([...this.sessions].filter(([, i]) => i.contextId === contextId).map(([sid]) => this.applyOffline(sid, offline)));
    }

    async refreshFetch(contextId) {
        const ctx = this.ctx(contextId);
        await Promise.all([...this.sessions].filter(([, i]) => i.contextId === contextId).map(([sid, i]) => this.applyFetch(sid, i.type, ctx)));
    }

    async newContext() {
        const { browserContextId } = await this.cdp.send('Target.createBrowserContext', { disposeOnDetach: true });
        this.ctx(browserContextId);
        return browserContextId;
    }

    async newPage(contextId) {
        let resolveAttach;
        const attached = new Promise((r) => { resolveAttach = r; });
        let targetId = null;
        const pendingMsgs = [];
        const waiter = (msg) => {
            if (msg.method !== 'pwa-check.attached') return;
            if (targetId === null) pendingMsgs.push(msg); else if (msg.params.targetInfo.targetId === targetId) resolveAttach(msg.params.sessionId);
        };
        this.waiters.push(waiter);
        ({ targetId } = await this.cdp.send('Target.createTarget', { url: 'about:blank', ...(contextId ? { browserContextId: contextId } : {}) }));
        for (const m of pendingMsgs) waiter(m);
        const sessionId = await Promise.race([attached, sleep(10000).then(() => null)]);
        this.waiters = this.waiters.filter((w) => w !== waiter);
        if (!sessionId) throw new Error('page target did not attach');
        return new PageHandle(this, sessionId, targetId);
    }

    async disposeContext(contextId) {
        await this.cdp.send('Target.disposeBrowserContext', { browserContextId: contextId }).catch(() => {});
    }
}

class PageHandle {
    constructor(h, sessionId, targetId) { this.h = h; this.sessionId = sessionId; this.targetId = targetId; }

    get contextId() { return this.h.sessions.get(this.sessionId)?.contextId; }

    send(m, p) { return this.h.cdp.send(m, p, this.sessionId); }

    async eval(expression, timeoutMs = 15000) {
        const r = await Promise.race([
            this.send('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true }),
            sleep(timeoutMs).then(() => ({ timeout: true })),
        ]);
        if (r.timeout) throw new Error(`evaluate timed out: ${expression.slice(0, 80)}`);
        if (r.exceptionDetails) throw new Error(`evaluate failed: ${r.exceptionDetails.exception?.description ?? r.exceptionDetails.text}`);
        return r.result.value;
    }

    /** Navigate and wait for load (or a navigation error). Returns { errorText, doc }. */
    async goto(url, settleMs = 600) {
        this.h.docs.delete(this.sessionId);
        const before = this.h.navEvents.get(this.sessionId) ?? 0;
        const nav = await this.send('Page.navigate', { url });
        if (nav.errorText) return { errorText: nav.errorText, doc: this.h.docs.get(this.sessionId) ?? null };
        await this.waitLoad(before);
        await sleep(settleMs);
        return { errorText: null, doc: this.h.docs.get(this.sessionId) ?? null };
    }

    async waitLoad(before, timeoutMs = 20000) {
        const start = Date.now();
        while (Date.now() - start < timeoutMs) {
            if ((this.h.navEvents.get(this.sessionId) ?? 0) > before) return true;
            await sleep(50);
        }
        return false;
    }

    loads() { return this.h.navEvents.get(this.sessionId) ?? 0; }

    async waitFor(expression, timeoutMs = 15000, stepMs = 200) {
        const start = Date.now();
        let last = null;
        while (Date.now() - start < timeoutMs) {
            try { last = await this.eval(expression, 5000); } catch { last = null; }
            if (last) return last;
            await sleep(stepMs);
        }
        return last;
    }

    async shot(file) {
        const { data } = await this.send('Page.captureScreenshot', { format: 'png' });
        writeFileSync(file, Buffer.from(data, 'base64'));
        return file;
    }

    async key(key, code, keyCode) {
        await this.send('Input.dispatchKeyEvent', { type: 'keyDown', key, code, windowsVirtualKeyCode: keyCode });
        await this.send('Input.dispatchKeyEvent', { type: 'keyUp', key, code, windowsVirtualKeyCode: keyCode });
    }

    close() { return this.h.cdp.send('Target.closeTarget', { targetId: this.targetId }).catch(() => {}); }
}

/* ------------------------------------------------------------------------------------------------ *
 * Page-side expressions
 * ------------------------------------------------------------------------------------------------ */

const JS = {
    controlled: `!!(navigator.serviceWorker && navigator.serviceWorker.controller)`,
    registration: `navigator.serviceWorker.getRegistration('/').then((r) => r ? {
        scope: r.scope, active: r.active && r.active.state, script: r.active && r.active.scriptURL,
        waiting: !!r.waiting, installing: !!r.installing, updateViaCache: r.updateViaCache } : null)`,
    cacheMap: `caches.keys().then((names) => Promise.all(names.map((n) => caches.open(n).then((c) => c.keys())
        .then((reqs) => [n, reqs.map((q) => { const u = new URL(q.url); return u.pathname + u.search; })])))).then(Object.fromEntries)`,
    isOffline: `!!document.querySelector('[data-pwa-retry]')`,
    title: `document.title`,
    h1: `(document.querySelector('h1') || {}).textContent?.trim() || ''`,
    soft: `(() => { const el = [...document.querySelectorAll('[role=status]')].find((n) => n.textContent.includes(${JSON.stringify(TEXT.soft)})); return el ? el.textContent.trim() : null; })()`,
    forced: `(() => { const el = document.querySelector('[role=alertdialog][aria-modal=true]'); return el && el.textContent.includes(${JSON.stringify(TEXT.forced)}) ? el.textContent.trim() : null; })()`,
    install: `(() => { const el = document.querySelector('[role=region][aria-label=${JSON.stringify(TEXT.install)}]'); return el ? el.textContent.trim() : null; })()`,
    bip: `sessionStorage.getItem('pwa-check:bip') === '1'`,
};

/* ------------------------------------------------------------------------------------------------ *
 * Runner
 * ------------------------------------------------------------------------------------------------ */

function readBuild() {
    const idFile = join(ROOT, 'public/build/build-id.json');
    const manifestFile = join(ROOT, 'public/build/manifest.json');
    const swFile = join(ROOT, 'public/sw.js');
    const out = { hot: existsSync(join(ROOT, 'public/hot')), buildId: null, swBuildId: null, expected: [] };
    if (existsSync(idFile)) out.buildId = JSON.parse(readFileSync(idFile, 'utf8')).build_id ?? null;
    if (existsSync(swFile)) out.swBuildId = /\(build ([0-9a-z-]+)\)/i.exec(readFileSync(swFile, 'utf8').slice(0, 400))?.[1] ?? null;
    if (existsSync(manifestFile)) {
        const iconDir = join(ROOT, 'public/icons');
        const icons = existsSync(iconDir) ? readdirSync(iconDir).filter((n) => /\.(png|svg|ico)$/i.test(n)) : [];
        out.expected = precacheFromManifest(JSON.parse(readFileSync(manifestFile, 'utf8')), icons);
    }
    return out;
}

async function httpChecks(base, build) {
    const get = (path, init) => fetch(base + path, { redirect: 'manual', signal: AbortSignal.timeout(15000), ...init });

    const m = await get('/manifest.webmanifest');
    const mType = m.headers.get('content-type') ?? '';
    let manifest = null;
    try { manifest = await m.json(); } catch { /* invalid */ }
    expect('http', m.status === 200 && /application\/manifest\+json/.test(mType) && manifest !== null,
        '/manifest.webmanifest answers 200 application/manifest+json with JSON', [`${m.status} ${mType}`]);

    const v = await get('/pwa/version.json');
    const vCc = v.headers.get('cache-control') ?? '';
    const version = await v.json().catch(() => null);
    expect('http', v.status === 200 && /no-store/.test(vCc) && !v.headers.get('set-cookie') && version?.build_id === build.buildId,
        '/pwa/version.json is no-store, cookie-free and reports the deployed build id',
        [`${v.status} cache-control: ${vCc}`, `body: ${JSON.stringify(version)}`]);

    const sw = await get('/sw.js');
    const swBody = await sw.text();
    const swType = sw.headers.get('content-type') ?? '';
    expect('http', sw.status === 200 && /javascript/.test(swType) && swBody.includes(build.buildId ?? '§'),
        '/sw.js is served as JavaScript and carries the deployed build id',
        [`${sw.status} ${swType}, ${(swBody.length / 1024).toFixed(1)} kB, cache-control: ${sw.headers.get('cache-control')}`]);

    const off = await get('/offline');
    const offBody = await off.text();
    expect('http', off.status === 200 && /<meta[^>]+name="robots"[^>]+noindex/i.test(offBody) && offBody.includes('data-pwa-retry'),
        '/offline answers 200, noindex, with the retry button', [`${off.status}, robots noindex: ${/noindex/.test(offBody)}`]);

    return { manifest, version };
}

function manifestMembers(manifest, base) {
    const problems = [];
    if (!manifest) return ['no manifest'];
    for (const k of ['name', 'short_name', 'start_url', 'display', 'theme_color', 'background_color', 'id']) {
        if (!manifest[k]) problems.push(`missing ${k}`);
    }
    if (!['standalone', 'fullscreen', 'minimal-ui'].includes(manifest.display)) problems.push(`display=${manifest.display}`);
    const icons = Array.isArray(manifest.icons) ? manifest.icons : [];
    const has = (size, purpose) => icons.some((i) => String(i.sizes ?? '').split(/\s+/).includes(size) && String(i.purpose ?? 'any').split(/\s+/).includes(purpose));
    if (!has('192x192', 'any')) problems.push('no 192x192 any icon');
    if (!has('512x512', 'any')) problems.push('no 512x512 any icon');
    if (!icons.some((i) => String(i.purpose ?? '').includes('maskable'))) problems.push('no maskable icon');
    for (const i of icons) {
        const u = new URL(i.src, base);
        if (!isLocalUrl(u.href)) problems.push(`icon off-origin: ${i.src}`);
    }
    return problems;
}

async function run(opts) {
    const out = resolve(ROOT, opts.out);
    mkdirSync(out, { recursive: true });
    const shots = [];
    const shot = async (page, name) => { shots.push(name); return page.shot(join(out, name)); };

    if (opts.build) {
        console.log('… npm run build');
        execFileSync('npm', ['run', 'build'], { cwd: ROOT, stdio: 'inherit' });
    }

    const build = readBuild();
    expect('build', !build.hot && !!build.buildId && build.buildId === build.swBuildId && build.expected.length > 0,
        'production build present: no public/hot, public/sw.js stamped with build-id.json',
        [`build-id.json ${build.buildId}, sw.js ${build.swBuildId}, hot: ${build.hot}, ${build.expected.length} precache URLs expected`]);
    if (results.some((r) => r.status === 'fail')) return { shots };

    const server = opts.base ? { base: opts.base, stop: async () => {} } : await startServer();
    const base = server.base;
    console.log(`… site ${base}${opts.base ? '' : ' (own php -S with the Laravel router)'}`);

    let chrome = null;
    let cdp = null;
    let originalMin;
    let minChanged = false;
    try {
        const { manifest } = await httpChecks(base, build);
        originalMin = readMinBuildId();
        if (originalMin !== null) check('forced', 'warn', `min_build_id was already set before the run: ${originalMin}`);

        chrome = await launchChrome();
        cdp = await Cdp.connect(chrome.wsUrl);
        const h = new Harness(cdp);
        await h.init();

        /* ---------------- main context: install, precache, offline, never-cache ---------------- */
        // The default (non-incognito) context: an incognito context reports `in-incognito` installability errors.
        const page = await h.newPage(null);
        const ctxA = page.contextId;
        await page.goto(`${base}/`);
        const controlled = await page.waitFor(JS.controlled, 20000);
        const reg = await page.eval(JS.registration).catch(() => null);
        expect('sw', !!controlled && reg?.active === 'activated' && reg?.scope === `${base}/`,
            'service worker registers (scope /), activates and controls the page',
            [`registration: ${JSON.stringify(reg)}`, `controller: ${!!controlled}`]);

        const appManifest = await page.send('Page.getAppManifest').catch((e) => ({ error: e.message }));
        const parsed = appManifest.data ? JSON.parse(appManifest.data) : null;
        const memberProblems = manifestMembers(parsed, base);
        expect('manifest', !appManifest.error && (appManifest.errors ?? []).length === 0 && memberProblems.length === 0,
            'Page.getAppManifest: no errors; name, short_name, id, start_url, standalone, colours, 192/512 + maskable icons',
            [`url: ${appManifest.url}`, `errors: ${JSON.stringify(appManifest.errors ?? appManifest.error)}`,
                `members: ${memberProblems.length ? memberProblems.join('; ') : 'ok'}`,
                parsed ? `name «${parsed.name}», start_url ${parsed.start_url}, display ${parsed.display}, ${parsed.icons?.length} icons` : '']);
        const inst = await page.send('Page.getInstallabilityErrors').catch((e) => ({ error: e.message }));
        expect('manifest', !inst.error && (inst.installabilityErrors ?? []).length === 0,
            'Page.getInstallabilityErrors is empty (installable)', [`${JSON.stringify(inst.installabilityErrors ?? inst.error)}`]);

        // Controlled navigations: / and the "visited" page land in the page cache.
        await page.goto(`${base}/`);
        await shot(page, 'home-390.png');
        const visitedOnline = await page.goto(`${base}${VISITED}`);
        const visitedTitle = await page.eval(JS.title);
        await sleep(800); // the worker stores the page in event.waitUntil

        // Online never-cache navigations through the controlled page.
        const never = {};
        for (const path of [ADMIN, CART]) {
            const r = await page.goto(`${base}${path}`);
            never[path] = r.doc;
        }
        const ver = await page.eval(`fetch('/pwa/version.json', {cache: 'no-store'}).then((r) => r.status)`);
        const caches1 = await page.eval(JS.cacheMap);
        const shellName = `ritme-shell-${build.buildId}`;
        const pagesName = `ritme-pages-${build.buildId}`;
        const shell = caches1[shellName] ?? [];
        const missing = build.expected.filter((u) => !shell.includes(u));
        const extra = shell.filter((u) => !build.expected.includes(u));
        const adminInShell = shell.filter((u) => /filament|\/admin|\/livewire/.test(u));
        expect('precache', missing.length === 0 && adminInShell.length === 0,
            `${shellName} holds every precache URL tools/build-sw.mjs derives (incl. /offline + icons), no admin assets`,
            [`caches: ${Object.keys(caches1).join(', ')}`, `shell ${shell.length} entries (expected ${build.expected.length}), missing ${missing.length}${missing.length ? `: ${missing.slice(0, 5).join(', ')}` : ''}, extra ${extra.length}${extra.length ? ` (runtime cache-first): ${extra.slice(0, 5).join(', ')}` : ''}`,
                `offline page precached: ${shell.includes('/offline')}`]);
        const pagesCache = caches1[pagesName] ?? [];
        expect('sw', pagesCache.includes('/') && pagesCache.includes(VISITED),
            `visited pages are stored in ${pagesName}`, [`${pagesName}: ${JSON.stringify(pagesCache)}`, `${VISITED} online: ${visitedOnline.doc?.status}, «${visitedTitle}»`]);
        const allCached = Object.values(caches1).flat();
        const leaked = allCached.filter((u) => [ADMIN, CART, '/pwa/version.json', '/sw.js', '/manifest.webmanifest'].some((p) => u.startsWith(p)) || u.startsWith('/livewire'));
        expect('never', leaked.length === 0 && Object.values(never).every((d) => d && !d.fromServiceWorker) && ver === 200,
            `never-cache routes (${ADMIN}, ${CART}, /pwa/version.json) bypass the worker and are in no ritme-* cache`,
            [...Object.entries(never).map(([p, d]) => `${p}: ${d?.status} fromServiceWorker=${d?.fromServiceWorker}`), `cached never-cache URLs: ${leaked.length ? leaked.join(', ') : 'none'}`]);

        /* ---------------- offline ---------------- */
        await h.setOffline(ctxA, true);
        const offVisited = await page.goto(`${base}${VISITED}`, 300);
        const offVisitedTitle = await page.eval(JS.title).catch(() => '');
        const offVisitedIsFallback = await page.eval(JS.isOffline).catch(() => true);
        await shot(page, 'offline-visited-390.png');
        expect('offline', !offVisited.errorText && offVisited.doc?.fromServiceWorker && offVisitedTitle === visitedTitle && !offVisitedIsFallback,
            `offline: visited ${VISITED} is served from the page cache`,
            [`navigation error: ${offVisited.errorText ?? 'none'}, status ${offVisited.doc?.status}, fromServiceWorker=${offVisited.doc?.fromServiceWorker}`, `title «${offVisitedTitle}» (online «${visitedTitle}»)`]);

        const offUnvisited = await page.goto(`${base}${UNVISITED}`, 300);
        const offUnvisitedIsFallback = await page.eval(JS.isOffline).catch(() => false);
        const offUnvisitedTitle = await page.eval(JS.title).catch(() => '');
        const retryVisible = await page.eval(`(() => { const b = document.querySelector('[data-pwa-retry]'); return !!b && !b.hidden; })()`).catch(() => false);
        await shot(page, 'offline-unvisited-390.png');
        expect('offline', !offUnvisited.errorText && offUnvisited.doc?.fromServiceWorker && offUnvisitedIsFallback,
            `offline: unvisited ${UNVISITED} falls back to the precached /offline page`,
            [`navigation error: ${offUnvisited.errorText ?? 'none'}, status ${offUnvisited.doc?.status}, fromServiceWorker=${offUnvisited.doc?.fromServiceWorker}`, `title «${offUnvisitedTitle}», retry button visible: ${retryVisible}`]);

        const offNever = {};
        for (const path of [ADMIN, CART]) {
            const r = await page.goto(`${base}${path}`, 300);
            const fallback = await page.eval(JS.isOffline).catch(() => false);
            offNever[path] = { error: r.errorText, sw: r.doc?.fromServiceWorker ?? false, fallback };
        }
        expect('offline', Object.values(offNever).every((r) => !!r.error && !r.sw && !r.fallback),
            `offline: never-cache routes (${ADMIN}, ${CART}) are not answered by the worker (browser network error, no cached copy, no /offline)`,
            Object.entries(offNever).map(([p, r]) => `${p}: ${r.error ?? 'loaded'}, fromServiceWorker=${r.sw}, offline page=${r.fallback}`));
        await h.setOffline(ctxA, false);

        /* ---------------- install prompt (views ≥ 2, 4 s delay) ---------------- */
        await page.goto(`${base}/`);
        const bannerText = await page.waitFor(JS.install, 9000, 300);
        const bip = await page.eval(JS.bip).catch(() => false);
        if (bannerText) {
            await shot(page, 'install-banner-390.png');
            check('install', 'pass', 'beforeinstallprompt → own install banner «ریتمی را روی صفحه اصلی داشته باش» on a later page view', [bannerText.slice(0, 120)]);
        } else {
            check('install', bip ? 'fail' : 'warn', bip ? 'beforeinstallprompt fired but no install banner appeared'
                : 'beforeinstallprompt did not fire in headless Chrome (banner not exercisable here; installability errors are empty)', [`beforeinstallprompt: ${bip}`]);
        }

        /* ---------------- soft update, service-worker channel (byte-changed sw.js) ---------------- */
        if (server.redeploySw) {
            server.redeploySw(String(Date.now()));
            const upd = await page.eval(`navigator.serviceWorker.getRegistration('/').then((r) => r.update()).then(() => true, (e) => String(e))`);
            const swSoft = await page.waitFor(JS.soft, 20000, 300);
            const regAfter = await page.eval(JS.registration).catch(() => null);
            if (swSoft) await shot(page, 'soft-sw-channel-390.png');
            expect('soft', !!swSoft && !!regAfter?.waiting,
                'service-worker channel: a redeployed (byte-changed) sw.js installs, waits for SKIP_WAITING, and the toast appears',
                [`registration.update(): ${upd}`, `waiting worker: ${regAfter?.waiting}`, `toast: ${swSoft ? 'shown' : 'missing'}`]);
            if (swSoft) {
                const loadsBefore = page.loads();
                await page.eval(`(() => { const t = [...document.querySelectorAll('[role=status]')].find((n) => n.textContent.includes(${JSON.stringify(TEXT.soft)})); const b = t && t.querySelector('button'); b && b.click(); return !!b; })()`);
                const reloaded = await page.waitLoad(loadsBefore, 10000);
                await sleep(1500);
                const extraLoads = page.loads() - loadsBefore - 1;
                const regApplied = await page.eval(JS.registration).catch(() => null);
                const stillToast = await page.eval(JS.soft).catch(() => null);
                const controlledAfter = await page.eval(JS.controlled).catch(() => false);
                expect('soft', reloaded && extraLoads === 0 && !regApplied?.waiting && regApplied?.active === 'activated' && controlledAfter && !stillToast,
                    'toast «به‌روزرسانی» activates the waiting worker (SKIP_WAITING) and reloads exactly once',
                    [`reloaded: ${reloaded}, extra reloads: ${extraLoads}, waiting after: ${regApplied?.waiting}, active: ${regApplied?.active}, toast after reload: ${!!stillToast}`]);
            }
            server.redeploySw('');
        } else {
            check('soft', 'warn', 'service-worker channel skipped: needs the tool\'s own server (run without --base)', []);
        }
        await page.close();

        /* ---------------- soft update, polling channel (newer deployed build id) ---------------- */
        const rewriteOld = (url, body) => (body.includes(build.buildId) ? body.split(build.buildId).join(FAKE_OLD_BUILD) : null);
        const ctxB = await h.newContext();
        h.ctx(ctxB).rewrite = rewriteOld;
        const pageB = await h.newPage(ctxB);
        await pageB.goto(`${base}${VISITED}`);
        const softText = await pageB.waitFor(JS.soft, 15000, 300);
        const rewritten = h.paused.filter((p) => !/sw\.js/.test(p.url)).map((p) => p.url.replace(base, ''));
        if (softText) await shot(pageB, 'soft-toast-390.png');
        expect('soft', !!softText && rewritten.length > 0,
            `polling channel: running build ${FAKE_OLD_BUILD} < deployed ${build.buildId} → toast «${TEXT.soft}» (non-blocking)`,
            [`bundle rewritten on the wire: ${rewritten.slice(0, 3).join(', ') || 'none'}`, `toast: ${softText ? softText.slice(0, 100) : 'missing'}`]);
        const softDismiss = softText ? await pageB.eval(`(() => { const t = [...document.querySelectorAll('[role=status]')].find((n) => n.textContent.includes(${JSON.stringify(TEXT.soft)})); const b = t && t.querySelectorAll('button')[1]; b && b.click(); return !!b; })()`) : false;
        const softGone = softDismiss ? !(await pageB.eval(JS.soft)) : false;
        const pageInteractive = await pageB.eval(`![...document.body.children].some((c) => c.hasAttribute('inert'))`);
        expect('soft', softGone && pageInteractive, 'soft toast is dismissible («بعداً») and never blocks the page',
            [`dismissed: ${softGone}, page inert: ${!pageInteractive}`]);
        await pageB.close();
        await h.disposeContext(ctxB);

        /* ---------------- forced update (admin min_build_id raised) ---------------- */
        writeMinBuildId(build.buildId);
        minChanged = true;
        const vForced = await (await fetch(`${base}/pwa/version.json`, { cache: 'no-store' })).json();
        // An old client: a page whose worker is already installed (so no first-install auto-apply) keeps running an
        // older bundle. Network.setBypassServiceWorker lets the page fetch the bundle from the network, where its build
        // id is rewritten to an old one; the real /pwa/version.json then carries the raised minimum.
        const ctxC = await h.newContext();
        const pageC = await h.newPage(ctxC);
        await pageC.goto(`${base}/`);
        await pageC.waitFor(JS.controlled, 20000);
        await pageC.send('Network.setBypassServiceWorker', { bypass: true });
        h.ctx(ctxC).rewrite = rewriteOld;
        await h.refreshFetch(ctxC);
        await pageC.goto(`${base}${VISITED}`);
        const forcedText = await pageC.waitFor(JS.forced, 15000, 300);
        let forcedEvidence = [`/pwa/version.json after bump: ${JSON.stringify(vForced)}`, `screen: ${forcedText ? forcedText.slice(0, 90) : 'missing'}`];
        let forcedOk = !!forcedText && vForced.min_build_id === build.buildId;
        if (forcedText) {
            await shot(pageC, 'forced-390.png');
            await pageC.key('Escape', 'Escape', 27);
            await sleep(300);
            const afterEsc = await pageC.eval(JS.forced);
            await pageC.eval('history.back(); true');
            await sleep(800);
            const afterBack = await pageC.eval(JS.forced).catch(() => null);
            const inert = await pageC.eval(`[...document.body.children].filter((c) => c.getAttribute('role') !== 'alertdialog').every((c) => c.hasAttribute('inert'))`).catch(() => false);
            const focus = await pageC.eval(`!!document.activeElement && !!document.activeElement.closest('[role=alertdialog]')`).catch(() => false);
            // The button updates/reloads; the bundle is still old (rewritten), so the screen must come back.
            const loadsBefore = pageC.loads();
            await pageC.eval(`(() => { const b = document.querySelector('[role=alertdialog] button'); b && b.click(); return !!b; })()`);
            const reloaded = await pageC.waitLoad(loadsBefore, 15000);
            const again = reloaded ? await pageC.waitFor(JS.forced, 15000, 300) : null;
            forcedOk = forcedOk && !!afterEsc && !!afterBack && inert && focus && reloaded && !!again;
            forcedEvidence = forcedEvidence.concat([
                `after Escape: ${afterEsc ? 'still shown' : 'GONE'}, after Back: ${afterBack ? 'still shown' : 'GONE'}`,
                `rest of page inert: ${inert}, focus inside screen: ${focus}`,
                `«به‌روزرسانی» reloaded: ${reloaded}; still-old build after reload → screen again: ${!!again}`]);
        }
        expect('forced', forcedOk, `min_build_id = deployed build → blocking screen «${TEXT.forced}» for build ${FAKE_OLD_BUILD}; Escape/Back cannot leave`, forcedEvidence);
        await pageC.close();
        await h.disposeContext(ctxC);

        /* ---------------- run-wide ---------------- */
        const buildAfter = readBuild().buildId;
        if (buildAfter !== build.buildId) {
            check('build', 'fail', 'the build changed during the run (concurrent `npm run build`) — results above are unreliable, rerun',
                [`start ${build.buildId}, end ${buildAfter}`]);
        }
        expect('external', h.external.size === 0, 'no request to a non-local origin from any page or service worker',
            h.external.size ? [...h.external].slice(0, 10) : ['0 external requests']);
        expect('csp', h.csp.length === 0, 'no Content-Security-Policy violation on any page', h.csp.length ? h.csp.slice(0, 10) : ['0 violations']);
        const relevantErrors = h.errors.filter((e) => !/ERR_INTERNET_DISCONNECTED|Failed to load resource|net::ERR_/.test(e));
        if (relevantErrors.length) check('console', 'warn', 'console errors / exceptions seen (offline network errors excluded)', relevantErrors.slice(0, 10));
    } finally {
        if (minChanged && !opts.keepMin) {
            try {
                writeMinBuildId(originalMin ?? null);
                const back = readMinBuildId();
                expect('forced', back === (originalMin ?? null), `min_build_id restored to ${JSON.stringify(originalMin ?? null)}`, [`now: ${JSON.stringify(back)}`]);
            } catch (e) {
                check('forced', 'fail', `could not restore min_build_id (${JSON.stringify(originalMin)}) — restore it in the admin`, [e.message]);
            }
        }
        if (cdp) cdp.close();
        if (chrome) await chrome.close();
        await server.stop();
    }
    return { shots, base, buildId: build.buildId };
}

const HELP = `pwa-check — PWA verification of the production build (L8-03)

Usage: node tools/pwa-check.mjs [--build] [--base <local url>] [--out <dir>] [--keep-min]

  --build      run \`npm run build\` first
  --base       use a running local site instead of starting one (the worker-channel check is then skipped)
  --out        screenshots + pwa-check.json, default docs/qa/L8
  --keep-min   leave min_build_id raised after the forced-update check (default: restore)
`;

async function main(argv) {
    const opts = parseArgs(argv);
    if (opts.help) { console.log(HELP); return 0; }
    let meta = {};
    try {
        meta = await run(opts);
    } catch (e) {
        check('tool', 'fail', `tool error: ${e.message}`, [e.stack?.split('\n').slice(1, 3).join(' | ') ?? '']);
    }
    const counts = { pass: 0, fail: 0, warn: 0 };
    for (const r of results) counts[r.status]++;
    const report = { tool: 'pwa-check', date: new Date().toISOString(), build: meta.buildId ?? null, counts, results, screenshots: meta.shots ?? [] };
    writeFileSync(resolve(ROOT, opts.out, 'pwa-check.json'), `${JSON.stringify(report, null, 2)}\n`);
    console.log(`\n${counts.pass} passed, ${counts.fail} failed, ${counts.warn} warnings — ${opts.out}/pwa-check.json`);
    return counts.fail ? 1 : 0;
}

main(process.argv.slice(2)).then((code) => process.exit(code), (e) => { console.error(e); process.exit(1); });
