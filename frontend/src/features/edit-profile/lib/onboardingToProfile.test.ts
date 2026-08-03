import { describe, expect, it } from 'vitest';

import type { OnboardingData } from '@/entities/user';

import { onboardingToProfileInput } from './onboardingToProfile';

/** A fully-populated onboarding answer set with sensible in-range values. */
function baseData(overrides: Partial<OnboardingData> = {}): OnboardingData {
  return {
    phone: '09120000000',
    name: 'Sara',
    locale: 'fa',
    birth: { d: 15, m: 3, y: 1373 },
    weightUnit: 'kg',
    weight: 60,
    heightUnit: 'cm',
    height: 165,
    intention: 'avoiding',
    pregnancyBasis: {
      source: null,
      lmp: null,
      ultrasoundDate: null,
      ultrasoundWeeks: null,
      ultrasoundDays: null,
      manualWeeks: null,
      manualDays: null,
    },
    chronicConditions: [],
    periodLen: 5,
    cycleDuration: 28,
    lastPeriod: { year: 1403, month: 1, day: 10 },
    ...overrides,
  };
}

describe('onboardingToProfileInput', () => {
  it('maps the standard answers onto the API payload', () => {
    const payload = onboardingToProfileInput(baseData());

    expect(payload.name).toBe('Sara');
    expect(payload.period_duration).toBe(5);
    expect(payload.cycle_duration).toBe(28);
    expect(payload.weight).toBe(60);
    expect(payload.height).toBe(165);
    // Dates are serialized as Gregorian ISO for the API boundary (§7).
    expect(payload.birthday).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    expect(payload.last_period_start).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });

  // `birth.m` is a 1-based Jalali month. It used to hold the raw 0-based wheel
  // index, so every birthday was saved a month early — the picked year matched
  // but the month and day did not.
  it('keeps the picked Jalali birthday intact (month is 1-based)', () => {
    // ۲۵ دی ۱۳۷۳ === 1995-01-15.
    const payload = onboardingToProfileInput(baseData({ birth: { d: 25, m: 10, y: 1373 } }));
    expect(payload.birthday).toBe('1995-01-15');
  });

  it('converts imperial units to the API canonical units (kg, cm)', () => {
    const payload = onboardingToProfileInput(
      baseData({ weightUnit: 'lb', weight: 154, heightUnit: 'ft', height: 5.5 }),
    );

    // 154 lb / 2.205 ≈ 70 kg; 5.5 ft * 30.48 ≈ 168 cm.
    expect(payload.weight).toBe(70);
    expect(payload.height).toBe(168);
  });

  it('clamps out-of-range durations to the API validation bounds', () => {
    const payload = onboardingToProfileInput(
      baseData({ cycleDuration: 999, periodLen: 0 }),
    );

    expect(payload.cycle_duration).toBe(60);
    expect(payload.period_duration).toBe(1);
  });

  it('omits an empty name rather than sending a blank string', () => {
    const payload = onboardingToProfileInput(baseData({ name: '   ' }));
    expect(payload.name).toBeUndefined();
  });

  it('omits the last period when it was never picked', () => {
    const payload = onboardingToProfileInput(baseData({ lastPeriod: null }));
    expect(payload.last_period_start).toBeUndefined();
  });

  it('sends the intention and chronic conditions', () => {
    const payload = onboardingToProfileInput(
      baseData({ intention: 'trying', chronicConditions: ['pcos', 'diabetes'] }),
    );
    expect(payload.pregnancy_intention).toBe('trying');
    expect(payload.chronic_conditions).toEqual(['pcos', 'diabetes']);
  });

  it('omits cycle fields for pregnant users', () => {
    const payload = onboardingToProfileInput(baseData({ intention: 'pregnant' }));
    expect(payload.pregnancy_intention).toBe('pregnant');
    expect(payload.period_duration).toBeUndefined();
    expect(payload.cycle_duration).toBeUndefined();
    expect(payload.last_period_start).toBeUndefined();
  });
});
