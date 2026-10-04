#!/usr/bin/env node
/**
 * pwa-icons — regenerates the committed default PWA icons, favicons and manifest screenshots (L8-01).
 *
 *   node tools/pwa-icons.mjs                       icons + favicons only
 *   node tools/pwa-icons.mjs --screenshots         also capture public/icons/screenshot-{mobile,desktop}.webp
 *                                                  from a running local site (default http://127.0.0.1:8000)
 *   node tools/pwa-icons.mjs --screenshots --base http://127.0.0.1:8001
 *
 * - Brand mark = the header logo: a `primary` disc with the white `drop` icon (resources/svg/icons/drop.svg).
 *   Colours are read from the `@theme` tokens in resources/css/app.css (no hex is hard-coded here).
 * - Rasterised by headless Chrome (`--screenshot`, transparent background). No npm deps, no network: the SVGs are
 *   local files and screenshots may only target a local origin. Chrome: CHROME_PATH or /Applications/Google Chrome.app.
 * - favicon.ico (16 + 32 + 48, PNG-encoded entries) is packed here; screenshots are converted to WebP with PHP GD.
 *
 * Writes: public/icons/{icon-96,icon-192,icon-512,maskable-192,maskable-512,monochrome-512,apple-touch-icon}.png,
 * public/icons/{favicon,mask-icon}.svg, public/favicon.ico. When an admin uploads a logo, App\Domain\Pwa generates
 * the same set from it at runtime; these files are the fallback.
 */
import { execFileSync, spawn } from 'node:child_process';
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, statSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const OUT = join(ROOT, 'public/icons');
const CHROME = process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';

const args = process.argv.slice(2);
const withScreenshots = args.includes('--screenshots');
const baseIndex = args.indexOf('--base');
const base = baseIndex >= 0 ? args[baseIndex + 1] : 'http://127.0.0.1:8000';

/* ---------------------------------------------------------------- tokens + drop path */

const css = readFileSync(join(ROOT, 'resources/css/app.css'), 'utf8');
const token = (name) => {
    const m = css.match(new RegExp(`--color-${name}:\\s*(#[0-9a-fA-F]{6})`));
    if (!m) throw new Error(`token --color-${name} not found in app.css`);
    return m[1].toUpperCase();
};
const PRIMARY = token('primary');
const WHITE = token('white');
const BLACK = token('black');

const dropSvg = readFileSync(join(ROOT, 'resources/svg/icons/drop.svg'), 'utf8');
const DROP = dropSvg.match(/<path d="([^"]+)"/)[1];

/** The drop (24-unit icon, stroke 2) scaled to `box` px and centred on (cx, cy). */
const drop = (box, cx, cy, color) => {
    const k = box / 24;
    return `<path d="${DROP}" transform="translate(${cx - 12 * k} ${cy - 11.5 * k}) scale(${k})" fill="none" `
        + `stroke="${color}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>`;
};
const svg = (size, body) => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 ${size} ${size}">${body}</svg>`;

// Same proportions as the header logo: drop = half of the disc.
const disc = (s) => svg(s, `<circle cx="${s / 2}" cy="${s / 2}" r="${s / 2}" fill="${PRIMARY}"/>${drop(s / 2, s / 2, s / 2, WHITE)}`);
// Maskable: full-bleed, the mark inside the 80 % safe circle (drop at 40 % of the canvas).
const maskable = (s) => svg(s, `<rect width="${s}" height="${s}" fill="${PRIMARY}"/>${drop(s * 0.4, s / 2, s / 2, WHITE)}`);
// Apple touch icon: opaque square (iOS rounds the corners itself and paints transparency black).
const apple = (s) => svg(s, `<rect width="${s}" height="${s}" fill="${PRIMARY}"/>${drop(s * 0.5, s / 2, s / 2, WHITE)}`);
// Monochrome: only the alpha channel counts; the drop alone, inside the safe zone.
const monochrome = (s) => svg(s, drop(s * 0.6, s / 2, s / 2, WHITE));

/* ---------------------------------------------------------------- rasterise with Chrome */

if (!existsSync(CHROME)) throw new Error(`Chrome not found at ${CHROME} (set CHROME_PATH)`);
const work = mkdtempSync(join(tmpdir(), 'ritme-pwa-icons-'));

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

