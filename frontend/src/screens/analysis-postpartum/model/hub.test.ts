import { describe, expect, it } from 'vitest';

import type { EpdsHistory, EpdsPoint } from '../api/epds';
import { epdsGeometry, epdsTrend } from './hub';

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
