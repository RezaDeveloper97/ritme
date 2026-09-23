import type { NextConfig } from 'next';
import createNextIntlPlugin from 'next-intl/plugin';

// next-intl request config lives in the shared layer (cookie-selected UI locale, no URL routing).
const withNextIntl = createNextIntlPlugin('./src/shared/i18n/request.ts');

// Dev only: proxy /api/* to backend-go so the browser stays same-origin (session
// cookie + CSRF work with no CORS). Production: nginx on the admin host does this.
const proxyTarget = process.env.ADMIN_API_PROXY_TARGET?.replace(/\/+$/, '');

const isProd = process.env.NODE_ENV === 'production';

// Optional URL prefix (Next `basePath`), build time. Empty in production: the
// admin owns adpanell.ritme.app. Staging serves it at `/panel` on the shared
// stage.ritmeapp.ir origin (deploy/vhost-stage.inc). The client reads the same
// variable (src/shared/config/base-path.ts) for its few raw window.location
// redirects; everything else (Link, router, middleware) is handled by Next.
const basePath = (process.env.NEXT_PUBLIC_ADMIN_BASE_PATH ?? '').trim().replace(/\/+$/, '');
if (basePath !== '' && !/^\/[A-Za-z0-9._~-]+(\/[A-Za-z0-9._~-]+)*$/.test(basePath)) {
  throw new Error(`NEXT_PUBLIC_ADMIN_BASE_PATH must look like "/panel", got "${basePath}"`);
}

function originOf(url: string | undefined): string | null {
  if (!url) return null;
  try {
    return new URL(url).origin; // relative paths (the default) throw → same-origin
  } catch {
    return null;
  }
}
const apiOrigins = [process.env.NEXT_PUBLIC_ADMIN_API_BASE_URL, process.env.NEXT_PUBLIC_API_BASE_URL]
  .map(originOf)
  .filter((o): o is string => o !== null);

// Enforced: everything the admin loads is same-origin (plus an absolute API base
// if one is configured), except uploaded images, which may come from the API
// host's /storage.
const csp = [
  "default-src 'self'",
  `script-src 'self' 'unsafe-inline'${isProd ? '' : " 'unsafe-eval'"}`,
  "style-src 'self' 'unsafe-inline'",
  "font-src 'self' data:",
  "img-src 'self' data: blob: https:",
  ["connect-src 'self'", ...apiOrigins, ...(isProd ? [] : ['ws:', 'wss:'])].join(' '),
  "object-src 'none'",
  "base-uri 'self'",
  "form-action 'self'",
  "frame-ancestors 'none'",
].join('; ');

const nextConfig: NextConfig = {
  reactStrictMode: true,
  ...(basePath ? { basePath } : {}),
  output: 'standalone',
  // admin-web is its own npm project; don't let a lockfile higher up become the trace root.
  outputFileTracingRoot: __dirname,
  poweredByHeader: false,
  async headers() {
    return [
      {
        source: '/:path*',
        headers: [
          { key: 'X-Content-Type-Options', value: 'nosniff' },
          { key: 'X-Frame-Options', value: 'DENY' },
          { key: 'Referrer-Policy', value: 'same-origin' },
          { key: 'X-Robots-Tag', value: 'noindex, nofollow' },
          { key: 'Content-Security-Policy', value: csp },
        ],
      },
    ];
  },
  async rewrites() {
    // `basePath: false`: the APIs live at the origin root (/api/admin/v1,
    // /api/v1), not under the admin's own base path.
    return proxyTarget
      ? [{ source: '/api/:path*', destination: `${proxyTarget}/api/:path*`, basePath: false as const }]
      : [];
  },
};

export default withNextIntl(nextConfig);
