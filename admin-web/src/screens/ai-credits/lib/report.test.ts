import { describe, expect, it } from 'vitest';

import { lastDays, mergeDays, milliToUsd, usdToMilli, usdToToman } from './report';

describe('mergeDays', () => {
  it('fills every day of the window and joins cost with credits', () => {
    const days = mergeDays(
      '2026-11-09',
      '2026-11-11',
      [{ day: '2026-11-10', calls: 4, cost_usd: 0.5 }],
      [
        { day: '2026-11-09', credits: 3 },
        { day: '2026-11-10', credits: 7 },
      ],
    );
    expect(days).toEqual([
      { day: '2026-11-09', calls: 0, costUsd: 0, credits: 3 },
      { day: '2026-11-10', calls: 4, costUsd: 0.5, credits: 7 },
      { day: '2026-11-11', calls: 0, costUsd: 0, credits: 0 },
    ]);
  });

  it('is empty for an invalid window', () => {
    expect(mergeDays('2026-11-12', '2026-11-11', [], [])).toEqual([]);
    expect(mergeDays('x', '2026-11-11', [], [])).toEqual([]);
  });
});

describe('money and prices', () => {
  it('converts USD to toman only with a rate', () => {
    expect(usdToToman(2, 650000)).toBe(130000);
    expect(usdToToman(2, null)).toBeNull();
  });

  it('converts prices between USD per 1M tokens and milli-USD', () => {
    expect(milliToUsd(2500)).toBe(2.5);
    expect(usdToMilli('2.5')).toBe(2500);
    expect(usdToMilli('0.075')).toBe(75);
    expect(usdToMilli('')).toBe(0);
    expect(usdToMilli('1.2345')).toBeNull();
    expect(usdToMilli('-1')).toBeNull();
  });

  it('computes a trailing window', () => {
    expect(lastDays('2026-11-12', 7)).toEqual({ from: '2026-11-06', to: '2026-11-12' });
  });
});
