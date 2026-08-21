import { describe, expect, it } from 'vitest';

import {
  isOnboardingStep,
  nextOnboardingRoute,
  onboardingRoute,
  onboardingStepFromPath,
  onboardingSteps,
  SETTING_UP_ROUTE,
  stepPosition,
} from './steps';

// Pregnancy mode is postponed and hidden from the frontend, so the intention
// question and the pregnant branch are disabled: every intention (including
// 'pregnant', which can no longer be chosen) walks the same cycle sequence.
// Restore the branch assertions in git history when pregnancy comes back.
const CYCLE_STEPS = [
  'name', 'birthday', 'weight', 'height',
  'periodLen', 'cycleDuration', 'cycleLen', 'conditions',
];

describe('onboardingSteps', () => {
  it('routes every user through the cycle questions, without pregnancy steps', () => {
    for (const intention of ['avoiding', 'pregnant', 'trying', 'unsure', null] as const) {
      const steps = onboardingSteps(intention);
      expect(steps).toEqual(CYCLE_STEPS);
      expect(steps).not.toContain('intention');
      expect(steps).not.toContain('pregnancyBasis');
    }
  });
});

describe('stepPosition', () => {
  it('reports 1-based index and total', () => {
    expect(stepPosition('name', null)).toEqual({ index: 1, total: 8 });
    expect(stepPosition('conditions', 'avoiding')).toEqual({ index: 8, total: 8 });
  });
});

describe('nextOnboardingRoute', () => {
  it('advances through the head straight into the cycle questions', () => {
    expect(nextOnboardingRoute('name', null)).toBe(onboardingRoute('birthday'));
    expect(nextOnboardingRoute('height', null)).toBe(onboardingRoute('periodLen'));
  });

  it('lands on the setting-up screen after the last step', () => {
    expect(nextOnboardingRoute('conditions', null)).toBe(SETTING_UP_ROUTE);
    expect(nextOnboardingRoute('conditions', 'avoiding')).toBe(SETTING_UP_ROUTE);
  });
});

// The resume gate reads a path (middleware) or writes one (the tracker), so
// both directions of the route↔key mapping have to hold for an interrupted
// signup to come back to the step it stopped on.
describe('onboardingStepFromPath', () => {
  it('reads the step key out of a locale-prefixed path', () => {
    expect(onboardingStepFromPath('/fa/onboarding/period-len')).toBe('periodLen');
    expect(onboardingStepFromPath('/en/onboarding/name')).toBe('name');
  });

  it('accepts a path with no locale prefix', () => {
    expect(onboardingStepFromPath('/onboarding/height')).toBe('height');
  });

  it('returns null outside the flow — including the save screen, which is not a step', () => {
    expect(onboardingStepFromPath('/fa/home')).toBeNull();
    expect(onboardingStepFromPath('/fa/onboarding/setting-up')).toBeNull();
    expect(onboardingStepFromPath('/fa/onboarding')).toBeNull();
  });

  it('round-trips every step key through its route', () => {
    for (const key of onboardingSteps(null)) {
      expect(onboardingStepFromPath(`/fa${onboardingRoute(key)}`)).toBe(key);
    }
  });
});

describe('isOnboardingStep', () => {
  it('accepts known keys and rejects anything else read back from the cookie', () => {
    expect(isOnboardingStep('name')).toBe(true);
    expect(isOnboardingStep('cycleLen')).toBe(true);
    expect(isOnboardingStep('')).toBe(false);
    expect(isOnboardingStep('settingUp')).toBe(false);
    expect(isOnboardingStep('toString')).toBe(false);
  });
});
