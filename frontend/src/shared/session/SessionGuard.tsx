'use client';

import { usePathname, useRouter } from 'next/navigation';
import { useCallback, useEffect } from 'react';

import { PUBLIC_SEGMENTS, SESSION_CLEARED_EVENT } from './cookie';
import { clearAuthToken, getAuthToken, hasAuthCookie, setAuthToken } from './token';

/**
 * Keeps the edge's view of the session (the `ritme_auth` flag cookie) and the
 * browser's (the JWT in `localStorage`) from drifting apart.
 *
 * They drift because they have different scopes: the cookie belongs to the host,
 * the token to the origin. When the site moved from `http://` to `https://`,
 * every signed-in visitor kept the cookie but lost sight of the token — the
 * middleware waved them through to `/home` while every request went out
 * unauthenticated, so the app rendered empty placeholders forever and never
 * offered a way back to sign-in. Clearing the token alone (e.g. the 401
 * interceptor) leaves the same zombie state, hence the event listener too.
 *
 * Repairs run in both directions: flag without token → sign out; token without
 * flag → restore the flag, so a genuinely signed-in user isn't bounced.
 */
export function SessionGuard() {
  const router = useRouter();
  const pathname = usePathname();

  const localeOf = useCallback(
    (path: string) => path.split('/').filter(Boolean)[0] ?? 'fa',
    [],
  );

  const isPublicScreen = useCallback((path: string) => {
    const [, segment] = path.split('/').filter(Boolean);
    // Locale root (`/fa`) is public — it redirects on to splash.
    if (!segment) return true;
    return (PUBLIC_SEGMENTS as readonly string[]).includes(segment);
  }, []);

  useEffect(() => {
    const token = getAuthToken();

    if (token) {
      if (!hasAuthCookie()) setAuthToken(token);
      return;
    }

    if (!hasAuthCookie()) return;

    // A flag with no token can never authenticate a request — drop it and, if
    // the user is sitting on a guarded screen, send them to sign in.
    clearAuthToken();
    if (!isPublicScreen(pathname)) {
      router.replace(`/${localeOf(pathname)}/signup`);
    }
  }, [isPublicScreen, localeOf, pathname, router]);

  useEffect(() => {
    const onCleared = () => {
      if (isPublicScreen(pathname)) return;
      router.replace(`/${localeOf(pathname)}/signup`);
    };

    window.addEventListener(SESSION_CLEARED_EVENT, onCleared);
    return () => window.removeEventListener(SESSION_CLEARED_EVENT, onCleared);
  }, [isPublicScreen, localeOf, pathname, router]);

  return null;
}
