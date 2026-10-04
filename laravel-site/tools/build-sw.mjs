#!/usr/bin/env node
/**
 * Generates public/sw.js (L8-02). Runs after `vite build` (`npm run build` = `vite build && node tools/build-sw.mjs`).
 *
 * Reads public/build/manifest.json + public/build/build-id.json (both written by the Vite build) and bundles
 * resources/js/sw/sw.js into one classic-script IIFE with three constants stamped in:
 *   __BUILD_ID__          the build id (cache names; the `pwa` module carries the same id),
 *   __PRECACHE__          hashed CSS/JS/fonts/sprite of the public site + /offline + public/icons/*,
 *   __NEVER_CACHE_EXTRA__ the configured admin path (ADMIN_PATH from the environment or .env) when not "admin".
 *
 * public/sw.js is git-ignored and shipped in the deploy package; never edit it by hand. Vite also exports
 * BUILD_ID_FILE / resolveBuildId from here (vite.config.js).
 */
import { execSync } from 'node:child_process';
import { existsSync, readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const BUILD_DIR = resolve(ROOT, 'public/build');
const ICON_DIR = resolve(ROOT, 'public/icons');
const SOURCE = resolve(ROOT, 'resources/js/sw/sw.js');
const TARGET = resolve(ROOT, 'public/sw.js');

export const BUILD_ID_FILE = 'build-id.json';
export const OFFLINE_URL = '/offline';

/** Manifest entries that belong to the admin only (never precached for site visitors). */
const ADMIN_SOURCES = [/^resources\/css\/filament\//];

const BUILD_ID_PATTERN = /^\d{14}(?:-[0-9a-z]+)?$/i;

/** `YYYYMMDDHHmmss-<short sha>` (UTC); a valid BUILD_ID environment variable wins. */
export function resolveBuildId(env = process.env, now = new Date()) {
    if (env.BUILD_ID && BUILD_ID_PATTERN.test(env.BUILD_ID)) return env.BUILD_ID;

    const stamp = now.toISOString().replace(/\D/g, '').slice(0, 14);
    let sha = 'local';
    try {
        sha = execSync('git rev-parse --short HEAD', { cwd: ROOT, stdio: ['ignore', 'pipe', 'ignore'] }).toString().trim() || sha;
    } catch {
        // not a git checkout (e.g. a deploy package): keep "local"
    }

    return `${stamp}-${sha.toLowerCase()}`;
}

/** Same-origin URLs to precache from a Vite manifest object (admin entries, source maps and JSON skipped). */
export function precacheFromManifest(manifest, icons = []) {
    const files = new Set();
    for (const [key, chunk] of Object.entries(manifest)) {
        const source = chunk.src ?? key;
        if (ADMIN_SOURCES.some((pattern) => pattern.test(source))) continue;
        for (const file of [chunk.file, ...(chunk.css ?? []), ...(chunk.assets ?? [])]) {
            if (file && !/\.(map|json)$/.test(file)) files.add(`/build/${file}`);
        }
    }

    return [...[...files].sort(), OFFLINE_URL, ...icons.map((name) => `/icons/${name}`).sort()];
}

/** Admin path from ADMIN_PATH (environment first, then .env), as an extra never-cache prefix. */
export function adminPrefixes(env = process.env) {
    let path = env.ADMIN_PATH;
    if (path === undefined && existsSync(resolve(ROOT, '.env'))) {
        const match = /^ADMIN_PATH\s*=\s*["']?([^"'\s#]*)/m.exec(readFileSync(resolve(ROOT, '.env'), 'utf8'));
        path = match ? match[1] : undefined;
    }
    const clean = (path ?? 'admin').replace(/^\/+|\/+$/g, '');

    return clean === '' || clean === 'admin' ? [] : [`/${clean}`];
}

export async function buildServiceWorker() {
    const manifestFile = resolve(BUILD_DIR, 'manifest.json');
    const idFile = resolve(BUILD_DIR, BUILD_ID_FILE);
    if (!existsSync(manifestFile) || !existsSync(idFile)) {
        throw new Error('public/build/manifest.json or build-id.json missing — run `vite build` first.');
    }

    const buildId = JSON.parse(readFileSync(idFile, 'utf8')).build_id;
    if (typeof buildId !== 'string' || !BUILD_ID_PATTERN.test(buildId)) {
        throw new Error(`Invalid build id in ${BUILD_ID_FILE}: ${buildId}`);
    }

    const icons = existsSync(ICON_DIR) ? readdirSync(ICON_DIR).filter((name) => /\.(png|svg|ico)$/i.test(name)) : [];
    const precache = precacheFromManifest(JSON.parse(readFileSync(manifestFile, 'utf8')), icons);

    const { build } = await import('vite');
    const result = await build({
        configFile: false,
        root: ROOT,
        publicDir: false,
        logLevel: 'warn',
        define: {
            __BUILD_ID__: JSON.stringify(buildId),
            __PRECACHE__: JSON.stringify(precache),
            __NEVER_CACHE_EXTRA__: JSON.stringify(adminPrefixes()),
        },
        build: {
            write: false,
            minify: true,
            target: 'es2020',
            lib: { entry: SOURCE, formats: ['iife'], name: 'ritmeServiceWorker', fileName: () => 'sw.js' },
        },
    });

    const outputs = (Array.isArray(result) ? result : [result]).flatMap((entry) => entry.output ?? []);
    const chunk = outputs.find((item) => item.type === 'chunk');
    if (!chunk) throw new Error('Service worker bundle produced no output.');

    const banner = `/* Ritme service worker — GENERATED by tools/build-sw.mjs from resources/js/sw (build ${buildId}). Do not edit. */\n`;
    writeFileSync(TARGET, banner + chunk.code);

    return { buildId, precache, bytes: Buffer.byteLength(banner + chunk.code) };
}

if (import.meta.url === pathToFileURL(process.argv[1] ?? '').href) {
    buildServiceWorker()
        .then(({ buildId, precache, bytes }) => {
            console.log(`public/sw.js ${(bytes / 1024).toFixed(1)} kB — build ${buildId}, ${precache.length} precached URLs`);
        })
        .catch((error) => {
            console.error(error.message);
            process.exit(1);
        });
}
