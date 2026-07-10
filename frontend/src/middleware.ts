import createMiddleware from 'next-intl/middleware';
import { type NextRequest, NextResponse } from 'next/server';

import { AUTH_COOKIE } from '@/shared/session';
import { routing } from '@/shared/i18n';

const intlMiddleware = createMiddleware(routing);

// Public auth screens — reachable without a session. Everything else under a
// locale prefix requires the auth flag cookie (set on verify-otp, §8.1). The
// cookie is a value-less flag, never the token itself (§11).
const PUBLIC_SEGMENTS = ['splash', 'welcome', 'signup', 'otp'];

/** Strips the leading `/fa` or `/en` locale prefix, returning the first path segment. */
function firstSegment(pathname: string): string {
  const parts = pathname.split('/').filter(Boolean);
  const [maybeLocale, ...rest] = parts;
  const isLocale = (routing.locales as readonly string[]).includes(maybeLocale);
  const segments = isLocale ? rest : parts;
  return segments[0] ?? '';
}

function localeOf(pathname: string): string {
  const [first] = pathname.split('/').filter(Boolean);
  return (routing.locales as readonly string[]).includes(first) ? first : routing.defaultLocale;
}

export default function middleware(request: NextRequest) {
  // Let next-intl handle locale detection / prefixing first.
  const response = intlMiddleware(request);
  // If it's already redirecting (e.g. adding the locale prefix), don't fight it.
  if (response.headers.get('location')) return response;

  const { pathname } = request.nextUrl;
  const segment = firstSegment(pathname);
  const locale = localeOf(pathname);
  const isAuthed = request.cookies.has(AUTH_COOKIE);
  const isPublic = segment === '' || PUBLIC_SEGMENTS.includes(segment);

  // Guard app screens: no session → back to sign-in.
  if (!isPublic && !isAuthed) {
    return NextResponse.redirect(new URL(`/${locale}/signup`, request.url));
  }

  // Signed-in users shouldn't sit on the sign-in/OTP screens.
  if (isAuthed && (segment === 'signup' || segment === 'otp')) {
    return NextResponse.redirect(new URL(`/${locale}/home`, request.url));
  }

  return response;
}

export const config = {
  // Run on everything except API routes, Next internals, and files with an
  // extension (e.g. `/favicon.ico`, `/_next/...`).
  matcher: ['/((?!api|_next|_vercel|.*\\..*).*)'],
};
