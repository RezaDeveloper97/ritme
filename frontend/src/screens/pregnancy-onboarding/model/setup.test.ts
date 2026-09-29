import { describe, expect, it } from 'vitest';

import {
  EMPTY_DATING,
  EMPTY_HISTORY,
  isDatingComplete,
  noneFirst,
  toggleCondition,
  toOnboardingInput,
  toPreviewInput,
} from './setup';

describe('toggleCondition', () => {
  it('makes «none» exclusive', () => {
    expect(toggleCondition(['diabetes'], 'none')).toEqual(['none']);
    expect(toggleCondition(['none'], 'diabetes')).toEqual(['diabetes']);
    expect(toggleCondition(['none'], 'none')).toEqual([]);
    expect(toggleCondition(['diabetes'], 'diabetes')).toEqual([]);
  });
});

describe('dating payloads', () => {
  it('requires the inputs of the chosen source', () => {
    expect(isDatingComplete(EMPTY_DATING)).toBe(false);
    expect(isDatingComplete({ ...EMPTY_DATING, source: 'manual', manualWeeks: 8 })).toBe(true);
    expect(toPreviewInput(EMPTY_DATING, 'en')).toBeNull();
  });

  it('builds the preview and onboarding bodies (Gregorian ISO)', () => {
    const d = { ...EMPTY_DATING, lmp: { year: 2026, month: 8, day: 1 } };
    expect(toPreviewInput(d, 'en')).toEqual({ source: 'lmp', lmp_date: '2026-08-01' });
    expect(toOnboardingInput(d, null, 'en')).toEqual({ age_source: 'lmp', lmp_date: '2026-08-01' });
  });

  it('adds only the history the user gave', () => {
    const d = { ...EMPTY_DATING, source: 'manual' as const, manualWeeks: 10, manualDays: 2 };
    expect(toOnboardingInput(d, EMPTY_HISTORY, 'en')).toEqual({ age_source: 'manual', manual_weeks: 10, manual_days: 2 });
    expect(
      toOnboardingInput(d, { ...EMPTY_HISTORY, highRisk: true, conditions: ['none'], rhFactor: 'negative' }, 'en'),
    ).toEqual({
      age_source: 'manual',
      manual_weeks: 10,
      manual_days: 2,
      has_high_risk_history: true,
      pre_existing_conditions: ['none'],
      rh_factor: 'negative',
    });
  });
});

describe('noneFirst', () => {
  it('puts «هیچ‌کدام» first', () => {
    expect(noneFirst(['diabetes', 'hypothyroidism', 'none'])).toEqual(['none', 'diabetes', 'hypothyroidism']);
    expect(noneFirst(['diabetes'])).toEqual(['diabetes']);
  });
});
