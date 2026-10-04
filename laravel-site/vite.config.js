import { defineConfig } from 'vite';
import laravel from 'laravel-vite-plugin';
import tailwindcss from '@tailwindcss/vite';
import { buildSprite, ICON_DIR, ILLUSTRATION_DIR, SPRITE_MANIFEST_KEY, unoptimizedIllustrations } from './tools/build-sprite.mjs';

// Tailwind v4 baseline (Safari 16.4 / Chrome 111 / Firefox 128). Lightning CSS encodes versions as
// (major << 16) | (minor << 8).
const cssTargets = {
    chrome: 111 << 16,
    edge: 111 << 16,
    firefox: 128 << 16,
    safari: (16 << 16) | (4 << 8),
    ios_saf: (16 << 16) | (4 << 8),
};

/**
 * Icon sprite (L0-07): SVGO-optimised resources/svg/icons/*.svg → one <symbol> per icon, emitted as a hashed asset
 * under the manifest key "resources/svg/sprite.svg" (read by App\\View\\Components\\Icon via Vite::asset()).
 * Also fails the build when an illustration on disk is not SVGO-optimised (they are inlined as-is).
 */
function svgSprite() {
    return {
        name: 'ritme-svg-sprite',
        apply: 'build',
        buildStart() {
            this.addWatchFile(ICON_DIR);
            this.addWatchFile(ILLUSTRATION_DIR);
            const stale = unoptimizedIllustrations();
            if (stale.length > 0) {
                this.error(`Unoptimised illustrations (run "node tools/build-sprite.mjs --optimize"): ${stale.join(', ')}`);
            }
        },
        generateBundle() {
            const { sprite } = buildSprite();
            this.emitFile({
                type: 'asset',
                name: 'sprite.svg',
                originalFileName: SPRITE_MANIFEST_KEY,
                source: sprite,
            });
        },
    };
}

export default defineConfig(({ mode }) => ({
    plugins: [
        laravel({
            input: ['resources/css/app.css', 'resources/js/app.js', 'resources/css/filament/admin/theme.css'],
            refresh: true,
        }),
        tailwindcss(),
        svgSprite(),
    ],
    css: {
        lightningcss: { targets: cssTargets },
    },
    build: {
        // public/build/manifest.json — read by @vite; every file name carries a content hash.
        manifest: 'manifest.json',
        target: ['es2022', 'chrome111', 'edge111', 'firefox128', 'safari16.4'],
        cssMinify: 'lightningcss',
        sourcemap: mode !== 'production',
        assetsInlineLimit: 2048,
        rollupOptions: {
            output: {
                entryFileNames: 'assets/[name]-[hash].js',
                chunkFileNames: 'assets/[name]-[hash].js',
                assetFileNames: 'assets/[name]-[hash][extname]',
            },
        },
    },
    server: {
        watch: {
            ignored: ['**/storage/framework/views/**'],
        },
    },
}));
