#!/usr/bin/env node
/**
 * shot — fidelity screenshots: design export page vs Laravel route, per width, with a pixel diff.
 *
 *   node tools/shot.mjs --design cycle.html --route /cycle [--widths 390,768,1440] [--out docs/qa/L3-03]
 *   node tools/shot.mjs --all [--out docs/qa/L3-11]
 *
 * - Drives headless Chrome over the DevTools protocol with Node's built-in WebSocket (Node ≥ 22). No npm deps.
 *   Chrome: /Applications/Google Chrome.app (override with CHROME_PATH).
 * - Full-page PNGs of design/html/<file> (file://) and <base><route> (default base http://127.0.0.1:8000).
 * - Per width: % of differing pixels (over the larger of the two canvases) and a diff PNG
 *   (faded design, differing pixels red, area only one page covers magenta). Pure-JS PNG decode/encode (zlib).
 * - Records console errors, page exceptions, failed requests and HTTP ≥ 400 responses.
 * - Every request to a non-local origin is BLOCKED (Fetch domain) and FAILS the run (exit 1): "no externals".
 *   Local = file:, data:, blob:, about:, http(s)://127.0.0.1 | localhost | [::1].
 * - Exit codes: 0 ok, 1 external request / page error / tool error, 2 diff above --threshold with --strict.
 */
import { spawn } from 'node:child_process';
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { deflateSync, inflateSync } from 'node:zlib';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const DEFAULT_CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const MAX_HEIGHT = 20000;

const HELP = `shot — design HTML vs Laravel route screenshots + pixel diff (L0-08)

Usage:
  node tools/shot.mjs --design <file.html> --route </path> [options]
  node tools/shot.mjs --all [options]

Options:
  --design <file>     page in design/html/ (e.g. cycle.html)
  --route <path>      Laravel path (e.g. /cycle), requested at --base
  --all               every page of the docs/AUDIT.md §7 urlmap whose route answers 2xx (others skipped)
  --only <list>       with --all: comma list of design files, route names or owner task ids (e.g. L3-05)
  --widths <list>     viewport widths, default 390,1440
  --out <dir>         output dir, default docs/qa/adhoc (PNGs are git-ignored)
  --base <url>        local origin of the Laravel app, default http://127.0.0.1:8000
  --threshold <pct>   target max diff per width, default 3
  --strict            exit 2 when any width is above --threshold
  --json <file>       also write the machine-readable report to <file>
  --help              this text

Env: CHROME_PATH overrides ${DEFAULT_CHROME}

Outputs per page and width: <slug>-<w>-design.png, <slug>-<w>-route.png, <slug>-<w>-diff.png
Any request to a non-local origin is blocked and fails the run (exit 1).
`;

/* ------------------------------------------------------------------------------------------------ *
 * Arguments
 * ------------------------------------------------------------------------------------------------ */

export function parseArgs(argv) {
    const opts = { widths: [390, 1440], out: 'docs/qa/adhoc', base: 'http://127.0.0.1:8000', threshold: 3 };
    for (let i = 0; i < argv.length; i++) {
        const a = argv[i];
        const val = () => {
            const v = argv[++i];
            if (v === undefined) throw new Error(`${a} needs a value`);
            return v;
        };
        switch (a) {
            case '--help': case '-h': opts.help = true; break;
            case '--design': opts.design = val(); break;
            case '--route': opts.route = val(); break;
            case '--all': opts.all = true; break;
            case '--only': opts.only = val().split(',').map((s) => s.trim()).filter(Boolean); break;
            case '--widths':
                opts.widths = val().split(',').map((s) => Number.parseInt(s, 10));
                if (opts.widths.some((w) => !Number.isFinite(w) || w < 200 || w > 4000)) throw new Error('bad --widths');
                break;
            case '--out': opts.out = val(); break;
            case '--base': opts.base = val().replace(/\/+$/, ''); break;
            case '--threshold': opts.threshold = Number.parseFloat(val()); break;
            case '--strict': opts.strict = true; break;
            case '--json': opts.json = val(); break;
            default: throw new Error(`unknown argument ${a}`);
        }
    }
    if (!opts.help) {
        if (!opts.all && (!opts.design || !opts.route)) throw new Error('need --design and --route, or --all');
        if (!isLocalUrl(opts.base) || !/^https?:/.test(opts.base)) throw new Error(`--base must be a local http origin: ${opts.base}`);
    }
    return opts;
}

