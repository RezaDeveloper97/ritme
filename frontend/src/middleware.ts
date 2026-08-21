import { type NextRequest, NextResponse } from 'next/server';

// Imported from the leaf module, not the `@/shared/session` barrel: the barrel
// also exports client components, which have no place in the edge bundle.
import { AUTH_COOKIE, ONBOARDING_COOKIE, PUBLIC_SEGMENTS } from '@/shared/session/cookie';
// Leaf import for the same reason: `model/steps` is pure and framework-free,
// while the entity's barrel pulls in client components and a zustand store.
import { isOnboardingStep, onboardingRoute } from '@/entities/user/model/steps';
// Leaf imports again — the i18n barrel re-exports client components.
import { LOCALE_COOKIE } from '@/shared/i18n/navigation';
import { getLocaleRegistry } from '@/shared/i18n/registry';

/**
 * Locale prefixing + route guards.
 *
 * The locale prefix is applied here rather than by next-intl's middleware
 * because the set of locales is resolved at runtime from the backend
 * (CLAUDE.md §6) — next-intl needs it fixed at build time, which would mean
 * redeploying the frontend for every language an admin adds. The registry is
 * cached per worker, so this costs one API call every few minutes, and falls
 * back to the bundled locales if the backend is unreachable.
 */
export default async function middleware(request: NextRequest) {
  const { codes, defaultLocale } = await getLocaleRegistry();
  const { pathname } = request.nextUrl;
  const parts = pathname.split('/').filter(Boolean);
  const [maybeLocale, ...rest] = parts;

  // Every URL carries its locale, so an unprefixed one is redirected rather
  // than rendered — the user's last choice wins, else the product default.
  if (!maybeLocale || !codes.includes(maybeLocale)) {
    const preferred = request.cookies.get(LOCALE_COOKIE)?.value;
    const locale = preferred && codes.includes(preferred) ? preferred : defaultLocale;
    const url = request.nextUrl.clone();
    url.pathname = `/${locale}${pathname === '/' ? '' : pathname}`;

    return NextResponse.redirect(url);
  }

  const locale = maybeLocale;
  const segment = rest[0] ?? '';
  const isAuthed = request.cookies.has(AUTH_COOKIE);
  const isPublic =
    segment === '' || (PUBLIC_SEGMENTS as readonly string[]).includes(segment);

  // Guard app screens: no session → back to sign-in.
  if (!isPublic && !isAuthed) {
    return NextResponse.redirect(new URL(`/${locale}/signup`, request.url));
  }

  // A verified mobile is not a finished registration. Verifying an OTP hands
  // out the auth cookie before a single onboarding question is answered, so
  // without this a visitor who backs out of the flow — or re-enters a different
  // number from the name screen — is waved straight into the app with no
  // profile at all. The cookie carries the step they stopped on, so the flow
  // resumes rather than restarts.
  const pendingStep = request.cookies.get(ONBOARDING_COOKIE)?.value;
  if (isAuthed && pendingStep) {
    // `signup`/`otp` stay reachable on purpose: changing your mind about the
    // number is exactly the case that used to leak into the app.
    const allowed =
      segment === 'onboarding' || (PUBLIC_SEGMENTS as readonly string[]).includes(segment);
    if (!allowed) {
      const step = isOnboardingStep(pendingStep) ? pendingStep : 'name';
      return NextResponse.redirect(
        new URL(`/${locale}${onboardingRoute(step)}`, request.url),
      );
    }
    return NextResponse.next();
  }

  // Signed-in users shouldn't sit on the sign-in/OTP screens.
  if (isAuthed && (segment === 'signup' || segment === 'otp')) {
    return NextResponse.redirect(new URL(`/${locale}/home`, request.url));
  }

  return NextResponse.next();
}

export const config = {
  // Run on everything except API routes, Next internals, and files with an
  // extension (e.g. `/favicon.ico`, `/_next/...`).
  matcher: ['/((?!api|_next|_vercel|.*\\..*).*)'],
};
