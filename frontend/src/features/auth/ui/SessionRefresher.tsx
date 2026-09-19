'use client';

import { useEffect } from 'react';

import { sessionRefresher } from '../api/refresh';

/**
 * Keeps a signed-in session from running out: checks the token's expiry on
 * start and whenever the app comes back to the foreground, and refreshes it
 * only when it is inside the refresh window. Renders nothing.
 */
export function SessionRefresher() {
  useEffect(() => {
    void sessionRefresher.refreshIfDue();
    const onVisible = () => {
      if (document.visibilityState === 'visible') void sessionRefresher.refreshIfDue();
    };
    document.addEventListener('visibilitychange', onVisible);
    return () => document.removeEventListener('visibilitychange', onVisible);
  }, []);

  return null;
}