/* ------------------------------------------------------------------------------------------------ *
 * Local-origin rule
 * ------------------------------------------------------------------------------------------------ */

export function isLocalUrl(url) {
    let u;
    try { u = new URL(url); } catch { return false; }
    if (['file:', 'data:', 'blob:', 'about:'].includes(u.protocol)) return true;
    if (u.protocol === 'chrome-error:' || u.protocol === 'devtools:') return true;
    if (!['http:', 'https:', 'ws:', 'wss:'].includes(u.protocol)) return false;
    return ['127.0.0.1', 'localhost', '[::1]'].includes(u.hostname);
}

/* ------------------------------------------------------------------------------------------------ *
 * URL map (docs/AUDIT.md §7)
 * ------------------------------------------------------------------------------------------------ */

export function readUrlMap(file = join(ROOT, 'docs/AUDIT.md')) {
    const md = readFileSync(file, 'utf8');
    const m = md.match(/```json urlmap\n([\s\S]*?)```/);
    if (!m) throw new Error('no ```json urlmap block in docs/AUDIT.md');
    return JSON.parse(m[1]);
}

/* ------------------------------------------------------------------------------------------------ *
 * PNG (8-bit RGB/RGBA/grey, non-interlaced — what Chrome emits) decode + RGBA encode
 * ------------------------------------------------------------------------------------------------ */

const CRC_TABLE = (() => {
    const t = new Uint32Array(256);
    for (let n = 0; n < 256; n++) {
        let c = n;
        for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
        t[n] = c >>> 0;
    }
    return t;
})();

function crc32(buf) {
    let c = 0xffffffff;
    for (let i = 0; i < buf.length; i++) c = CRC_TABLE[(c ^ buf[i]) & 0xff] ^ (c >>> 8);
    return (c ^ 0xffffffff) >>> 0;
}

export function decodePng(buf) {
    if (buf.readUInt32BE(0) !== 0x89504e47) throw new Error('not a PNG');
    let pos = 8;
    let width = 0, height = 0, bitDepth = 0, colorType = 0, interlace = 0, palette = null, trns = null;
    const idat = [];
    while (pos < buf.length) {
        const len = buf.readUInt32BE(pos);
        const type = buf.toString('latin1', pos + 4, pos + 8);
        const data = buf.subarray(pos + 8, pos + 8 + len);
        if (type === 'IHDR') {
            width = data.readUInt32BE(0); height = data.readUInt32BE(4);
            bitDepth = data[8]; colorType = data[9]; interlace = data[12];
        } else if (type === 'PLTE') palette = data;
        else if (type === 'tRNS') trns = data;
        else if (type === 'IDAT') idat.push(data);
        else if (type === 'IEND') break;
        pos += 12 + len;
    }
    if (bitDepth !== 8 || interlace !== 0) throw new Error(`unsupported PNG (depth ${bitDepth}, interlace ${interlace})`);
    const channels = { 0: 1, 2: 3, 3: 1, 4: 2, 6: 4 }[colorType];
    if (!channels) throw new Error(`unsupported PNG colour type ${colorType}`);
    const raw = inflateSync(Buffer.concat(idat));
    const stride = width * channels;
    const px = Buffer.alloc(stride * height);
    let prev = Buffer.alloc(stride);
    for (let y = 0; y < height; y++) {
        const f = raw[y * (stride + 1)];
        const line = raw.subarray(y * (stride + 1) + 1, (y + 1) * (stride + 1));
        const out = px.subarray(y * stride, (y + 1) * stride);
        for (let x = 0; x < stride; x++) {
            const a = x >= channels ? out[x - channels] : 0;
            const b = prev[x];
            const c = x >= channels ? prev[x - channels] : 0;
            let v = line[x];
            if (f === 1) v += a;
            else if (f === 2) v += b;
            else if (f === 3) v += (a + b) >> 1;
            else if (f === 4) {
                const p = a + b - c, pa = Math.abs(p - a), pb = Math.abs(p - b), pc = Math.abs(p - c);
                v += pa <= pb && pa <= pc ? a : pb <= pc ? b : c;
            }
            out[x] = v & 0xff;
        }
        prev = out;
    }
    const rgba = Buffer.alloc(width * height * 4);
    for (let i = 0, j = 0; i < width * height; i++, j += channels) {
        const o = i * 4;
        if (colorType === 6) { rgba[o] = px[j]; rgba[o + 1] = px[j + 1]; rgba[o + 2] = px[j + 2]; rgba[o + 3] = px[j + 3]; }
        else if (colorType === 2) { rgba[o] = px[j]; rgba[o + 1] = px[j + 1]; rgba[o + 2] = px[j + 2]; rgba[o + 3] = 255; }
        else if (colorType === 0) { rgba[o] = rgba[o + 1] = rgba[o + 2] = px[j]; rgba[o + 3] = 255; }
        else if (colorType === 4) { rgba[o] = rgba[o + 1] = rgba[o + 2] = px[j]; rgba[o + 3] = px[j + 1]; }
        else {
            const k = px[j];
            rgba[o] = palette[k * 3]; rgba[o + 1] = palette[k * 3 + 1]; rgba[o + 2] = palette[k * 3 + 2];
            rgba[o + 3] = trns && k < trns.length ? trns[k] : 255;
        }
    }
    return { width, height, data: rgba };
}

