import { type NextRequest, NextResponse } from 'next/server';

// Both constants come from the directive-free `cookie.ts`, so they arrive as
// plain strings/numbers here even though the barrel also exports client modules
// (server components already read `INTRO_COOKIE` the same way).
import { AUTH_COOKIE, AUTH_FLAG_MAX_AGE } from '@/shared/session';

/**
 * Sets the middleware's `ritme_auth` flag with an HTTP `Set-Cookie`.
 *
 * WebKit (iOS Safari and Home Screen apps) caps any cookie written through
 * `document.cookie` at 7 days, whatever its `max-age`. When that flag lapsed the
 * middleware rendered `/signup` to users whose token was still valid
 * (docs/investigations/session-logout.md, root cause #1). A cookie set by a
 * same-origin response keeps its full lifetime, so the client calls this on
 * sign-in and on every start/resume.
 *
 * The cookie is a boolean marker, nothing more: this route never sees or stores
 * the token (CLAUDE.md §11). It is deliberately not HttpOnly so the client can
 * read it and drop it on sign-out. Anyone can set a value-less flag on their own
 * browser anyway; it only picks which screen renders, and `SessionGuard` clears
 * a flag that has no token behind it.
 */
export const dynamic = 'force-dynamic';

function isCrossSite(request: NextRequest): boolean {
  return request.headers.get('sec-fetch-site') === 'cross-site';
}

function isHttps(request: NextRequest): boolean {
  const forwarded = request.headers.get('x-forwarded-proto');
  return (forwarded ?? request.nextUrl.protocol.replace(':', '')) === 'https';
}

export function POST(request: NextRequest) {
  if (isCrossSite(request)) return new NextResponse(null, { status: 403 });

  const response = new NextResponse(null, { status: 204 });
  response.headers.set('Cache-Control', 'no-store');
  response.cookies.set({
    name: AUTH_COOKIE,
    value: '1',
    path: '/',
    maxAge: AUTH_FLAG_MAX_AGE,
    sameSite: 'lax',
    secure: isHttps(request),
    httpOnly: false,
  });
  return response;
}
