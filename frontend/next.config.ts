import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import type { NextConfig } from 'next';
import createNextIntlPlugin from 'next-intl/plugin';

// Point the next-intl plugin at our request config, which lives in the
// `shared` layer (domain-agnostic foundation) per FSD.
const withNextIntl = createNextIntlPlugin('./src/shared/i18n/request.ts');

// The running bundle's own version, baked in at build time and compared
// against /version.json by the PWA update system (shared/pwa).
const pkg = JSON.parse(readFileSync(join(__dirname, 'package.json'), 'utf8')) as {
  version: string;
};

// The per-build id stamped by scripts/generate-version.mjs (prebuild) into
// public/version.json and public/sw.js. Reusing it as Next's BUILD_ID and
// baking it into the bundle keeps worker, bundle and version.json in step, so
// the update poll can also spot a deploy that forgot to bump `version`.
function readBuildId(): string | null {
  try {
    const data = JSON.parse(readFileSync(join(__dirname, 'public', 'version.json'), 'utf8')) as {
      buildId?: unknown;
    };
    return typeof data.buildId === 'string' && /^[A-Za-z0-9._-]{1,80}$/.test(data.buildId)
      ? data.buildId
      : null;
  } catch {
    return null;
  }
}
const buildId = readBuildId();

const isProd = process.env.NODE_ENV === 'production';

function originOf(url: string | undefined): string | null {
  if (!url) return null;
  try {
    return new URL(url).origin;
  } catch {
    return null;
  }
}
// Cross-origin in production (api.ritme.app), same-origin on staging.
const apiOrigin = originOf(process.env.NEXT_PUBLIC_API_BASE_URL);

// Content-Security-Policy, REPORT-ONLY for now (pwa-audit H-4): violations are
// logged to the console, nothing is blocked. Before switching to enforcing:
//  - script-src still needs 'unsafe-inline' for Next's inline bootstrap and
//    the theme/no-flash scripts (no nonce plumbing yet);
//  - style-src 'unsafe-inline' is required by Next/font and style attributes.
// worker-src/manifest-src keep the service worker and manifest allowed.
const csp = [
  "default-src 'self'",
  `script-src 'self' 'unsafe-inline'${isProd ? '' : " 'unsafe-eval'"}`,
  "style-src 'self' 'unsafe-inline'",
  "font-src 'self' data:",
  // Admin-uploaded banners/articles come from the API's /storage.
  "img-src 'self' data: blob: https:",
  `connect-src 'self'${apiOrigin ? ` ${apiOrigin}` : ''}${isProd ? '' : ' ws: wss:'}`,
  "worker-src 'self'",
  "manifest-src 'self'",
  "object-src 'none'",
  "base-uri 'self'",
  "form-action 'self'",
  "frame-ancestors 'self'",
].join('; ');

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // Emit a self-contained server bundle (.next/standalone) so the Docker
  // runtime image can ship without node_modules or the full source tree.
  output: 'standalone',
  env: {
    NEXT_PUBLIC_APP_VERSION: pkg.version,
    NEXT_PUBLIC_BUILD_ID: buildId ?? '',
  },
  ...(buildId ? { generateBuildId: () => buildId } : {}),
  async headers() {
    // Order matters: for the same header key the LAST matching rule wins, so
    // the broad rules come first and the file-specific ones override them.
    return [
      {
        source: '/:path*',
        headers: [
          { key: 'X-Content-Type-Options', value: 'nosniff' },
          { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
          { key: 'Content-Security-Policy-Report-Only', value: csp },
        ],
      },
      {
        // HTML documents (and their RSC payloads): anything outside /_next/
        // and /api/ (route handlers keep their own, stricter no-store) without
        // a file extension. Next would otherwise send prerendered
        // pages with s-maxage=31536000, letting a CDN pin HTML that points at
        // chunks the next deploy deletes (pwa-audit H-3). Next keeps a
        // Cache-Control header that was set here instead of its own.
        source: '/:path((?!_next/|api/)(?!.*\\.[A-Za-z0-9]+$).*)',
        headers: [{ key: 'Cache-Control', value: 'private, no-cache' }],
      },
      {
        // Update-detection source of truth — a cached copy would hide releases.
        source: '/version.json',
        headers: [{ key: 'Cache-Control', value: 'no-store, max-age=0' }],
      },
      {
        // The browser must always see a new worker byte-for-byte on deploy.
        source: '/sw.js',
        headers: [
          { key: 'Cache-Control', value: 'no-store, max-age=0' },
          { key: 'Service-Worker-Allowed', value: '/' },
        ],
      },
      {
        source: '/manifest.webmanifest',
        headers: [{ key: 'Cache-Control', value: 'no-cache' }],
      },
    ];
  },
};

export default withNextIntl(nextConfig);