export function encodePng({ width, height, data }) {
    const stride = width * 4;
    const raw = Buffer.alloc((stride + 1) * height);
    for (let y = 0; y < height; y++) data.copy(raw, y * (stride + 1) + 1, y * stride, (y + 1) * stride);
    const chunk = (type, body) => {
        const len = Buffer.alloc(4); len.writeUInt32BE(body.length);
        const tb = Buffer.concat([Buffer.from(type, 'latin1'), body]);
        const crc = Buffer.alloc(4); crc.writeUInt32BE(crc32(tb));
        return Buffer.concat([len, tb, crc]);
    };
    const ihdr = Buffer.alloc(13);
    ihdr.writeUInt32BE(width, 0); ihdr.writeUInt32BE(height, 4);
    ihdr[8] = 8; ihdr[9] = 6; ihdr[10] = 0; ihdr[11] = 0; ihdr[12] = 0;
    return Buffer.concat([
        Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
        chunk('IHDR', ihdr), chunk('IDAT', deflateSync(raw, { level: 6 })), chunk('IEND', Buffer.alloc(0)),
    ]);
}

/* ------------------------------------------------------------------------------------------------ *
 * Diff: a pixel differs when any channel (over white) moves by more than `tolerance` (anti-aliasing noise).
 * Percentage is over max(w) × max(h); area covered by only one image counts as different.
 * ------------------------------------------------------------------------------------------------ */

export function diffImages(a, b, tolerance = 32) {
    const width = Math.max(a.width, b.width);
    const height = Math.max(a.height, b.height);
    const out = Buffer.alloc(width * height * 4);
    let diff = 0;
    const flat = (img, x, y, c) => {
        const o = (y * img.width + x) * 4;
        const al = img.data[o + 3] / 255;
        return img.data[o + c] * al + 255 * (1 - al);
    };
    for (let y = 0; y < height; y++) {
        for (let x = 0; x < width; x++) {
            const o = (y * width + x) * 4;
            const inA = x < a.width && y < a.height;
            const inB = x < b.width && y < b.height;
            if (!inA || !inB) {
                diff++;
                out[o] = 255; out[o + 1] = 0; out[o + 2] = 255; out[o + 3] = 255;
                continue;
            }
            let delta = 0, lum = 0;
            for (let c = 0; c < 3; c++) {
                const va = flat(a, x, y, c);
                delta = Math.max(delta, Math.abs(va - flat(b, x, y, c)));
                lum += va;
            }
            if (delta > tolerance) {
                diff++;
                out[o] = 230; out[o + 1] = 0; out[o + 2] = 40; out[o + 3] = 255;
            } else {
                const g = 255 - Math.round((255 - lum / 3) * 0.25);
                out[o] = out[o + 1] = out[o + 2] = g; out[o + 3] = 255;
            }
        }
    }
    return { width, height, data: out, diffPixels: diff, percent: width * height ? (diff / (width * height)) * 100 : 0 };
}