/** Headless Chrome `--screenshot`; Chrome lingers after writing the file, so it is killed once the PNG is complete. */
async function chromeShot(url, width, height, out, { scale = 1, transparent = true } = {}) {
    const profile = join(work, `profile-${Math.random().toString(36).slice(2)}`);
    const proc = spawn(CHROME, [
        '--headless=new', `--user-data-dir=${profile}`, '--no-first-run', '--no-default-browser-check',
        '--disable-extensions', '--disable-sync', '--disable-background-networking', '--disable-component-update',
        '--hide-scrollbars', `--force-device-scale-factor=${scale}`, '--force-color-profile=srgb',
        ...(transparent ? ['--default-background-color=00000000'] : []),
        '--virtual-time-budget=4000', `--window-size=${width},${height}`, `--screenshot=${out}`, url,
    ], { stdio: ['ignore', 'ignore', 'pipe'] });
    let stderr = '';
    proc.stderr.on('data', (d) => { stderr += d; });
    let last = -1;
    try {
        for (let i = 0; i < 600; i++) {
            if (existsSync(out)) {
                const { size } = statSync(out);
                const buf = size > 12 ? readFileSync(out) : null;
                // complete PNG = ends with the IEND chunk
                if (buf && size === last && buf.subarray(-8, -4).toString('latin1') === 'IEND') return;
                last = size;
            } else if (proc.exitCode !== null) {
                break;
            }
            await sleep(100);
        }
        throw new Error(`Chrome screenshot failed for ${url}: ${stderr.slice(-500)}`);
    } finally {
        proc.kill('SIGKILL');
    }
}

async function rasterise(name, size, markup) {
    const file = join(work, `${name}.svg`);
    writeFileSync(file, markup);
    const out = join(work, `${name}.png`);
    await chromeShot(`file://${file}`, size, size, out);
    return readFileSync(out);
}

mkdirSync(OUT, { recursive: true });

const pngs = {
    'icon-96': await rasterise('icon-96', 96, disc(96)),
    'icon-192': await rasterise('icon-192', 192, disc(192)),
    'icon-512': await rasterise('icon-512', 512, disc(512)),
    'maskable-192': await rasterise('maskable-192', 192, maskable(192)),
    'maskable-512': await rasterise('maskable-512', 512, maskable(512)),
    'monochrome-512': await rasterise('monochrome-512', 512, monochrome(512)),
    'apple-touch-icon': await rasterise('apple-touch-icon', 180, apple(180)),
};
for (const [name, buf] of Object.entries(pngs)) writeFileSync(join(OUT, `${name}.png`), buf);

// Vector favicon + Safari pinned-tab mask (single colour, filled shapes only matter).
writeFileSync(join(OUT, 'favicon.svg'), `${disc(32).replace('width="32" height="32" ', '')}\n`);
writeFileSync(join(OUT, 'mask-icon.svg'), `${svg(16, drop(14, 8, 8, BLACK)).replace('width="16" height="16" ', '')}\n`);

/* ---------------------------------------------------------------- favicon.ico (PNG entries) */

const icoSizes = [16, 32, 48];
const icoImages = [];
for (const s of icoSizes) icoImages.push(await rasterise(`ico-${s}`, s, disc(s)));
const header = Buffer.alloc(6);
header.writeUInt16LE(0, 0); header.writeUInt16LE(1, 2); header.writeUInt16LE(icoSizes.length, 4);
let offset = 6 + 16 * icoSizes.length;
const entries = icoSizes.map((s, i) => {
    const e = Buffer.alloc(16);
    e.writeUInt8(s, 0); e.writeUInt8(s, 1); e.writeUInt8(0, 2); e.writeUInt8(0, 3);
    e.writeUInt16LE(1, 4); e.writeUInt16LE(32, 6);
    e.writeUInt32LE(icoImages[i].length, 8); e.writeUInt32LE(offset, 12);
    offset += icoImages[i].length;
    return e;
});
writeFileSync(join(ROOT, 'public/favicon.ico'), Buffer.concat([header, ...entries, ...icoImages]));

/* ---------------------------------------------------------------- screenshots (optional) */

if (withScreenshots) {
    const { hostname } = new URL(base);
    if (!['127.0.0.1', 'localhost', '[::1]'].includes(hostname)) throw new Error('--base must be a local origin');
    const shots = [
        // Chrome's desktop window is at least 500 px wide, so the narrow shot uses 540 CSS px (still the mobile layout).
        ['screenshot-mobile', 540, 1170, 2],  // narrow, 1080×2340
        ['screenshot-desktop', 1280, 800, 1], // wide, 1280×800
    ];
    for (const [name, w, h, scale] of shots) {
        const png = join(work, `${name}.png`);
        await chromeShot(`${base}/`, w, h, png, { scale, transparent: false });
        execFileSync('php', ['-r', `$i = imagecreatefrompng($argv[1]); imagewebp($i, $argv[2], 80);`, png, join(OUT, `${name}.webp`)]);
    }
}

rmSync(work, { recursive: true, force: true });
process.stdout.write(`pwa-icons: wrote ${Object.keys(pngs).length} PNGs, favicon.svg, mask-icon.svg, favicon.ico`
    + `${withScreenshots ? ', 2 screenshots' : ''} (primary ${PRIMARY})\n`);
