#!/usr/bin/env node
/**
 * Icon sprite + illustration pipeline (L0-07).
 *
 *   node tools/build-sprite.mjs             build the sprite and print a size report
 *   node tools/build-sprite.mjs --out=FILE  also write the sprite to FILE (debugging)
 *   node tools/build-sprite.mjs --optimize  re-run SVGO over resources/svg/{icons,illustrations} in place
 *   node tools/build-sprite.mjs --extract   (re)extract icons + illustrations from design/html/*.html
 *
 * Sources of truth:
 *   resources/svg/icons/<name>.svg          24×24 stroke icons, no colour/stroke attributes (currentColor comes
 *                                           from the referencing <svg>, see <x-icon>)
 *   resources/svg/illustrations/<name>.svg  multi-colour artwork, inlined by <x-illustration>
 *
 * The sprite itself is never committed: vite.config.js imports buildSprite() and emits it as a hashed asset
 * (manifest key "resources/svg/sprite.svg"), so `@vite`/Vite::asset() always point at the current hash.
 *
 * Preload: deliberately none. Every page's header uses an icon (the drop logo), so the parser discovers the
 * sprite in the first few KB of HTML; it is ~2.4 KB gzip, immutable-cached, and fetched once per page by all
 * <use> elements. A <link rel="preload" as="image"> does not reliably match the request <use> makes (Chrome and
 * Safari log an unused-preload warning), so it would cost a request instead of saving one.
 */
