'use client';

import { usePathname, useRouter } from 'next/navigation';
import { useEffect } from 'react';

import { SESSION_CLEARED_EVENT } from './cookie';
import { getOnboardingPending } from './onboarding';
import { requestPersistentStorage } from './persist';
import {
  SIGN_IN_ROUTE,
  isPublicSegment,
  reconcileOnMount,
  reconcileOnResume,
  segmentOf,
} from './reconcile';
import { assertAuthFlag, clearAuthToken, getAuthToken, hasAuthCookie } from './token';

const localeOf = (path: string) => path.split('/').filter(Boolean)[0] ?? 'fa';

/**
 * Keeps the edge's view of the session (the `ritme_auth` flag cookie) and the
 * browser's (the JWT in `localStorage`) from drifting apart, with the token as
 * the source of truth. The decisions live in `reconcile.ts` (unit-tested); this
 * component only carries them out.
 *
 * - Token present → re-assert the flag on every start and resume (never only
 *   when it is missing, so its expiry keeps sliding), and move the user off the
 *   splash/welcome/sign-in screens: a lost flag made the middleware render
 *   `/signup` to someone who was still signed in, and they signed in again.
 * - Flag without token → sign out; a guarded screen goes to sign-in.
 */
export function SessionGuard() {
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    const token = getAuthToken();
    const decision = reconcileOnMount({
      hasToken: token !== null,
      hasFlag: hasAuthCookie(),
      segment: segmentOf(pathname),
      onboardingPending: getOnboardingPending() !== null,
    });

    if (decision.assertFlag) {
      assertAuthFlag();
      requestPersistentStorage();
    }
    if (decision.clearSession) clearAuthToken();
    if (decision.redirect) router.replace(`/${localeOf(pathname)}${decision.redirect}`);
  }, [pathname, router]);

  /**
   * Re-check on restore. A screen can come back without re-mounting — the
   * bfcache reviving it, the PWA or the Android WebView resuming a backgrounded
   * tab. A signed-in user gets the flag re-asserted; a guarded screen whose
   * token is gone must not keep showing the previous session's data.
   */
  useEffect(() => {
    const revalidate = () => {
      const decision = reconcileOnResume({
        hasToken: getAuthToken() !== null,
        segment: segmentOf(pathname),
      });
      if (decision.assertFlag) assertAuthFlag();
      if (decision.signIn) window.location.replace(`/${localeOf(pathname)}${SIGN_IN_ROUTE}`);
    };

    const onPageShow = (event: PageTransitionEvent) => {
      if (event.persisted) revalidate();
    };
    const onVisible = () => {
      if (document.visibilityState === 'visible') revalidate();
    };

    window.addEventListener('pageshow', onPageShow);
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      window.removeEventListener('pageshow', onPageShow);
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, [pathname]);

  useEffect(() => {
    const onCleared = () => {
      if (isPublicSegment(segmentOf(pathname))) return;
      router.replace(`/${localeOf(pathname)}${SIGN_IN_ROUTE}`);
    };

    window.addEventListener(SESSION_CLEARED_EVENT, onCleared);
    return () => window.removeEventListener(SESSION_CLEARED_EVENT, onCleared);
  }, [pathname, router]);

  return null;
}
