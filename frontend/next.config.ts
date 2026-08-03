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

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // Emit a self-contained server bundle (.next/standalone) so the Docker
  // runtime image can ship without node_modules or the full source tree.
  output: 'standalone',
  env: {
    NEXT_PUBLIC_APP_VERSION: pkg.version,
  },
  async headers() {
    return [
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
