import { describe, expect, it, vi } from 'vitest';

import { EMPTY_HISTORY, finishPregnancySetup, noneFirst, toggleCondition, withHistory } from './setup';

describe('toggleCondition', () => {
  it('makes «none» exclusive', () => {
    expect(toggleCondition(['diabetes'], 'none')).toEqual(['none']);
    expect(toggleCondition(['none'], 'diabetes')).toEqual(['diabetes']);
    expect(toggleCondition(['none'], 'none')).toEqual([]);
    expect(toggleCondition(['diabetes'], 'diabetes')).toEqual([]);
  });
});

describe('withHistory', () => {
  const body = { age_source: 'manual' as const, manual_weeks: 10, manual_days: 2 };

  it('keeps the dating body as is without history', () => {
    expect(withHistory(body, null)).toEqual(body);
    expect(withHistory(body, EMPTY_HISTORY)).toEqual(body);
  });

  it('adds only the history the user gave', () => {
    expect(withHistory(body, { ...EMPTY_HISTORY, highRisk: true, conditions: ['none'], rhFactor: 'negative' })).toEqual({
      ...body,
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

describe('finishPregnancySetup (stage B-3)', () => {
  it('stores the mode only after activate and onboarding succeed', async () => {
    const calls: string[] = [];
    const step = (name: string) => vi.fn(async () => void calls.push(name));
    await finishPregnancySetup({ activate: step('activate'), onboard: step('onboard'), storeMode: step('storeMode') });
    expect(calls).toEqual(['activate', 'onboard', 'storeMode']);
  });

  it('never stores the mode when onboarding fails', async () => {
    const storeMode = vi.fn(async () => undefined);
    await expect(
      finishPregnancySetup({
        activate: async () => undefined,
        onboard: async () => {
          throw new Error('422');
        },
        storeMode,
      }),
    ).rejects.toThrow('422');
    expect(storeMode).not.toHaveBeenCalled();
  });
});
