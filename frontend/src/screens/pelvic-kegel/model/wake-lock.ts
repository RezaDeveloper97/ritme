'use client';

import { useEffect } from 'react';

interface Sentinel {
  release: () => Promise<void>;
}

type WakeLockNavigator = Navigator & { wakeLock?: { request: (type: 'screen') => Promise<Sentinel> } };

/**
 * Keeps the screen on while `active` (Screen Wake Lock API, where supported — silently nothing elsewhere). The
 * browser drops the lock when the tab is hidden, so it is taken again on return; released on `active=false` and on
 * unmount.
 */
export function useWakeLock(active: boolean): void {
  useEffect(() => {
    const nav = typeof navigator === 'undefined' ? undefined : (navigator as WakeLockNavigator);
    if (!active || !nav?.wakeLock) return;
    let sentinel: Sentinel | null = null;
    let disposed = false;
    const acquire = () => {
      nav.wakeLock
        ?.request('screen')
        .then((lock) => {
          if (disposed) void lock.release().catch(() => undefined);
          else sentinel = lock;
        })
        .catch(() => undefined); // denied (battery saver, hidden tab): the session still runs
    };
    const onVisible = () => {
      if (document.visibilityState === 'visible') acquire();
    };
    acquire();
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      disposed = true;
      document.removeEventListener('visibilitychange', onVisible);
      void sentinel?.release().catch(() => undefined);
      sentinel = null;
    };
  }, [active]);
}
