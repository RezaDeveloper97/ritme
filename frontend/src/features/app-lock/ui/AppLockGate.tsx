'use client';

import { type ReactNode, Suspense, useEffect } from 'react';

import { isAuthenticated } from '@/shared/session';

import type { LockController } from '../model/controller';
import { getLockController, useAppLock } from '../model/store';
import { LockScreen } from './LockScreen';

/**
 * App lock + «پنهان کردن پیش‌نمایش» (B-N1-12). Mounted once in the locale
 * layout around every route and every sheet, so no navigation — client-side,
 * back/forward, a deep link — can render a screen past it.
 *
 * - Undecided (server render / hydration): the server HTML is kept, hidden by
 *   the pre-paint `html[data-app-locked]`; when this device has a lock the
 *   app's tree does not mount at all ({@link HoldWhileLocked}).
 * - Locked: **only** the lock screen renders; the app's tree is unmounted, so
 *   its data is not in the DOM either.
 * - Unlocked: the children, and the pre-paint marker is removed.
 *
 * Background/resume: `visibilitychange` + `pagehide`/`pageshow` (bfcache)
 * feed the controller, which re-locks after the configured minutes. While the
 * page is hidden and «پنهان کردن پیش‌نمایش» is on, a blur veil covers the app
 * so the OS app-switcher snapshot shows nothing readable.
 */
let unlockWait: Promise<void> | null = null;

/** Resolves when the controller reports unlocked (one shared promise while waiting). */
function waitForUnlock(c: LockController): Promise<void> {
  unlockWait ??= new Promise<void>((resolve) => {
    const off = c.subscribe(() => {
      if (!c.getSnapshot().locked) {
        off();
        unlockWait = null;
        resolve();
      }
    });
  });
  return unlockWait;
}

/**
 * Keeps the app's tree from mounting while the gate is still undecided (the
 * hydration pass, where `useAppLock()` reports null) but the device has a lock:
 * it suspends, so React leaves the server HTML dehydrated — no component of the
 * screen mounts, no effect or query runs — until the gate swaps it for the lock
 * screen (or the app unlocks). On the server there is no controller, so it
 * never suspends there.
 */
function HoldWhileLocked({ children }: { children: ReactNode }) {
  const c = getLockController();
  if (c?.getSnapshot().locked) throw waitForUnlock(c);
  return <>{children}</>;
}

export function AppLockGate({ children }: { children: ReactNode }) {
  const state = useAppLock();

  useEffect(() => {
    const c = getLockController();
    if (!c) return;
    // A lock without a session protects nothing and would trap the sign-in screens.
    if (!isAuthenticated() && c.getSnapshot().enabled) c.reset();
    const onVisibility = () => (document.visibilityState === 'hidden' ? c.onHidden() : c.onVisible());
    const onPageHide = () => c.onHidden();
    const onPageShow = () => c.onVisible();
    document.addEventListener('visibilitychange', onVisibility);
    window.addEventListener('pagehide', onPageHide);
    window.addEventListener('pageshow', onPageShow);
    return () => {
      document.removeEventListener('visibilitychange', onVisibility);
      window.removeEventListener('pagehide', onPageHide);
      window.removeEventListener('pageshow', onPageShow);
    };
  }, []);

  const locked = state?.locked ?? false;
  const decided = state !== null;
  useEffect(() => {
    if (!decided) return;
    const root = document.documentElement;
    if (locked) root.setAttribute('data-app-locked', '');
    else root.removeAttribute('data-app-locked');
  }, [decided, locked]);

  const veil = state !== null && state.hidePreview && state.hidden;

  if (state?.locked) {
    return (
      <>
        <LockScreen state={state} />
        {veil ? <div className="app-veil" aria-hidden /> : null}
      </>
    );
  }
  return (
    <>
      <Suspense fallback={null}>
        <HoldWhileLocked>{children}</HoldWhileLocked>
      </Suspense>
      {veil ? <div className="app-veil" aria-hidden /> : null}
    </>
  );
}
