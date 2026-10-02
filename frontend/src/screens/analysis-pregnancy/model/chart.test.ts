import { describe, expect, it } from 'vitest';

import { bandAt, bpChart, gainChart, gainUntilWeek, isUserRow, tileItems, weekdayKeyOf } from './chart';

const band = [
  { gaWeeks: 0, min: 0, max: 0 },
  { gaWeeks: 13, min: 0.5, max: 2 },
  { gaWeeks: 40, min: 11.5, max: 16 },
];

describe('bandAt', () => {
  it('interpolates the piecewise band and stays flat after term', () => {
    expect(bandAt(band, 0)).toEqual({ min: 0, max: 0 });
    expect(bandAt(band, 13)).toEqual({ min: 0.5, max: 2 });
    const mid = bandAt(band, 26.5);
    expect(mid?.min).toBeCloseTo(6, 5);
    expect(mid?.max).toBeCloseTo(9, 5);
    expect(bandAt(band, 42)).toEqual({ min: 11.5, max: 16 });
    expect(bandAt([], 10)).toBeNull();
  });
});

describe('gainChart', () => {
  it('draws the band, the line and the axis ticks left to right', () => {
    const g = gainChart({
      band,
      points: [
        { date: '2026-04-01', gaDays: 42, week: 7, weight: 60.5, gain: 0.5 },
        { date: '2026-09-20', gaDays: 212, week: 31, weight: 68.4, gain: 8.4 },
      ],
    });
    expect(g.band.startsWith('M')).toBe(true);
    expect(g.band.endsWith('Z')).toBe(true);
    expect(g.line.split('L')).toHaveLength(2);
    expect(g.ticks.map((t) => t.week)).toEqual([0, 13, 28, 40]);
    expect(g.ticks[0].x).toBeLessThan(g.ticks[3].x);
    expect(g.end?.x).toBeGreaterThan(g.ticks[2].x);
  });

  it('copes without a band or points', () => {
    const g = gainChart({ band: [], points: [] });
    expect(g.band).toBe('');
    expect(g.line).toBe('');
    expect(g.end).toBeNull();
  });
});

describe('helpers', () => {
  it('finds the weekday of an API date', () => {
    expect(weekdayKeyOf('2026-09-23')).toBe('wed');
    expect(weekdayKeyOf('2026-09-19')).toBe('sat');
  });
  it('marks the user row and keeps two tile items', () => {
    expect(isUserRow({ category: 'normal' }, 'normal')).toBe(true);
    expect(isUserRow({ category: 'normal' }, null)).toBe(false);
    expect(tileItems([1, 2, 3])).toEqual([1, 2]);
  });
});

describe('bpChart', () => {
  it('keeps a fixed clinical domain and marks high readings', () => {
    const g = bpChart(
      [
        { systolic: 110, high: false },
        { systolic: 114, high: false },
        { systolic: 145, high: true },
      ],
      140,
    );
    expect(g.highs).toHaveLength(1);
    expect(g.thresholdY).toBeGreaterThan(g.highs[0].y);
    const flat = bpChart([{ systolic: 110, high: false }, { systolic: 112, high: false }], 140);
    const ys = flat.line.match(/ (\d+(\.\d+)?)/g)?.map(Number) ?? [];
    expect(Math.abs(ys[0] - ys[1])).toBeLessThan(5);
    expect(flat.thresholdY).toBeLessThan(ys[0]);
  });
});

describe('gainUntilWeek', () => {
  it('names the week only when the weigh-in is in today\'s week (B-N3-14b)', () => {
    expect(gainUntilWeek(23, 23)).toBe(true);
    expect(gainUntilWeek(22, 23)).toBe(false);
    expect(gainUntilWeek(0, 23)).toBe(true);
  });
});
