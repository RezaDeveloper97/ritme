/**
 * Optional URL prefix the whole admin is served under (Next `basePath`).
 *
 * Production runs on its own host at `/` (empty base path). Staging shares one
 * origin with the user app and the Blade panel, so admin-web lives under
 * `/panel` there. Set at BUILD time via NEXT_PUBLIC_ADMIN_BASE_PATH (next.config
 * reads the same variable for `basePath`), so it is inlined into the bundle.
 *
 * Next's `<Link>`, `router.*`, `redirect()` and middleware `nextUrl` add/strip
 * it on their own. Only raw browser APIs (`window.location`) need these helpers.
 * The API bases are NOT prefixed: `/api/admin/v1` and `/api/v1` stay at the
 * origin root, where nginx routes them to backend-go.
 */

/** `''` or `/segment[/segment…]` — leading slash, no trailing slash. */
export function normalizeBasePath(raw: string | undefined | null): string {
  const trimmed = (raw ?? '').trim().replace(/\/+$/, '');
  if (trimmed === '') return '';
  return trimmed.startsWith('/') ? trimmed : `/${trimmed}`;
}

export const BASE_PATH = normalizeBasePath(process.env.NEXT_PUBLIC_ADMIN_BASE_PATH);

/** App path (`/login?x=1`) → browser path (`/panel/login?x=1`). */
export function withBasePath(path: string, base: string = BASE_PATH): string {
  if (!base) return path;
  if (path === '/' || path === '') return base;
  return `${base}${path.startsWith('/') ? '' : '/'}${path}`;
}

/** Browser pathname (`/panel/users`) → app path (`/users`). Paths outside the base are returned as-is. */
export function stripBasePath(pathname: string, base: string = BASE_PATH): string {
  if (!base) return pathname;
  if (pathname === base) return '/';
  if (pathname.startsWith(`${base}/`)) return pathname.slice(base.length);
  return pathname;
}
