'use client';

import { useEffect } from 'react';

import { useRouter } from '@/shared/i18n';
import { isAuthenticated } from '@/shared/session';

/** Below this width (px) a signed-in visitor gets the app's own screens instead of the desktop layout. */
export const WEB_DESKTOP_MIN_WIDTH = 1024;

/**
 * <1024px + a session → `router.replace(appHref)`. A visitor without a session keeps the (single-column,
 * responsive) web page: the app route would only bounce them to sign-up, and crawlers — which render narrow
 * and never hold a session — must keep seeing the public content.
 */
export function useAppRedirect(appHref: string | undefined): void {
  const router = useRouter();
  useEffect(() => {
    if (!appHref || !isAuthenticated()) return;
    const query = window.matchMedia(`(max-width: ${WEB_DESKTOP_MIN_WIDTH - 1}px)`);
    const go = () => {
      if (query.matches) router.replace(appHref);
    };
    go();
    query.addEventListener('change', go);
    return () => query.removeEventListener('change', go);
  }, [appHref, router]);
}
