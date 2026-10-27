import { describe, expect, it } from 'vitest';

import { phaseBar } from './phase-bar';

const days = (L: number, engine = {}) => phaseBar(3, L, engine)!.segments.map((s) => s.days);

describe('phaseBar', () => {
  it('splits a 29-day cycle into period, follicular, fertile (O−5…O) and luteal', () => {
    const bar = phaseBar(22, 29)!;
    expect(bar.segments.map((s) => s.days)).toEqual([5, 4, 6, 14]);
    expect(bar.segments.reduce((sum, s) => sum + s.days, 0)).toBe(29);
    expect(bar.markerPercent).toBeCloseTo(74.1, 1);
  });

  it('keeps the segments summing to the cycle length for a short cycle', () => {
    const bar = phaseBar(3, 21)!;
    expect(bar.segments.reduce((sum, s) => sum + s.days, 0)).toBe(21);
    expect(bar.segments.every((s) => s.days >= 0)).toBe(true);
  });

  it('clamps a late cycle day to the end of the bar', () => {
    expect(phaseBar(40, 28)!.markerPercent).toBeLessThan(100);
  });

  it('has no bar without a usable length', () => {
    expect(phaseBar(5, 0)).toBeNull();
    expect(phaseBar(0, 28)).toBeNull();
  });

  // SPEC §2.36: O = L − 14, fertile = O−5 … O (clamped to start after the period).
  it.each([
    { L: 21, expected: [5, 0, 2, 14] },
    { L: 28, expected: [5, 3, 6, 14] },
    { L: 35, expected: [5, 10, 6, 14] },
  ])('matches the §2.36 table for a $L-day cycle', ({ L, expected }) => {
    expect(days(L)).toEqual(expected);
    expect(days(L).reduce((a, b) => a + b, 0)).toBe(L);
  });

  it('uses her engine period length and ovulation day', () => {
    expect(days(28, { periodLength: 7, ovulationDay: 15 })).toEqual([7, 2, 6, 13]);
  });

  it('draws no fertile segment under clinical suppression', () => {
    const d = days(28, { periodLength: 6, fertilitySuppressed: true });
    expect(d[2]).toBe(0);
    expect(d[3]).toBe(0);
    expect(d.reduce((a, b) => a + b, 0)).toBe(28);
  });
});
