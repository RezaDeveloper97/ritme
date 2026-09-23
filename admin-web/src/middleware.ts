import { NextResponse, type NextRequest } from 'next/server';

import { SESSION_COOKIE_NAMES } from '@/shared/config';
import { publicOrigin } from '@/shared/lib';

/**
 * Cheap first gate: no session cookie → /login, before any JS loads. It can't
 * tell whether a present cookie is still valid — the API decides that, and a
 * 401 sends the admin to /login from the client (shared/api).
 *
 * Base path (NEXT_PUBLIC_ADMIN_BASE_PATH, e.g. `/panel` on staging): Next strips
 * it from `nextUrl.pathname` and prefixes the matcher below, so `next` is an app
 * path (what router.replace expects); the redirect target re-adds it.
 */
export function middleware(request: NextRequest) {
  const hasSession = SESSION_COOKIE_NAMES.some((name) => request.cookies.has(name));
  if (hasSession) return NextResponse.next();
  const next = request.nextUrl.pathname + request.nextUrl.search;
  const query = next === '/' ? '' : `?next=${encodeURIComponent(next)}`;
  // Absolute on the PUBLIC origin: nextUrl.origin is the server's bind address
  // (localhost:3000) behind nginx. basePath is re-added by hand for the same reason.
  const origin = publicOrigin(request.headers, request.nextUrl.origin);
  return NextResponse.redirect(new URL(`${request.nextUrl.basePath}/login${query}`, origin));
}

export const config = {
  // Everything except the login page, Next internals, the API proxy and static files.
  // The explicit '/' is for a base path: Next prefixes matchers with it, and
  // `/panel/(…)` alone would not match the bare `/panel` (the dashboard).
  matcher: ['/', '/((?!login|api/|_next/|favicon.ico|.*\\.[A-Za-z0-9]+$).*)'],
};
