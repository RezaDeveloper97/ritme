import { describe, expect, it } from 'vitest';

import {
  clampWeek,
  derivePregnancyProgress,
  trimesterOfWeek,
} from './progress';

describe('trimesterOfWeek', () => {
  it('maps weeks to the standard obstetric trimesters', () => {
    expect(trimesterOfWeek(1)).toBe(1);
    expect(trimesterOfWeek(12)).toBe(1);
    expect(trimesterOfWeek(13)).toBe(2);
    expect(trimesterOfWeek(27)).toBe(2);
    expect(trimesterOfWeek(28)).toBe(3);
    expect(trimesterOfWeek(40)).toBe(3);
  });
});

describe('clampWeek', () => {
  it('keeps valid weeks and clamps out-of-range values into 1..40', () => {
    expect(clampWeek(20)).toBe(20);
    expect(clampWeek(0)).toBe(1);
    expect(clampWeek(-5)).toBe(1);
    expect(clampWeek(99)).toBe(40);
  });

  it('truncates fractional weeks and defends against NaN', () => {
    expect(clampWeek(12.9)).toBe(12);
    expect(clampWeek(Number.NaN)).toBe(1);
  });
});

describe('derivePregnancyProgress', () => {
  it('returns null before an active week exists', () => {
    expect(derivePregnancyProgress(null)).toBeNull();
    expect(derivePregnancyProgress(0)).toBeNull();
  });

  it('derives week, trimester, weeks remaining and percent for a mid-pregnancy week', () => {
    expect(derivePregnancyProgress(20)).toEqual({
      currentWeek: 20,
      trimester: 2,
      weeksRemaining: 20,
      progressPct: 50,
    });
  });

  it('never reports negative weeks remaining and caps at week 40', () => {
    const full = derivePregnancyProgress(40);
    expect(full).toEqual({
      currentWeek: 40,
      trimester: 3,
      weeksRemaining: 0,
      progressPct: 100,
    });

    const over = derivePregnancyProgress(45);
    expect(over?.currentWeek).toBe(40);
    expect(over?.weeksRemaining).toBe(0);
  });

  it('handles the very first week', () => {
    expect(derivePregnancyProgress(1)).toEqual({
      currentWeek: 1,
      trimester: 1,
      weeksRemaining: 39,
      progressPct: 3,
    });
  });
});
