import type { PregnancyIntention } from './types';

/**
 * Onboarding is a set of independent full-page route screens rather than a
 * component wizard. This module is the single source of truth for their order
 * so the flow can **branch** on pregnancy intention without every screen
 * hardcoding its own "next" route or step number (which is what made the old
 * fixed 8-step flow brittle).
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

const HEAD: OnboardingStepKey[] = ['name', 'birthday', 'weight', 'height', 'intention'];

/**
 * The ordered step keys for the given intention. Before the intention step is
 * answered (`null`), we assume the cycle branch so the shared head steps still
 * have a stable position/total to display.
 */
export function onboardingSteps(intention: PregnancyIntention | null): OnboardingStepKey[] {
  if (intention === 'pregnant') return [...HEAD, 'pregnancyBasis', 'conditions'];
  return [...HEAD, 'periodLen', 'cycleDuration', 'cycleLen', 'conditions'];
}

/** The route for a step key. */
export function onboardingRoute(key: OnboardingStepKey): string {
  return STEP_ROUTES[key];
}

/** 1-based position and total, for the "N / M" header. */
export function stepPosition(
  key: OnboardingStepKey,
  intention: PregnancyIntention | null,
): { index: number; total: number } {
  const steps = onboardingSteps(intention);
  return { index: steps.indexOf(key) + 1, total: steps.length };
}

/** The route to advance to after `key`, or the setting-up screen when last. */
export function nextOnboardingRoute(
  key: OnboardingStepKey,
  intention: PregnancyIntention | null,
): string {
  const steps = onboardingSteps(intention);
  const next = steps[steps.indexOf(key) + 1];
  return next ? STEP_ROUTES[next] : SETTING_UP_ROUTE;
}
