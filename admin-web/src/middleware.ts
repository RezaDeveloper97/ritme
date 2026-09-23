import { NextResponse, type NextRequest } from 'next/server';

import { SESSION_COOKIE_NAMES } from '@/shared/config';

/**
 * Cheap first gate: no session cookie → /login, before any JS loads. It can't
 * tell whether a present cookie is still valid — the API decides that, and a
 * 401 sends the admin to /login from the client (shared/api).
 */
export function middleware(request: NextRequest) {
  const hasSession = SESSION_COOKIE_NAMES.some((name) => request.cookies.has(name));
  if (hasSession) return NextResponse.next();
  const url = request.nextUrl.clone();
  const next = request.nextUrl.pathname + request.nextUrl.search;
  url.pathname = '/login';
  url.search = next === '/' ? '' : `?next=${encodeURIComponent(next)}`;
  return NextResponse.redirect(url);
}

export const config = {
  // Everything except the login page, Next internals, the API proxy and static files.
  matcher: ['/((?!login|api/|_next/|favicon.ico|.*\\.[A-Za-z0-9]+$).*)'],
};