import { createHash } from 'node:crypto';
import { mkdirSync, readFileSync, readdirSync, rmSync, writeFileSync } from 'node:fs';
import { basename, dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { optimize } from 'svgo';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
export const ICON_DIR = join(ROOT, 'resources/svg/icons');
export const ILLUSTRATION_DIR = join(ROOT, 'resources/svg/illustrations');
export const SPRITE_MANIFEST_KEY = 'resources/svg/sprite.svg';
const DESIGN_DIR = join(ROOT, 'design/html');

const NAME_RE = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

// ---------------------------------------------------------------------------------------------- SVGO configs

/**
 * Icons: the source file keeps the design's stroke setup on the root <svg> (so SVGO knows the shapes are stroked
 * with round caps and keeps zero-length "dot" segments); buildSprite() drops the root and keeps only the geometry,
 * so colour, width, caps and joins are inherited from the referencing <svg> (<x-icon>).
 */
const ICON_ROOT_ATTRS = 'fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"';
const iconSvgo = {
    multipass: true,
    floatPrecision: 2,
    plugins: [
        // convertPathData rewrites the design's "h.01" dots into zero-length closed subpaths, which render nothing.
        { name: 'preset-default', params: { overrides: { convertPathData: false } } },
        'convertStyleToAttrs',
        'removeDimensions',
        { name: 'removeAttrs', params: { attrs: ['svg:(class|style|aria-hidden|role)', '(class|style)'] } },
    ],
};

/** Illustrations: keep colours (they are artwork), drop dimensions (the component sets width/height). */
const illustrationSvgo = {
    multipass: true,
    floatPrecision: 2,
    plugins: [
        'preset-default',
        'convertStyleToAttrs',
        'removeDimensions',
        { name: 'removeAttrs', params: { attrs: ['svg:(class|style|aria-hidden|role)'] } },
    ],
};

export function optimizeIcon(svg, path = 'icon.svg') {
    return optimize(svg, { ...iconSvgo, path }).data;
}

export function optimizeIllustration(svg, path = 'illustration.svg') {
    return optimize(svg, { ...illustrationSvgo, path }).data;
}

// ------------------------------------------------------------------------------------------------- sprite

const listSvgs = (dir) =>
    readdirSync(dir)
        .filter((f) => f.endsWith('.svg'))
        .sort();

function assertName(name, file) {
    if (!NAME_RE.test(name)) {
        throw new Error(`[build-sprite] invalid SVG file name "${file}" (use kebab-case).`);
    }
}

function assertClean(svg, file) {
    if (/\sstyle=/.test(svg)) {
        throw new Error(`[build-sprite] ${file} still has a style="" attribute.`);
    }
    if (/(?:href|url\()\s*=?\s*["']?(?:https?:)?\/\//i.test(svg)) {
        throw new Error(`[build-sprite] ${file} references an external resource.`);
    }
}

/**
 * @returns {{ sprite: string, icons: string[], bytes: number }}
 */
export function buildSprite(dir = ICON_DIR) {
    const icons = [];
    const symbols = [];

    for (const file of listSvgs(dir)) {
        const name = basename(file, '.svg');
        assertName(name, file);

        const svg = optimizeIcon(readFileSync(join(dir, file), 'utf8'), file);
        assertClean(svg, file);

        const viewBox = svg.match(/viewBox="([^"]+)"/)?.[1] ?? '0 0 24 24';
        const inner = svg.replace(/^[\s\S]*?<svg\b[^>]*>/, '').replace(/<\/svg>\s*$/, '');
        if (inner.trim() === '') {
            throw new Error(`[build-sprite] ${file} is empty after optimisation.`);
        }

        icons.push(name);
        symbols.push(`<symbol id="${name}" viewBox="${viewBox}">${inner}</symbol>`);
    }

    const sprite = `<svg xmlns="http://www.w3.org/2000/svg">${symbols.join('')}</svg>`;

    return { sprite, icons, bytes: Buffer.byteLength(sprite) };
}

/**
 * Illustrations are served from disk as-is, so they must already be optimised. Returns the files that are not.
 */
export function unoptimizedIllustrations(dir = ILLUSTRATION_DIR) {
    return listSvgs(dir).filter((file) => {
        assertName(basename(file, '.svg'), file);
        const source = readFileSync(join(dir, file), 'utf8');
        assertClean(source, file);

        return optimizeIllustration(source, file) !== source;
    });
}

function optimizeInPlace() {
    for (const file of listSvgs(ICON_DIR)) {
        const path = join(ICON_DIR, file);
        writeFileSync(path, optimizeIcon(readFileSync(path, 'utf8'), file) + '\n');
    }
    for (const file of listSvgs(ILLUSTRATION_DIR)) {
        const path = join(ILLUSTRATION_DIR, file);
        // Converge: SVGO output fed back to SVGO must be stable for the build-time check.
        let svg = readFileSync(path, 'utf8');
        for (let i = 0; i < 5; i++) {
            const next = optimizeIllustration(svg, file);
            if (next === svg) break;
            svg = next;
        }
        writeFileSync(path, svg);
    }
}

// ------------------------------------------------------------------------------------------------ extract

/** Normalised inner markup hash (sha1, 10 chars) → icon name. Names follow docs/AUDIT.md §4.1. */
const ICON_NAMES = {
    '3a6f0fb0dd': 'download',
    '8fc3e48bd5': 'check',
    'd973963897': 'drop',
    '75c147c2bf': 'user',
    '06b13d60c9': 'star',
    '34c7d010d4': 'heart',
    '4d3e86794b': 'arrow-left',
    '09bdb1c143': 'chevron-left',
    '770f7a4c8a': 'book',
    '94ca785b0f': 'sprout',
    'ef031c6cd8': 'person',
    '52a670a6bd': 'store',
    '29b4b7b84e': 'mic',
    '28b8575a12': 'x',
    '04520c4c1a': 'shield-check',
    '1924625159': 'egg',
    '5328d8fa7c': 'calendar',
    '017afb2599': 'stethoscope',
    'f39464e1a6': 'target',
    '9ae8360ff6': 'bookmark',
    'c3ca564454': 'alert-triangle',
    '503aa22870': 'moon',
    'a0c2cce9ef': 'users',
    '0d5caed45d': 'lock',
    'f9758e0427': 'check-square',
    '1310aa5f8e': 'flame',
    '60d2940796': 'sparkle',
    '6fd2bd82ca': 'map-pin',
    '8ed468f17a': 'calculator',
    '083e7f9361': 'plus',
    '192e297d5d': 'trash',
    'c36e4d2683': 'map',
    'b66913513e': 'file-text',
    '529499ac2b': 'package',
    '46d12c4a6f': 'search',
    '77afd00a67': 'truck',
    '9ab79d5312': 'minus',
    'dd7cf9cdc9': 'message',
    '3f09992c6d': 'bell',
    '15a05bb463': 'eye-off',
    '09a3b7c04a': 'bag',
    '9a60316bf7': 'navigate',
    'b6f3b20d25': 'waves',
    'f8e46c785a': 'phone',
    'c4f0d82359': 'pill',
    'cb02fd75b3': 'credit-card',
    'a3446e1dfd': 'info',
    '90a242eb55': 'grid',
    'e4d7c48a99': 'home',
    '105ab2f878': 'syringe',
    '59814c5051': 'return',
    'f8532e2a5f': 'blocks',
    '0a9a330d53': 'music',
    '236ccf59d6': 'exercise',
    '85396eff6b': 'hand',
    '49354a88db': 'mail',
    '950f6bd749': 'paper-plane',
    '353ceb7c05': 'upload',
    '7ebb33ddfe': 'share',
    '0dddff3379': 'camera',
    'b53532a393': 'female',
    'e08c695af6': 'thermometer',
    'acb8092543': 'bottle',
    'bfc8d833fb': 'bowl',
    'cca9300931': 'cart',
    'd577fa064f': 'parking',
    '6d46fe6263': 'tree',
    '0c45069f98': 'filter',
    '8bc9d9af34': 'chart-bar',
    '7626fc83e2': 'clock',
    'c6f3dda5e6': 'folder',
    '1a3ee742d4': 'pulse',
    'abe90a5631': 'tag',
    '150bf58910': 'card',
    '07bd7eb55a': 'ruler',
    'c32c692d8a': 'cart-alt',
    '71d1a3707c': 'flask',
};
/** Inner markup hash → illustration name. Names follow docs/AUDIT.md §4.2. */
const ILLUSTRATION_NAMES = {
    '09084ead73': 'hero-orbit',
    '7022f6183e': 'place-cover-pool',
    '5a65c77862': 'map-join',
    '872462ee7a': 'place-cover-massage',
    'e54b18a760': 'place-cover-movement',
    '63d6f3274e': 'place-cover-yoga',
    'c7c8d96474': 'place-cover-playhouse',
    'e664b8bd90': 'map-place',
    'a4bcba39db': 'place-cover-music',
    'ca0fea3e60': 'map-directory',
    '46938ab48e': 'chart-sparkline',
    'bf85919ffa': 'chart-growth',
    'f7096f1335': 'product-bodysuit',
    '78cc7c0520': 'product-sleepsuit',
    'cf7d2dc5bc': 'product-pad',
    '50549a6976': 'product-socks',
    'b93dfa94bd': 'product-hat',
    '1789539474': 'product-blanket',
    '01f3bd6555': 'product-shampoo',
    '4f8e22f4cd': 'product-cup',
    'b160312aa4': 'map-checkout',
    'fd0987718a': 'product-bottle',
    '35e14c98bf': 'product-serum',
    '2449ed2880': 'category-stroller',
    'ad02fccc47': 'category-crib',
    '47b2b6e0b9': 'product-sunscreen',
    'a28b0029cd': 'product-lipstick',
    '9035e4d49d': 'category-bath',
};
/** Shapes merged into another icon (AUDIT: cart-alt → cart). */
const MERGED = new Set(['cart-alt']);

const hash = (s) => createHash('sha1').update(s).digest('hex').slice(0, 10);
const normaliseIcon = (inner) =>
    inner
        .replace(/\s(?:stroke|fill|stroke-width|stroke-linecap|stroke-linejoin|opacity|fill-opacity|stroke-opacity)="[^"]*"/g, '')
        .replace(/\s+/g, ' ')
        .trim();

function extract() {
    const icons = new Map();
    const illustrations = new Map();

    for (const file of readdirSync(DESIGN_DIR).filter((f) => f.endsWith('.html')).sort()) {
        const html = readFileSync(join(DESIGN_DIR, file), 'utf8');
        for (const [, attrs, inner] of html.matchAll(/<svg\b([^>]*)>([\s\S]*?)<\/svg>/g)) {
            const viewBox = attrs.match(/viewBox="([^"]+)"/)?.[1];
            if (viewBox === '0 0 24 24') {
                const norm = normaliseIcon(inner);
                const entry = icons.get(norm) ?? { uses: 0 };
                entry.uses++;
                icons.set(norm, entry);
            } else if (!illustrations.has(inner)) {
                const par = attrs.match(/preserveAspectRatio="([^"]+)"/)?.[1];
                illustrations.set(inner, { viewBox, par });
            }
        }
    }

    const unknown = [];
    rmSync(ICON_DIR, { recursive: true, force: true });
    rmSync(ILLUSTRATION_DIR, { recursive: true, force: true });
    mkdirSync(ICON_DIR, { recursive: true });
    mkdirSync(ILLUSTRATION_DIR, { recursive: true });

    for (const [norm, { uses }] of icons) {
        const name = ICON_NAMES[hash(norm)];
        if (!name) {
            unknown.push(`icon ${hash(norm)} (${uses} uses): ${norm.slice(0, 80)}`);
            continue;
        }
        if (MERGED.has(name)) continue;
        const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" ${ICON_ROOT_ATTRS}>${norm}</svg>`;
        writeFileSync(join(ICON_DIR, `${name}.svg`), optimizeIcon(svg, name) + '\n');
    }

    for (const [inner, { viewBox, par }] of illustrations) {
        const name = ILLUSTRATION_NAMES[hash(inner)];
        if (!name) {
            unknown.push(`illustration ${hash(inner)} (${viewBox}): ${inner.slice(0, 60)}`);
            continue;
        }
        const parAttr = par ? ` preserveAspectRatio="${par}"` : '';
        writeFileSync(
            join(ILLUSTRATION_DIR, `${name}.svg`),
            `<svg xmlns="http://www.w3.org/2000/svg" viewBox="${viewBox}"${parAttr}>${inner}</svg>`,
        );
    }

    optimizeInPlace();

    if (unknown.length > 0) {
        console.error(`[build-sprite] ${unknown.length} unnamed shape(s) — add them to ICON_NAMES/ILLUSTRATION_NAMES:`);
        unknown.forEach((u) => console.error(`  ${u}`));
        process.exitCode = 1;
    }
}

// -------------------------------------------------------------------------------------------------- CLI

const isCli = process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url);

if (isCli) {
    const args = process.argv.slice(2);
    if (args.includes('--extract')) {
        extract();
    } else if (args.includes('--optimize')) {
        optimizeInPlace();
    }

    const { sprite, icons, bytes } = buildSprite();
    const out = args.find((a) => a.startsWith('--out='))?.slice('--out='.length);
    if (out) writeFileSync(resolve(out), sprite);

    const { gzipSync } = await import('node:zlib');
    const illustrations = listSvgs(ILLUSTRATION_DIR);
    const stale = unoptimizedIllustrations();
    console.log(
        `sprite: ${icons.length} icons, ${(bytes / 1024).toFixed(1)} KB (${(gzipSync(sprite).length / 1024).toFixed(1)} KB gzip)`,
    );
    console.log(`illustrations: ${illustrations.length}${stale.length ? `, NOT optimised: ${stale.join(', ')}` : ', all optimised'}`);
    if (stale.length) process.exitCode = 1;
}
