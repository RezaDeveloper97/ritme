'use client';

/**
 * "Has this visitor seen the pre-signup welcome intro?" — a one-bit, non-health
 * preference (CLAUDE.md §11), so it belongs in domain-agnostic `shared` where
 * both the splash screen (which decides where to send first-timers) and the
 * welcome screen (which sets it) can reach it without importing upward.
 *
 * Kept in `localStorage`, not a cookie: the decision is made client-side only,
 * so it never needs to be read on the server/edge.
 */
const INTRO_SEEN_KEY = 'ritme_intro_seen';

const isBrowser = (): boolean => typeof window !== 'undefined';

export function hasSeenIntro(): boolean {
  if (!isBrowser()) return false;
  return window.localStorage.getItem(INTRO_SEEN_KEY) === '1';
}

export function markIntroSeen(): void {
  if (!isBrowser()) return;
  window.localStorage.setItem(INTRO_SEEN_KEY, '1');
}
