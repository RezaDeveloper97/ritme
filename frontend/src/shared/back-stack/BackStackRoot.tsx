'use client';

import { useEffect } from 'react';

import { getBackStack } from './client';

/**
 * Binds the back stack to `window` on the first paint, so it sees every
 * history write from the start — the header back arrow needs to know which
 * screen is behind this one. Also marks link clicks as navigations (the
 * router wrapper in `shared/i18n` marks `push`/`replace`), so a sheet that
 * closes because its link was followed doesn't race the router with a `back()`.
 *
 * Mounted once, in the locale layout.
 */
export function BackStackRoot() {
  useEffect(() => {
    const stack = getBackStack();
    const onClick = (event: MouseEvent) => {
      const anchor = (event.target as Element | null)?.closest?.('a[href]');
      if (anchor instanceof HTMLAnchorElement && anchor.origin === window.location.origin) {
        stack.noteNavigation();
      }
    };
    document.addEventListener('click', onClick, true);
    return () => document.removeEventListener('click', onClick, true);
  }, []);

  return null;
}
