import { describe, expect, it } from 'vitest';

import { phaseBar } from './phase-bar';

describe('phaseBar', () => {
  it('splits a 29-day cycle into period, follicular, fertile and luteal', () => {
    const bar = phaseBar(22, 29)!;
    expect(bar.segments.map((s) => s.days)).toEqual([5, 5, 5, 14]);
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
});
