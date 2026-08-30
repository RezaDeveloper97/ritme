'use client';

import { hrefWithSheet, readSheetTarget } from './url';
import { topOf, useSheetStore } from './store';
import type { SheetTarget } from './types';

/**
 * How many history entries *we* pushed to open the sheets currently on screen.
 *
 * Closing then means going back — so the URL, the store and the browser's own
 * back button all agree — instead of pushing a further entry that would make
 * back re-open the sheet the user just dismissed. It drops to zero when a sheet
 * was deep-linked, because that entry belongs to whoever opened the link.
 *
 * Module scope on purpose: history is a single global, and so is this.
 */
let pushedEntries = 0;

/**
 * A deep-linked sheet the user dismissed. That branch of `closeSheet` can only
 * rewrite the *current* entry, so the entry underneath still carries
 * `?sheet=…`; without this the next back press would surface it and re-open
 * the sheet the user just closed.
 */
let dismissedDeepLink: SheetTarget | null = null;

const sameTarget = (a: SheetTarget | null, b: SheetTarget | null): boolean =>
  a !== null && b !== null && a.id === b.id && a.arg === b.arg;

/**
 * Opens a sheet over whatever is currently on screen — a plain screen, or
 * another sheet (the article library opening an article).
 *
 * Call it from an event handler: it reads `window`.
 */
export function openSheet(id: string, arg?: string): void {
  const target: SheetTarget = arg ? { id, arg } : { id };
  // Next patches pushState to preserve its own router state, so this is the
  // supported way to change the query string without a server round-trip.
  window.history.pushState(null, '', hrefWithSheet(window.location.href, target));
  pushedEntries += 1;
  dismissedDeepLink = null;

  const { stack, setStack } = useSheetStore.getState();
  setStack([...stack, target]);
}

/** Dismisses the top sheet, revealing whatever it was covering. */
export function closeSheet(): void {
  const { stack, setStack } = useSheetStore.getState();
  if (stack.length === 0) return;

  if (pushedEntries > 0) {
    // The popstate listener does the rest, which keeps "user pressed back" and
    // "user tapped close" on one code path.
    window.history.back();
    return;
  }

  // Deep link straight into a sheet: there is no entry of ours to pop, so drop
  // the parameters in place and leave the back button pointing out of the app.
  dismissedDeepLink = topOf(stack);
  window.history.replaceState(null, '', hrefWithSheet(window.location.href, null));
  setStack([]);
}

/**
 * Binds the store to the browser's history: adopts a sheet named in the URL on
 * first load (a shared link, a restored PWA session) and follows every
 * back/forward press afterwards.
 *
 * Returns the teardown; `SheetHost` owns the single call to it.
 */
export function syncSheetWithHistory(): () => void {
  const onPop = () => {
    const target = readSheetTarget(window.location.search);
    const { stack, setStack } = useSheetStore.getState();

    if (!target) {
      pushedEntries = 0;
      dismissedDeepLink = null;
      setStack([]);
      return;
    }

    // The back press landed on the pre-dismissal entry of a deep-linked sheet.
    // Adopting it would undo the user's own tap on the close button, so strip
    // the parameters instead and stay on the screen underneath.
    if (stack.length === 0 && sameTarget(dismissedDeepLink, target)) {
      window.history.replaceState(null, '', hrefWithSheet(window.location.href, null));
      return;
    }

    // Back one level: the URL now names the sheet directly beneath the top.
    if (stack.length >= 2 && sameTarget(stack[stack.length - 2], target)) {
      pushedEntries = Math.max(0, pushedEntries - 1);
      setStack(stack.slice(0, -1));
      return;
    }

    if (sameTarget(topOf(stack), target)) return;

    // Anything else — a deep link, a forward press, a restored entry we never
    // pushed — is a stack we can't reconstruct, so the URL wins outright.
    pushedEntries = 0;
    setStack([target]);
  };

  onPop();
  window.addEventListener('popstate', onPop);
  return () => window.removeEventListener('popstate', onPop);
}

/**
 * Closes every open sheet when the screen underneath changes (a bottom-nav tab,
 * a redirect). Without it a sheet would hang over an unrelated screen, because
 * a route change leaves our query parameters behind.
 */
export function closeSheetOnRouteChange(): void {
  if (useSheetStore.getState().stack.length === 0) return;
  pushedEntries = 0;
  dismissedDeepLink = null;
  useSheetStore.getState().setStack([]);
}
