import { describe, expect, it, vi } from 'vitest';

import type { PregnancyBasis } from '@/entities/user';

import { buildPregnancyOnboarding, planSetup, runSetup, type SetupStep } from './plan';

const noBasis: PregnancyBasis = {
  source: null,
  lmp: null,
  ultrasoundDate: null,
  ultrasoundWeeks: null,
  ultrasoundDays: null,
  manualWeeks: null,
  manualDays: null,
};
const manual: PregnancyBasis = { ...noBasis, source: 'manual', manualWeeks: 9, manualDays: 2 };

describe('planSetup', () => {
  it('never saves when the intention is missing (lost / reset store)', () => {
    expect(planSetup({ intention: null, pregnancyBasis: manual, locale: 'fa' })).toEqual({
      kind: 'resume',
      route: '/onboarding/intention',
    });
  });

  it('saves only the profile on the cycle path', () => {
    for (const intention of ['trying', 'avoiding', 'unsure'] as const) {
      expect(planSetup({ intention, pregnancyBasis: noBasis, locale: 'fa' })).toEqual({
        kind: 'save',
        steps: ['profile'],
        pregnancy: null,
        landing: '/home',
      });
    }
  });

  it('always activates and dates a pregnant signup', () => {
    expect(planSetup({ intention: 'pregnant', pregnancyBasis: manual, locale: 'fa' })).toEqual({
      kind: 'save',
      steps: ['profile', 'activate', 'onboarding'],
      pregnancy: { age_source: 'manual', manual_weeks: 9, manual_days: 2 },
      landing: '/pregnancy',
    });
  });

  it('sends a pregnant signup without a usable basis back to the basis step', () => {
    const incomplete: PregnancyBasis[] = [
      noBasis,
      { ...noBasis, source: 'lmp' },
      { ...noBasis, source: 'ultrasound', ultrasoundWeeks: 8 },
      { ...noBasis, source: 'manual' },
    ];
    for (const pregnancyBasis of incomplete) {
      expect(planSetup({ intention: 'pregnant', pregnancyBasis, locale: 'fa' })).toEqual({
        kind: 'resume',
        route: '/onboarding/pregnancy-basis',
      });
    }
  });
});

describe('buildPregnancyOnboarding', () => {
  it('converts the stored calendar parts to a Gregorian API date', () => {
    const lmp = { year: 1405, month: 5, day: 2 };
    expect(buildPregnancyOnboarding({ ...noBasis, source: 'lmp', lmp }, 'fa')).toEqual({
      age_source: 'lmp',
      lmp_date: '2026-07-24',
    });
    expect(
      buildPregnancyOnboarding(
        { ...noBasis, source: 'ultrasound', ultrasoundDate: { year: 2026, month: 9, day: 1 }, ultrasoundWeeks: 8 },
        'en',
      ),
    ).toEqual({ age_source: 'ultrasound', ultrasound_date: '2026-09-01', ultrasound_weeks: 8, ultrasound_days: 0 });
  });
});

describe('runSetup', () => {
  it('stops at the first failure and resumes from it on retry', async () => {
    const steps: SetupStep[] = ['profile', 'activate', 'onboarding'];
    const done = new Set<SetupStep>();
    const calls: SetupStep[] = [];
    let failActivate = true;
    const run = vi.fn(async (step: SetupStep) => {
      calls.push(step);
      if (step === 'activate' && failActivate) throw new Error('offline');
    });

    await expect(runSetup(steps, run, done)).rejects.toThrow('offline');
    expect([...done]).toEqual(['profile']);

    failActivate = false;
    await runSetup(steps, run, done);
    expect(calls).toEqual(['profile', 'activate', 'activate', 'onboarding']);
    expect([...done]).toEqual(steps);
  });
});