/* ------------------------------------------------------------------------------------------------ *
 * Chrome + CDP
 * ------------------------------------------------------------------------------------------------ */

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function launchChrome() {
    const bin = process.env.CHROME_PATH || DEFAULT_CHROME;
    if (!existsSync(bin)) throw new Error(`Chrome not found at ${bin} (set CHROME_PATH)`);
    const profile = mkdtempSync(join(tmpdir(), 'ritme-shot-'));
    const proc = spawn(bin, [
        '--headless=new', '--remote-debugging-port=0', `--user-data-dir=${profile}`,
        '--no-first-run', '--no-default-browser-check', '--disable-extensions', '--disable-sync',
        '--disable-background-networking', '--disable-component-update', '--disable-default-apps',
        '--disable-domain-reliability', '--metrics-recording-only', '--no-pings', '--mute-audio',
        '--hide-scrollbars', '--force-device-scale-factor=1', '--force-color-profile=srgb',
        '--font-render-hinting=none', '--allow-file-access-from-files', 'about:blank',
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

/** Load `url` at `width`, return { png, log }. Non-local requests are blocked and logged. */
async function capture(cdp, url, width) {
    const { targetId } = await cdp.send('Target.createTarget', { url: 'about:blank' });
    const { sessionId } = await cdp.send('Target.attachToTarget', { targetId, flatten: true });
    const s = (m, p) => cdp.send(m, p, sessionId);
    const log = { url, external: [], failed: [], console: [], status: null };
    let inflight = 0, lastActivity = Date.now(), loaded = false;
    const reqs = new Map();

    const off = cdp.on((msg) => {
        if (msg.sessionId !== sessionId) return;
        const p = msg.params;
        switch (msg.method) {
            case 'Fetch.requestPaused': {
                if (isLocalUrl(p.request.url)) s('Fetch.continueRequest', { requestId: p.requestId }).catch(() => {});
                else {
                    if (!log.external.includes(p.request.url)) log.external.push(p.request.url);
                    s('Fetch.failRequest', { requestId: p.requestId, errorReason: 'BlockedByClient' }).catch(() => {});
                }
                break;
            }
            case 'Network.requestWillBeSent':
                if (!isLocalUrl(p.request.url) && !log.external.includes(p.request.url)) log.external.push(p.request.url);
                if (!p.request.url.startsWith('data:')) { reqs.set(p.requestId, p.request.url); inflight++; }
                lastActivity = Date.now();
                break;
            case 'Network.responseReceived':
                if (p.type === 'Document' && p.response.url === url) log.status = p.response.status;
                if (p.response.status >= 400) log.failed.push(`${p.response.status} ${p.response.url}`);
                break;
            case 'Network.loadingFinished':
            case 'Network.loadingFailed':
                if (reqs.has(p.requestId)) {
                    if (msg.method === 'Network.loadingFailed' && !p.errorText.startsWith('net::ERR_BLOCKED_BY_CLIENT') && !p.canceled) {
                        log.failed.push(`${p.errorText} ${reqs.get(p.requestId)}`);
                    }
                    reqs.delete(p.requestId); inflight = Math.max(0, inflight - 1);
                }
                lastActivity = Date.now();
                break;
            case 'Runtime.consoleAPICalled':
                if (p.type === 'error' || p.type === 'assert') {
                    log.console.push(p.args.map((x) => x.value ?? x.description ?? '').join(' '));
                }
                break;
            case 'Runtime.exceptionThrown':
                log.console.push(`uncaught: ${p.exceptionDetails.exception?.description ?? p.exceptionDetails.text}`);
                break;
            case 'Log.entryAdded':
                if (p.entry.level === 'error' && !/ERR_BLOCKED_BY_CLIENT/.test(p.entry.text)) {
                    log.console.push(`${p.entry.source}: ${p.entry.text}${p.entry.url ? ` (${p.entry.url})` : ''}`);
                }
                break;
            case 'Page.loadEventFired': loaded = true; break;
        }
    });

    const idle = async (quietMs = 500, maxMs = 15000) => {
        const start = Date.now();
        while (Date.now() - start < maxMs) {
            if (loaded && inflight === 0 && Date.now() - lastActivity >= quietMs) return;
            await sleep(100);
        }
    };

    try {
        await s('Fetch.enable', { patterns: [{ urlPattern: '*' }] });
        await Promise.all([s('Network.enable'), s('Page.enable'), s('Runtime.enable'), s('Log.enable')]);
        await s('Network.setCacheDisabled', { cacheDisabled: true });
        // The site's strict CSP (L1-07) would block the tool's injected animation-off <style>; CSP is not under test here.
        await s('Page.setBypassCSP', { enabled: true });
        await s('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-reduced-motion', value: 'reduce' }, { name: 'prefers-color-scheme', value: 'light' }] });
        const mobile = width < 768;
        await s('Emulation.setDeviceMetricsOverride', { width, height: mobile ? 844 : 900, deviceScaleFactor: 1, mobile });
        await s('Page.addScriptToEvaluateOnNewDocument', {
            source: `addEventListener('DOMContentLoaded',()=>{const st=document.createElement('style');st.textContent='*,*::before,*::after{animation:none!important;transition:none!important;caret-color:transparent!important}';document.head.appendChild(st)})`,
        });
        const nav = await s('Page.navigate', { url });
        if (nav.errorText) throw new Error(`${nav.errorText} for ${url}`);
        await idle();
        await s('Runtime.evaluate', { expression: 'document.fonts ? document.fonts.ready.then(()=>1) : 1', awaitPromise: true });
        // Grow the viewport to the full page so lazy images load and nothing depends on scroll position.
        const measure = async () => (await s('Runtime.evaluate', {
            expression: 'Math.max(document.documentElement.scrollHeight, document.body ? document.body.scrollHeight : 0)', returnByValue: true,
        })).result.value;
        let height = Math.min(await measure(), MAX_HEIGHT);
        await s('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile });
        lastActivity = Date.now();
        await idle(400);
        height = Math.min(await measure(), MAX_HEIGHT);
        await s('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile });
        await sleep(150);
        const { data } = await s('Page.captureScreenshot', {
            format: 'png', captureBeyondViewport: true, clip: { x: 0, y: 0, width, height, scale: 1 },
        });
        return { png: Buffer.from(data, 'base64'), log };
    } finally {
        off();
        await cdp.send('Target.closeTarget', { targetId }).catch(() => {});
    }
}

/* ------------------------------------------------------------------------------------------------ *
 * Runner
 * ------------------------------------------------------------------------------------------------ */

const slugOf = (design) => design.replace(/^.*\//, '').replace(/\.html$/, '').replace(/[^a-z0-9-]+/gi, '-');

async function routeAnswers(url) {
    try {
        const res = await fetch(url, { redirect: 'manual', signal: AbortSignal.timeout(5000) });
        return res.status >= 200 && res.status < 300;
    } catch { return false; }
}

async function main(argv) {
    let opts;
    try { opts = parseArgs(argv); } catch (e) { process.stderr.write(`shot: ${e.message}\n\n${HELP}`); return 1; }
    if (opts.help) { process.stdout.write(HELP); return 0; }

    let jobs;
    if (opts.all) {
        jobs = readUrlMap().filter((e) => !opts.only || opts.only.some((o) => [e.design, e.name, e.owner].includes(o)));
    } else {
        jobs = [{ design: opts.design.replace(/^design\/html\//, ''), route: opts.route }];
    }

    const outDir = resolve(ROOT, opts.out);
    mkdirSync(outDir, { recursive: true });
    const report = { base: opts.base, widths: opts.widths, threshold: opts.threshold, pages: [], skipped: [] };
    let hardFail = false, above = false;

    const chrome = await launchChrome();
    const cdp = await Cdp.connect(chrome.wsUrl);
    try {
        for (const job of jobs) {
            // A path to an existing file is used as is (handy for ad-hoc checks); otherwise design/html/<file>.
            const designFile = job.design.includes('/') && existsSync(resolve(job.design)) ? resolve(job.design) : join(ROOT, 'design/html', job.design);
            const routeUrl = opts.base + (job.route.startsWith('/') ? job.route : `/${job.route}`);
            if (!existsSync(designFile)) { report.skipped.push({ ...job, reason: 'design file missing' }); hardFail ||= !opts.all; continue; }
            if (opts.all && (job.route.includes('{') || !(await routeAnswers(routeUrl)))) {
                report.skipped.push({ ...job, reason: job.route.includes('{') ? 'route needs a parameter' : 'route not 2xx (not converted yet)' });
                continue;
            }
            const slug = slugOf(job.design);
            const page = { design: job.design, route: job.route, widths: [] };
            for (const width of opts.widths) {
                const d = await capture(cdp, pathToFileURL(designFile).href, width);
                const r = await capture(cdp, routeUrl, width);
                const files = {
                    design: join(outDir, `${slug}-${width}-design.png`),
                    route: join(outDir, `${slug}-${width}-route.png`),
                    diff: join(outDir, `${slug}-${width}-diff.png`),
                };
                writeFileSync(files.design, d.png);
                writeFileSync(files.route, r.png);
                const a = decodePng(d.png), b = decodePng(r.png);
                const diff = diffImages(a, b);
                writeFileSync(files.diff, encodePng(diff));
                const entry = {
                    width,
                    diffPercent: Math.round(diff.percent * 100) / 100,
                    size: { design: [a.width, a.height], route: [b.width, b.height] },
                    files: Object.fromEntries(Object.entries(files).map(([k, v]) => [k, v.replace(`${ROOT}/`, '')])),
                    design: d.log, route: r.log,
                };
                if (r.log.status !== null && (r.log.status < 200 || r.log.status >= 300)) r.log.console.push(`document status ${r.log.status}`);
                if (d.log.external.length || r.log.external.length) hardFail = true;
                if (r.log.console.length || r.log.failed.length) hardFail = true;
                if (entry.diffPercent > opts.threshold) above = true;
                page.widths.push(entry);
                printEntry(job, entry, opts.threshold);
            }
            report.pages.push(page);
        }
    } finally {
        cdp.close();
        await chrome.close();
    }

    for (const s of report.skipped) process.stdout.write(`skip  ${s.design} → ${s.route}: ${s.reason}\n`);
    if (opts.json) writeFileSync(resolve(ROOT, opts.json), `${JSON.stringify(report, null, 2)}\n`);
    process.stdout.write(`\n${report.pages.length} page(s), out: ${opts.out}` +
        `${hardFail ? '  FAIL (external request / page error)' : ''}${above ? `  some widths above ${opts.threshold}%` : ''}\n`);
    if (hardFail) return 1;
    if (above && opts.strict) return 2;
    return 0;
}

function printEntry(job, e, threshold) {
    const mark = e.diffPercent > threshold ? '!' : 'ok';
    process.stdout.write(`${mark.padEnd(3)} ${job.design} → ${job.route} @${e.width}: diff ${e.diffPercent}%` +
        `  (design ${e.size.design.join('×')}, route ${e.size.route.join('×')})\n`);
    for (const [side, log] of [['design', e.design], ['route', e.route]]) {
        for (const u of log.external) process.stdout.write(`    EXTERNAL [${side}] ${u}\n`);
        for (const f of log.failed) process.stdout.write(`    failed   [${side}] ${f}\n`);
        for (const c of log.console) process.stdout.write(`    console  [${side}] ${c}\n`);
    }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
    main(process.argv.slice(2)).then((code) => process.exit(code), (e) => {
        process.stderr.write(`shot: ${e.stack || e.message}\n`);
        process.exit(1);
    });
}
