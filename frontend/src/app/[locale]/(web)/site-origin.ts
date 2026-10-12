import { headers } from 'next/headers';

/**
 * The public origin of the site, for `metadataBase` and absolute canonical / OpenGraph URLs on the (web) pages.
 *
 * `SITE_URL` is a server-only env (e.g. `https://ritmeapp.ir`, no path). It is read at request time, so it is NOT a
 * `NEXT_PUBLIC_*` value and never reaches the browser. Behind a proxy the request `Host` is whatever the proxy forwards
 * — not something a canonical URL may trust — so the request host is used only in development; a production server
 * without `SITE_URL` returns `undefined` and the pages fall back to relative (path-only) URLs.
 */
export async function siteOrigin(): Promise<string | undefined> {
  const configured = process.env.SITE_URL?.trim();
  if (configured) {
    try {
      const u = new URL(configured);
      if (u.protocol === 'http:' || u.protocol === 'https:') return u.origin;
    } catch {
      // A malformed value is treated as unset.
    }
  }
  if (process.env.NODE_ENV === 'production') return undefined;
  const h = await headers();
  const host = h.get('x-forwarded-host') ?? h.get('host');
  if (!host) return undefined;
  const proto = h.get('x-forwarded-proto') ?? (host.startsWith('localhost') || host.startsWith('127.') ? 'http' : 'https');
  return `${proto}://${host}`;
}
