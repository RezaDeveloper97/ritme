/**
 * What the session guard should do, as pure decisions (unit-tested in
 * `reconcile.test.ts`; `SessionGuard` only carries them out).
 *
 * The token in `localStorage` is the client's source of truth. The `ritme_auth`
 * flag cookie is only the edge middleware's copy of it, and that copy gets lost
 * on its own — WebKit caps script-written cookies at 7 days, users clear
 * cookies, the host/origin scopes differ. Losing the flag must never cost a
 * user their session (docs/investigations/session-logout.md, root cause #1).
 */

import { PUBLIC_SEGMENTS } from './cookie';

/** Where a signed-in visitor goes instead of an auth screen. The middleware
 *  sends them on to their pending onboarding step, if they have one. */
export const SIGNED_IN_HOME = '/home';
export const SIGN_IN_ROUTE = '/signup';

/** Entry screens a signed-in visitor always skips. */
const ENTRY_SEGMENTS = ['splash', 'welcome'] as const;
/** Sign-in screens. With onboarding still pending they stay reachable on
 *  purpose: the onboarding back arrow leads here to change the number. */
const SIGN_IN_SEGMENTS = ['signup', 'otp'] as const;

export interface SessionSnapshot {
  hasToken: boolean;
  hasFlag: boolean;
  /** The first path segment after the locale (`''` for the locale root). */
  segment: string;
  onboardingPending: boolean;
}

export interface MountDecision {
  /** Put the flag back (and have the server re-issue it). */
  assertFlag: boolean;
  /** Drop the flag and whatever is left of the session. */
  clearSession: boolean;
  /** Locale-relative route to replace the current one with. */
  redirect: string | null;
}

export function isPublicSegment(segment: string): boolean {
  // The locale root (`/fa`) is public — it redirects on to the splash.
  return segment === '' || (PUBLIC_SEGMENTS as readonly string[]).includes(segment);
}

/** The first segment after the locale in a pathname like `/fa/home/x`. */
export function segmentOf(pathname: string): string {
  return pathname.split('/').filter(Boolean)[1] ?? '';
}

/** Decision on start and on every route change. */
export function reconcileOnMount(s: SessionSnapshot): MountDecision {
  if (s.hasToken) {
    const leaves =
      (ENTRY_SEGMENTS as readonly string[]).includes(s.segment) ||
      (!s.onboardingPending && (SIGN_IN_SEGMENTS as readonly string[]).includes(s.segment));
    return { assertFlag: true, clearSession: false, redirect: leaves ? SIGNED_IN_HOME : null };
  }

  if (!s.hasFlag) return { assertFlag: false, clearSession: false, redirect: null };

  // A flag with no token can never authenticate a request.
  return {
    assertFlag: false,
    clearSession: true,
    redirect: isPublicSegment(s.segment) ? null : SIGN_IN_ROUTE,
  };
}

export interface ResumeDecision {
  assertFlag: boolean;
  /** Full-document replace to sign-in (the screen may hold a dead session's data). */
  signIn: boolean;
}

/**
 * Decision when the page comes back without re-mounting (bfcache, a resumed
 * PWA or WebView). A present token is never dropped here — only re-asserted.
 */
export function reconcileOnResume(s: Pick<SessionSnapshot, 'hasToken' | 'segment'>): ResumeDecision {
  if (s.hasToken) return { assertFlag: true, signIn: false };
  return { assertFlag: false, signIn: !isPublicSegment(s.segment) };
}
