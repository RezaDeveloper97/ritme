'use client';

/**
 * The "registration unfinished" flag, written client-side so the edge
 * middleware can gate on it without a round-trip (mirrors how `token.ts`
 * maintains the auth flag).
 *
 * It holds the step key to resume at rather than a bare `1`: the onboarding
 * steps are independent routes, so without a remembered position a returning
 * visitor would be dropped back on the first question with their answers
 * already filled in.
 */

import { ONBOARDING_COOKIE } from './cookie';

const isBrowser = (): boolean => typeof window !== 'undefined';

/** Mark onboarding as unfinished, resuming at `step` (a step key). */
export function setOnboardingPending(step: string): void {
  if (!isBrowser()) return;
  document.cookie = `${ONBOARDING_COOKIE}=${step}; path=/; max-age=31536000; SameSite=Lax`;
}

/** The step to resume at, or null when onboarding is finished / not started. */
export function getOnboardingPending(): string | null {
  if (!isBrowser()) return null;
  const entry = document.cookie
    .split(';')
    .map((c) => c.trim())
    .find((c) => c.startsWith(`${ONBOARDING_COOKIE}=`));
  const value = entry?.slice(ONBOARDING_COOKIE.length + 1) ?? '';
  return value || null;
}

/** Registration finished (or the session ended) — stop gating. */
export function clearOnboardingPending(): void {
  if (!isBrowser()) return;
  document.cookie = `${ONBOARDING_COOKIE}=; path=/; max-age=0; SameSite=Lax`;
}
