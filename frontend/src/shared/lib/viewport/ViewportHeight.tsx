'use client';

import { useEffect } from 'react';

/**
 * Keeps `--app-vh` in sync with the *visual* viewport.
 *
 * On mobile the soft keyboard overlays the layout viewport, so a `100dvh`
 * shell keeps its bottom action bar hidden behind the keyboard — the user has
 * to tap away to dismiss it before the submit button becomes reachable.
 * `visualViewport.height` shrinks when the keyboard opens, so driving the
 * shell height from it lifts the action bar right above the keyboard.
 */
export function ViewportHeight() {
  useEffect(() => {
    const vv = window.visualViewport;
    if (!vv) return;

    const apply = () => {
      document.documentElement.style.setProperty('--app-vh', `${vv.height}px`);
      // iOS scrolls the *page* to reveal the focused field; undo that so the
      // shell stays pinned to the visual viewport.
      window.scrollTo(0, 0);
    };

    apply();
    vv.addEventListener('resize', apply);
    vv.addEventListener('scroll', apply);
    return () => {
      vv.removeEventListener('resize', apply);
      vv.removeEventListener('scroll', apply);
      document.documentElement.style.removeProperty('--app-vh');
    };
  }, []);

  return null;
}
