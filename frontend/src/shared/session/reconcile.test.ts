import { describe, expect, it } from 'vitest';

import {
  reconcileOnMount,
  reconcileOnResume,
  segmentOf,
  SIGN_IN_ROUTE,
  SIGNED_IN_HOME,
  type SessionSnapshot,
} from './reconcile';

const signedIn = (over: Partial<SessionSnapshot> = {}): SessionSnapshot => ({
  hasToken: true,
  hasFlag: true,
  segment: 'home',
  onboardingPending: false,
  ...over,
});

describe('reconcileOnMount — token present (source of truth)', () => {
  it('never clears the session, with or without the flag', () => {
    for (const hasFlag of [true, false]) {
      for (const segment of ['', 'splash', 'welcome', 'signup', 'otp', 'home', 'onboarding']) {
        expect(reconcileOnMount(signedIn({ hasFlag, segment })).clearSession).toBe(false);
      }
    }
  });

  it('re-asserts the flag on every start, not only when it is missing', () => {
    expect(reconcileOnMount(signedIn({ hasFlag: true })).assertFlag).toBe(true);
    expect(reconcileOnMount(signedIn({ hasFlag: false })).assertFlag).toBe(true);
  });

  // Root cause #1: the flag lapsed, the middleware rendered /signup, and the
  // user typed an OTP again although the token was still valid.
  it.each(['splash', 'welcome', 'signup', 'otp'])(
    'moves a signed-in user with a missing flag off /%s to home',
    (segment) => {
      expect(reconcileOnMount(signedIn({ hasFlag: false, segment }))).toEqual({
        assertFlag: true,
        clearSession: false,
        redirect: SIGNED_IN_HOME,
      });
    },
  );

  it('leaves splash/welcome for home when onboarding is pending (middleware resumes the step)', () => {
    for (const segment of ['splash', 'welcome']) {
      expect(reconcileOnMount(signedIn({ segment, onboardingPending: true })).redirect).toBe(SIGNED_IN_HOME);
    }
  });

  it('keeps signup/otp reachable while onboarding is pending (changing the number)', () => {
    for (const segment of ['signup', 'otp']) {
      expect(reconcileOnMount(signedIn({ segment, onboardingPending: true })).redirect).toBeNull();
    }
  });

  it('does not redirect from app screens', () => {
    for (const segment of ['home', 'calendar', 'profile', 'onboarding', '']) {
      expect(reconcileOnMount(signedIn({ segment, hasFlag: false })).redirect).toBeNull();
    }
  });
});

describe('reconcileOnMount — no token', () => {
  it('does nothing when neither token nor flag exists', () => {
    expect(reconcileOnMount({ hasToken: false, hasFlag: false, segment: 'home', onboardingPending: false })).toEqual({
      assertFlag: false,
      clearSession: false,
      redirect: null,
    });
  });

  it('drops a flag with no token and sends a guarded screen to sign-in', () => {
    expect(reconcileOnMount({ hasToken: false, hasFlag: true, segment: 'home', onboardingPending: false })).toEqual({
      assertFlag: false,
      clearSession: true,
      redirect: SIGN_IN_ROUTE,
    });
  });

  it('drops a flag with no token but stays on a public screen', () => {
    for (const segment of ['', 'splash', 'welcome', 'signup', 'otp']) {
      expect(
        reconcileOnMount({ hasToken: false, hasFlag: true, segment, onboardingPending: false }).redirect,
      ).toBeNull();
    }
  });
});

describe('reconcileOnResume (visibilitychange / bfcache pageshow)', () => {
  it('re-asserts the flag and never signs out while a token exists', () => {
    for (const segment of ['home', 'signup', 'splash', '']) {
      expect(reconcileOnResume({ hasToken: true, segment })).toEqual({ assertFlag: true, signIn: false });
    }
  });

  it('sends a guarded screen without a token to sign-in', () => {
    expect(reconcileOnResume({ hasToken: false, segment: 'home' })).toEqual({ assertFlag: false, signIn: true });
  });

  it('leaves a public screen without a token alone', () => {
    expect(reconcileOnResume({ hasToken: false, segment: 'signup' })).toEqual({ assertFlag: false, signIn: false });
  });
});

describe('segmentOf', () => {
  it('reads the segment after the locale', () => {
    expect(segmentOf('/fa/home')).toBe('home');
    expect(segmentOf('/en/onboarding/name')).toBe('onboarding');
    expect(segmentOf('/fa')).toBe('');
    expect(segmentOf('/')).toBe('');
  });
});
