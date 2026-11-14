#!/usr/bin/env node
/**
 * pwa-icons — regenerates the committed default PWA icons, favicons and manifest screenshots (L8-01).
 *
 *   node tools/pwa-icons.mjs                       icons + favicons only
 *   node tools/pwa-icons.mjs --screenshots         also capture public/icons/screenshot-{mobile,desktop}.webp
 *                                                  from a running local site (default http://127.0.0.1:8000)
 *   node tools/pwa-icons.mjs --screenshots --base http://127.0.0.1:8001
 *
 * - Brand mark = the Ritme app logo (resources/images/brand/ritme-logo-512.png, copied from the Android store icon)
 *   on white: favicons/app icons are a white rounded tile, maskable + apple-touch are full-bleed white.
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

/* ---------------------------------------------------------------- tokens + logo */

const css = readFileSync(join(ROOT, 'resources/css/app.css'), 'utf8');
const token = (name) => {
    const m = css.match(new RegExp(`--color-${name}:\\s*(#[0-9a-fA-F]{6})`));
    if (!m) throw new Error(`token --color-${name} not found in app.css`);
    return m[1].toUpperCase();
};
const WHITE = token('white');
const BLACK = token('black');

const dropSvg = readFileSync(join(ROOT, 'resources/svg/icons/drop.svg'), 'utf8');
const DROP = dropSvg.match(/<path d="([^"]+)"/)[1];

/** The drop (24-unit icon, stroke 2) scaled to `box` px and centred on (cx, cy) — only for the Safari mask icon. */
const drop = (box, cx, cy, color, stroke = 2) => {
    const k = box / 24;
    return `<path d="${DROP}" transform="translate(${cx - 12 * k} ${cy - 11.5 * k}) scale(${k})" fill="none" `
        + `stroke="${color}" stroke-width="${stroke}" stroke-linecap="round" stroke-linejoin="round"/>`;
};
const svg = (size, body) => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 ${size} ${size}">${body}</svg>`;

/** The Ritme app logo (the «R» ring, transparent PNG — same file as the Android store icon), as a data URI. */
const LOGO = `data:image/png;base64,${readFileSync(join(ROOT, 'resources/images/brand/ritme-logo-512.png')).toString('base64')}`;

/** The logo at `scale` of an `s` canvas, centred; `white` turns it into a white silhouette (alpha kept). */
const logo = (s, scale, white = false) => {
    const box = s * scale;
    const at = (s - box) / 2;
    const filter = white ? ' filter="url(#white)"' : '';
    const defs = white ? '<defs><filter id="white"><feColorMatrix values="0 0 0 0 1  0 0 0 0 1  0 0 0 0 1  0 0 0 1 0"/></filter></defs>' : '';
    return `${defs}<image href="${LOGO}" x="${at}" y="${at}" width="${box}" height="${box}"${filter}/>`;
};

// Favicon: white rounded tile (readable on dark and light tabs), logo nearly edge to edge.
const tile = (s) => svg(s, `<rect width="${s}" height="${s}" rx="${s * 0.22}" fill="${WHITE}"/>${logo(s, 0.96)}`);
// App icon (`any`): the same white tile.
const disc = (s) => svg(s, `<rect width="${s}" height="${s}" rx="${s * 0.22}" fill="${WHITE}"/>${logo(s, 0.8)}`);
// Maskable: full-bleed white, the ring inside the 80 % safe circle.
const maskable = (s) => svg(s, `<rect width="${s}" height="${s}" fill="${WHITE}"/>${logo(s, 0.7)}`);
// Apple touch icon: opaque white square (iOS rounds the corners itself and paints transparency black).
const apple = (s) => svg(s, `<rect width="${s}" height="${s}" fill="${WHITE}"/>${logo(s, 0.78)}`);
// Monochrome: only the alpha channel counts; the logo as a white silhouette inside the safe zone.
const monochrome = (s) => svg(s, logo(s, 0.7, true));

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

// SVG favicon = a small 64 px raster of the tile (the 512 px logo would make it ~200 KB) + Safari pinned-tab mask.
const fav64 = (await rasterise('favicon-64', 64, tile(64))).toString('base64');
writeFileSync(join(OUT, 'favicon.svg'), `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"><image href="data:image/png;base64,${fav64}" width="64" height="64"/></svg>\n`);
writeFileSync(join(OUT, 'mask-icon.svg'), `${svg(16, drop(14, 8, 8, BLACK)).replace('width="16" height="16" ', '')}\n`);

/* ---------------------------------------------------------------- favicon.ico (PNG entries) */

const icoSizes = [16, 32, 48];
const icoImages = [];
for (const s of icoSizes) icoImages.push(await rasterise(`ico-${s}`, s, tile(s)));
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
    + `${withScreenshots ? ', 2 screenshots' : ''} (Ritme logo on white)\n`);
