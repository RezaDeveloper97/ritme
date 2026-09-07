'use client';

/**
 * "Has this visitor seen the pre-signup welcome intro?" — a one-bit, non-health
 * preference (CLAUDE.md §11), so it belongs in domain-agnostic `shared` where
 * both the splash screen (which decides where to send first-timers) and the
 * welcome screen (which sets it) can reach it without importing upward.
 *
 * Kept in a cookie rather than `localStorage` because the splash route now
 * renders the same decision on the server: its no-JS fallback needs a
 * destination baked into the HTML, and the edge cannot read `localStorage`.
 * Visitors carrying the old `localStorage` flag are still recognised (and
 * migrated on read) so nobody is shown the intro a second time.
 */

import { INTRO_COOKIE } from './cookie';

const LEGACY_INTRO_KEY = 'ritme_intro_seen';

const isBrowser = (): boolean => typeof window !== 'undefined';

function hasIntroCookie(): boolean {
  return document.cookie
    .split(';')
    .some((c) => c.trim().startsWith(`${INTRO_COOKIE}=1`));
}

export function hasSeenIntro(): boolean {
  if (!isBrowser()) return false;
  if (hasIntroCookie()) return true;
  if (window.localStorage.getItem(LEGACY_INTRO_KEY) !== '1') return false;
  markIntroSeen();
  return true;
}

export function markIntroSeen(): void {
  if (!isBrowser()) return;
  document.cookie = `${INTRO_COOKIE}=1; path=/; max-age=31536000; SameSite=Lax`;
}
