import { describe, expect, it } from 'vitest';

import { barGeometry, barScale, heatLevel, heatRuns, typicalBands } from './bars';
import type { CycleHistoryRow } from './schema';

const row = (length: number, periodDays = 5, isCurrent = false): CycleHistoryRow => ({
  start: '2026-01-01',
  end: null,
  periodDays,
  periodOngoing: false,
  length,
  isCurrent,
  inRange: null,
});

describe('barScale', () => {
  it('never drops below the minimum scale', () => {
    expect(barScale([row(28)], 28)).toBe(35);
  });
  it('fits the longest cycle', () => {
    expect(barScale([row(29), row(38)], 29)).toBe(38);
  });
});

describe('barGeometry', () => {
  it('scales the bleed and the cycle against the track', () => {
    expect(barGeometry(row(35, 7), 35)).toEqual({ period: 20, cycle: 100 });
  });
  it('never lets the bleed overrun the cycle', () => {
    expect(barGeometry(row(2, 5, true), 35).period).toBeCloseTo((2 / 35) * 100);
  });
});

describe('heat strips', () => {
  it('buckets values into four levels', () => {
    expect([0, 0.2, 0.5, 0.9].map(heatLevel)).toEqual([0, 1, 2, 3]);
  });
  it('merges same-level neighbours into runs', () => {
    expect(heatRuns([0, 0.9, 0.9, 0.5, 0, 0.2])).toEqual([
      { level: 3, start: 1, length: 2 },
      { level: 2, start: 3, length: 1 },
      { level: 1, start: 5, length: 1 },
    ]);
  });
});

describe('typicalBands', () => {
  it('derives the bands of a 28-day cycle', () => {
    expect(typicalBands(28, 5, 14)).toEqual({ period: [1, 5], fertile: [9, 14], ovulation: 14, pms: [26, 28] });
  });

  // SPEC §2.36: O = L − 14, fertile = O−5 … O (never O+1), PMS = last 3 days.
  it.each([
    { L: 21, O: 7, fertile: [2, 7], pms: [19, 21] },
    { L: 28, O: 14, fertile: [9, 14], pms: [26, 28] },
    { L: 35, O: 21, fertile: [16, 21], pms: [33, 35] },
  ])('matches the §2.36 table for a $L-day cycle', ({ L, O, fertile, pms }) => {
    const b = typicalBands(L, 5, O);
    expect(b.fertile).toEqual(fertile);
    expect(b.pms).toEqual(pms);
    expect(b.ovulation).toBe(O);
  });
});
