import { describe, expect, it } from 'vitest';

import type { LogDay } from '@/entities/health-log';

import type { EpdsHistory, EpdsPoint } from '../api/epds';
import { bleedingTrend, epdsGeometry, epdsTrend, hubChildId, hubRange, motherSleepAverage, weekAfterBirth, weightTrend } from './hub';

const TH = { shortElevated: 6, fullPossible: 10, fullLikely: 13 };
const check = (week: number, total: number, kind: 'full' | 'short' = 'full', urgent = false): EpdsPoint => ({
  kind,
  takenOn: `2026-09-${String(week).padStart(2, '0')}`,
  week,
  total,
  max: kind === 'full' ? 30 : 9,
  urgent,
});
const history = (checks: EpdsPoint[]): EpdsHistory => ({ checks: checks.slice().reverse(), thresholds: TH });
const day = (date: string, categories: LogDay['categories']): LogDay => ({ date, categories });

describe('epdsTrend', () => {
  it('returns null without checks', () => {
    expect(epdsTrend(history([]))).toBeNull();
  });

  it('plots the full checks oldest first and reports a peak that came down (the artboard)', () => {
    const tr = epdsTrend(history([check(1, 8), check(3, 9), check(5, 11), check(7, 9), check(9, 8), check(4, 3, 'short')]))!;
    expect(tr.kind).toBe('full');
    expect(tr.points.map((p) => p.week)).toEqual([1, 3, 5, 7, 9]);
    expect(tr.status).toBe('down');
    expect(tr.peak?.week).toBe(5);
    expect([tr.lower, tr.upper]).toEqual([10, 13]);
  });

  it('flags a latest score at the likely cut-off, or an urgent check, as urgent', () => {
    expect(epdsTrend(history([check(2, 6), check(4, 13)]))!.status).toBe('urgent');
    expect(epdsTrend(history([check(2, 4, 'full', true)]))!.status).toBe('urgent');
    expect(epdsTrend(history([check(2, 11)]))!.status).toBe('high');
    expect(epdsTrend(history([check(2, 4)]))!.status).toBe('low');
  });

  it('falls back to the short check with its single cut-off', () => {
    const tr = epdsTrend(history([check(2, 7, 'short')]))!;
    expect(tr.kind).toBe('short');
    expect(tr.upper).toBeNull();
    expect(tr.status).toBe('high');
  });

  it('draws the 13 band above the 10 band', () => {
    const tr = epdsTrend(history([check(1, 8), check(3, 12)]))!;
    const g = epdsGeometry(tr, 100);
    expect(g.upperY!).toBeLessThan(g.lowerY);
    expect(g.dots).toHaveLength(2);
    expect(g.dots[1].high).toBe(true);
  });
});

describe('bleedingTrend', () => {
  it('lays out the window oldest first, never before the birth', () => {
    const tr = bleedingTrend([], '2026-10-04', '2026-10-01');
    expect(tr.cells.map((c) => c.date)).toEqual(['2026-10-01', '2026-10-02', '2026-10-03', '2026-10-04']);
    expect(tr.trend).toBeNull();
  });

  it('reads a decreasing lochia and large clots', () => {
    const amounts = ['heavy', 'heavy', 'medium', 'medium', 'light', 'spotting', 'none'];
    const days = amounts.map((a, i) => day(`2026-10-0${i + 1}`, { bleeding: { lochia_amount: a } }));
    days[1] = day('2026-10-02', { bleeding: { lochia_amount: 'heavy', clot_size: 'large' } });
    const tr = bleedingTrend(days, '2026-10-07', '2026-09-20');
    expect(tr.cells).toHaveLength(18);
    expect(tr.logged).toBe(7);
    expect(tr.trend).toBe('decreasing');
    expect(tr.largeClots).toBe(true);
    expect(tr.cells.at(-1)).toEqual({ date: '2026-10-07', level: 0, code: 'none' });
  });

  it('reads a steady pattern and ignores unknown codes', () => {
    const days = ['light', 'light', 'odd', 'light', 'light'].map((a, i) => day(`2026-10-0${i + 1}`, { bleeding: { lochia_amount: a } }));
    const tr = bleedingTrend(days, '2026-10-05', null);
    expect(tr.trend).toBe('steady');
    expect(tr.cells.find((c) => c.date === '2026-10-03')?.level).toBeNull();
    expect(tr.largeClots).toBe(false);
  });
});

describe('mother sleep and weight', () => {
  const days = [
    day('2026-09-01', { measurements: { weight: 72.4 } }),
    day('2026-09-20', { measurements: { weight: 69.1 }, sleep: { hours: 4 } }),
    day('2026-09-30', { sleep: { hours: 5.5 } }),
    day('2026-10-04', { measurements: { weight: 68 }, sleep: { hours: 6 } }),
  ];

  it('averages the logged hours of the last 7 days', () => {
    expect(motherSleepAverage(days, '2026-10-04')).toBeCloseTo(5.75);
    expect(motherSleepAverage([], '2026-10-04')).toBeNull();
  });

  it('measures the weight change since the first weigh-in after the birth', () => {
    const w = weightTrend(days, '2026-08-30')!;
    expect(w.points).toHaveLength(3);
    expect(w.delta).toBe(-4.4);
    expect(w.fromWeek).toBe(1);
    expect(weightTrend(days, '2026-09-10')!.fromWeek).toBe(2);
    expect(weightTrend([], null)).toBeNull();
  });
});

describe('helpers', () => {
  it('counts weeks after the birth from 1', () => {
    expect(weekAfterBirth('2026-08-30', '2026-08-30')).toBe(1);
    expect(weekAfterBirth('2026-09-06', '2026-08-30')).toBe(2);
  });

  it('reads from the birth, at most a year back', () => {
    expect(hubRange('2026-10-04', '2026-08-30')).toEqual({ from: '2026-08-30', to: '2026-10-04' });
    expect(hubRange('2026-10-04', '2024-01-01').from).toBe('2025-10-04');
    expect(hubRange('2026-10-04', null).from).toBe('2025-10-04');
  });

  it('picks the own child born closest to the birth', () => {
    const kids = [
      { id: 1, role: 'owner', birthDate: '2024-02-01' },
      { id: 2, role: 'owner', birthDate: '2026-08-31' },
      { id: 3, role: 'shared', birthDate: '2026-08-30' },
    ];
    expect(hubChildId(kids, '2026-08-30')).toBe(2);
    expect(hubChildId(kids, null)).toBe(1);
    expect(hubChildId([kids[2]], '2026-08-30')).toBeNull();
  });
});
