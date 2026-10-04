import { defineConfig } from 'vite';
import laravel from 'laravel-vite-plugin';
import tailwindcss from '@tailwindcss/vite';

// Tailwind v4 baseline (Safari 16.4 / Chrome 111 / Firefox 128). Lightning CSS encodes versions as
// (major << 16) | (minor << 8).
const cssTargets = {
    chrome: 111 << 16,
    edge: 111 << 16,
    firefox: 128 << 16,
    safari: (16 << 16) | (4 << 8),
    ios_saf: (16 << 16) | (4 << 8),
};

export default defineConfig(({ mode }) => ({
    plugins: [
        laravel({
            input: ['resources/css/app.css', 'resources/js/app.js'],
            refresh: true,
        }),
        tailwindcss(),
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
