import type { NextConfig } from 'next';
import createNextIntlPlugin from 'next-intl/plugin';

// next-intl request config lives in the shared layer (cookie-selected UI locale, no URL routing).
const withNextIntl = createNextIntlPlugin('./src/shared/i18n/request.ts');

// Dev only: proxy /api/* to backend-go so the browser stays same-origin (session
// cookie + CSRF work with no CORS). Production: nginx on the admin host does this.
const proxyTarget = process.env.ADMIN_API_PROXY_TARGET?.replace(/\/+$/, '');

const isProd = process.env.NODE_ENV === 'production';

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
    return proxyTarget ? [{ source: '/api/:path*', destination: `${proxyTarget}/api/:path*` }] : [];
  },
};

export default withNextIntl(nextConfig);
