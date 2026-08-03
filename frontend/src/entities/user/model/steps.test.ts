import { describe, expect, it } from 'vitest';

import {
  nextOnboardingRoute,
  onboardingRoute,
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
