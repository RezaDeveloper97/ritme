'use client';

import { usePathname } from 'next/navigation';
import { useEffect } from 'react';

import { closeSheet, readSheetTarget, useSheetStore } from '@/shared/sheet';

/**
 * Takes the browser's back button out of the app's navigation.
 *
 * Ritme ships as a site, an installed PWA and an Android WebView shell, and in
 * all three the back gesture used to walk the history stack — which is wrong
 * for an app whose screens are a tab bar, not a document trail. The visible
 * damage was logging out: the sign-out replaced the screen, then one back press
 * resurrected the signed-in screens from the bfcache, still rendering the old
 * user's data over a session that no longer exists. Every screen the user can
 * legitimately return to already has an on-screen way back (the bottom nav, the
 * onboarding `NavBack` arrow), so history back has nothing left to do.
 *
 * The trap is a duplicate history entry. On every screen change we push a
 * second entry with the *same* URL, so a back press lands on an identical URL:
 * the router re-renders the screen the user is already on and nothing moves.
 * We then re-push the duplicate, so the trap survives any number of presses.
 * Blocking `popstate` isn't possible — the event is not cancellable — and
 * pushing a *different* URL back would leave the router rendering one screen
 * while the address bar shows another.
 *
 * The single exception is a sheet: sheets live in the query string precisely so
 * that back dismisses them (see `shared/sheet/url`), and that press never
 * leaves the screen. It is recognised by the URL we came *from* carrying the
 * sheet parameter, which is why `lastHref` is tracked through wrapped
 * `pushState`/`replaceState` rather than read off `history.state` (Next owns
 * that) or off the sheet store (whose listener may already have run).
 */
export function BackGuard() {
  const pathname = usePathname();

  // Re-arm per screen, not per URL change: a sheet's own entry must stay
  // directly on top of the screen's, or the back press that closes it would
  // land on a duplicate instead.
  useEffect(() => {
    window.history.pushState(null, '', window.location.href);
  }, [pathname]);

  useEffect(() => {
    let lastHref = window.location.href;

    // Wrapped, not replaced: Next patches these too, to keep its router tree in
    // history state. Calling through preserves that.
    const { pushState, replaceState } = window.history;
    const track =
      (original: typeof pushState) =>
      (...args: Parameters<typeof pushState>) => {
        original.apply(window.history, args);
        lastHref = window.location.href;
      };
    window.history.pushState = track(pushState);
    window.history.replaceState = track(replaceState);

    const onPop = () => {
      const cameFrom = lastHref;
      lastHref = window.location.href;
      if (readSheetTarget(new URL(cameFrom).search)) return;
      window.history.pushState(null, '', window.location.href);
    };

    window.addEventListener('popstate', onPop);

    // The Android shell asks the page first before treating the hardware back
    // press as "leave the app" (see MainActivity). Returning true means the
    // page consumed it.
    window.__ritmeBack = () => {
      if (useSheetStore.getState().stack.length === 0) return false;
      closeSheet();
      return true;
    };

    return () => {
      window.removeEventListener('popstate', onPop);
      window.history.pushState = pushState;
      window.history.replaceState = replaceState;
      delete window.__ritmeBack;
    };
  }, []);

  return null;
}
