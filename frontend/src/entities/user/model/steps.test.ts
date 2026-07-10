import { describe, expect, it } from 'vitest';

import {
  nextOnboardingRoute,
  onboardingRoute,
  onboardingSteps,
  SETTING_UP_ROUTE,
  stepPosition,
} from './steps';

describe('onboardingSteps', () => {
  it('routes pregnant users to the dating basis, skipping cycle questions', () => {
    const steps = onboardingSteps('pregnant');
    expect(steps).toEqual(['name', 'birthday', 'weight', 'height', 'intention', 'pregnancyBasis', 'conditions']);
    expect(steps).not.toContain('periodLen');
    expect(steps).not.toContain('cycleLen');
  });

  it('routes non-pregnant users through the cycle questions', () => {
    for (const intention of ['avoiding', 'trying', 'unsure', null] as const) {
      const steps = onboardingSteps(intention);
      expect(steps).toEqual([
        'name', 'birthday', 'weight', 'height', 'intention',
        'periodLen', 'cycleDuration', 'cycleLen', 'conditions',
      ]);
      expect(steps).not.toContain('pregnancyBasis');
    }
  });
});

describe('stepPosition', () => {
  it('reports 1-based index and branch-dependent total', () => {
    expect(stepPosition('name', null)).toEqual({ index: 1, total: 9 });
    expect(stepPosition('conditions', 'pregnant')).toEqual({ index: 7, total: 7 });
    expect(stepPosition('conditions', 'avoiding')).toEqual({ index: 9, total: 9 });
  });
});

describe('nextOnboardingRoute', () => {
  it('advances through the shared head to the intention step', () => {
    expect(nextOnboardingRoute('name', null)).toBe(onboardingRoute('birthday'));
    expect(nextOnboardingRoute('height', null)).toBe(onboardingRoute('intention'));
  });

  it('branches after intention on the stated intention', () => {
    expect(nextOnboardingRoute('intention', 'pregnant')).toBe(onboardingRoute('pregnancyBasis'));
    expect(nextOnboardingRoute('intention', 'avoiding')).toBe(onboardingRoute('periodLen'));
  });

  it('lands on the setting-up screen after the last step of each branch', () => {
    expect(nextOnboardingRoute('conditions', 'pregnant')).toBe(SETTING_UP_ROUTE);
    expect(nextOnboardingRoute('conditions', 'avoiding')).toBe(SETTING_UP_ROUTE);
  });
});
