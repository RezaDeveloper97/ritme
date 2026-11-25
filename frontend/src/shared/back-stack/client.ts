'use client';

import { createBackStack, type BackStack, type NavigationLike } from './stack';

let instance: BackStack | null = null;

/**
 * The app's one back stack, bound to `window`. Created on first use and never
 * torn down: history is a page-wide global, and Next wraps `pushState` on top
 * of ours — unwrapping on an unmount would strip Next's patch with it.
 *
 * Client only — call it from an effect or an event handler.
 */
export function getBackStack(): BackStack {
  if (instance) return instance;
  const w = window as Window & { navigation?: NavigationLike };
  instance = createBackStack({
    history: window.history,
    location: window.location,
    navigation: w.navigation,
    addEventListener: (type, listener) => window.addEventListener(type, listener),
    removeEventListener: (type, listener) => window.removeEventListener(type, listener),
    defer: (fn) => window.setTimeout(fn, 0),
    now: () => Date.now(),
  });
  return instance;
}
