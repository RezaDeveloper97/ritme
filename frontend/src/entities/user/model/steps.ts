import type { PregnancyIntention } from './types';

/**
 * Onboarding is a set of independent full-page route screens rather than a
 * component wizard. This module is the single source of truth for their order
 * so the flow can **branch** on pregnancy intention without every screen
 * hardcoding its own "next" route or step number (which is what made the old
 * fixed 8-step flow brittle).
 *
 * TEMPORARILY SINGLE-BRANCH: pregnancy mode is postponed and hidden from the
 * frontend, so the intention question and the pregnancy dating-basis step are
 * commented out below and every user walks the cycle branch. Restore the two
 * marked blocks (plus `HEAD`'s `'intention'`) to bring pregnancy back.
 *
 * Two branches share a common head, then diverge:
 * - `pregnant`     → dating basis, then optional conditions.
 * - anything else  → the cycle questions, then optional conditions.
 *
 * Pure and framework-free (CLAUDE.md §7/§10) — unit-tested in `steps.test.ts`.
 */
export type OnboardingStepKey =
  | 'name'
  | 'birthday'
  | 'weight'
  | 'height'
  | 'intention'
  | 'pregnancyBasis'
  | 'periodLen'
  | 'cycleDuration'
  | 'cycleLen'
  | 'conditions';

/** Route mounted by the App Router for each step. */
const STEP_ROUTES: Record<OnboardingStepKey, string> = {
  name: '/onboarding/name',
  birthday: '/onboarding/birthday',
  weight: '/onboarding/weight',
  height: '/onboarding/height',
  intention: '/onboarding/intention',
  pregnancyBasis: '/onboarding/pregnancy-basis',
  periodLen: '/onboarding/period-len',
  cycleDuration: '/onboarding/cycle-duration',
  cycleLen: '/onboarding/cycle-len',
  conditions: '/onboarding/conditions',
};

/** Where the flow lands after the last step: the save/progress screen. */
export const SETTING_UP_ROUTE = '/onboarding/setting-up';

// Pregnancy postponed: 'intention' (the "how do you relate to pregnancy?"
// question) is dropped from the head. Restore it to re-enable the branch.
// const HEAD: OnboardingStepKey[] = ['name', 'birthday', 'weight', 'height', 'intention'];
const HEAD: OnboardingStepKey[] = ['name', 'birthday', 'weight', 'height'];

/**
 * The ordered step keys for the given intention. Before the intention step is
 * answered (`null`), we assume the cycle branch so the shared head steps still
 * have a stable position/total to display.
 */
export function onboardingSteps(_intention: PregnancyIntention | null): OnboardingStepKey[] {
  // Pregnancy postponed: the pregnant branch is disabled, so the intention no
  // longer affects the sequence. Restore with:
  // if (intention === 'pregnant') return [...HEAD, 'pregnancyBasis', 'conditions'];
  return [...HEAD, 'periodLen', 'cycleDuration', 'cycleLen', 'conditions'];
}

/** The route for a step key. */
export function onboardingRoute(key: OnboardingStepKey): string {
  return STEP_ROUTES[key];
}

/**
 * The step key a path belongs to, or null when it isn't an onboarding step.
 * Accepts locale-prefixed paths (`/fa/onboarding/name`) so the middleware and
 * the client tracker can both feed it a raw pathname.
 */
export function onboardingStepFromPath(pathname: string): OnboardingStepKey | null {
  const segments = pathname.split('/').filter(Boolean);
  const at = segments.indexOf('onboarding');
  if (at === -1) return null;
  const route = `/onboarding/${segments[at + 1] ?? ''}`;
  const entry = (Object.entries(STEP_ROUTES) as [OnboardingStepKey, string][]).find(
    ([, value]) => value === route,
  );
  return entry?.[0] ?? null;
}

/** Is `key` a known step? Guards values read back out of the resume cookie. */
export function isOnboardingStep(key: string): key is OnboardingStepKey {
  // `in` would also accept inherited keys ('toString', 'constructor'), and this
  // value arrives from a cookie the user can edit.
  return Object.hasOwn(STEP_ROUTES, key);
}

/** 1-based position and total, for the "N / M" header. */
export function stepPosition(
  key: OnboardingStepKey,
  intention: PregnancyIntention | null,
): { index: number; total: number } {
  const steps = onboardingSteps(intention);
  return { index: steps.indexOf(key) + 1, total: steps.length };
}

/**
 * The route the `NavBack` arrow goes to, or `null` on the first step — the flow
 * starts at the phone number, which is not part of this sequence.
 *
 * The arrow navigates forwards to a known route instead of calling
 * `router.back()`, because browser/hardware back is trapped app-wide
 * (`shared/back-guard`) and would do nothing.
 */
export function previousOnboardingRoute(
  key: OnboardingStepKey,
  intention: PregnancyIntention | null,
): string | null {
  const steps = onboardingSteps(intention);
  const at = steps.indexOf(key);
  // A key outside the current flow (`intention`/`pregnancyBasis` while
  // pregnancy is postponed, but their routes still exist) would index -2 and
  // read as "first step", sending a signed-in user back out to /signup.
  if (at === -1) return steps.length > 0 ? STEP_ROUTES[steps[0]] : null;
  const previous = steps[at - 1];
  return previous ? STEP_ROUTES[previous] : null;
}

/** The route to advance to after `key`, or the setting-up screen when last. */
export function nextOnboardingRoute(
  key: OnboardingStepKey,
  intention: PregnancyIntention | null,
): string {
  const steps = onboardingSteps(intention);
  const at = steps.indexOf(key);
  // Same -1 trap as above: it would otherwise resolve to steps[0], silently
  // restarting the flow instead of continuing it.
  if (at === -1) return steps.length > 0 ? STEP_ROUTES[steps[0]] : SETTING_UP_ROUTE;
  const next = steps[at + 1];
  return next ? STEP_ROUTES[next] : SETTING_UP_ROUTE;
}
