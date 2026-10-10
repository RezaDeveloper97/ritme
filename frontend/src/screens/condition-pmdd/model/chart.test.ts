import { describe, expect, it } from 'vitest';

import type { PmddAlert, PmddCycle } from '@/entities/condition';

import { chartGeometry, cycleDayOn, daysBetween, pmddNotes } from './chart';

const cycle: PmddCycle = {
  start: '2026-09-15',
  end: '2026-10-12',
  closed: false,
  complete: false,
  period: { from: '2026-09-15', to: '2026-09-19' },
  lateLuteal: { from: '2026-10-03', to: '2026-10-12' },
  follicular: { from: '2026-09-18', to: '2026-09-24' },
  lutealMean: null,
  follicularMean: null,
  days: [
    { date: '2026-10-03', cycleDay: 19, inPeriod: false, lateLuteal: true, mean: 6 },
    { date: '2026-09-18', cycleDay: 4, inPeriod: true, lateLuteal: false, mean: 1 },
  ],
};

describe('chart model', () => {
  it('counts days across month ends', () => {
    expect(daysBetween('2026-09-15', '2026-10-12')).toBe(27);
    expect(cycleDayOn(cycle, '2026-10-08')).toBe(24);
    expect(cycleDayOn(cycle, '2026-09-01')).toBeNull();
    expect(cycleDayOn(undefined, '2026-10-08')).toBeNull();
  });

  it('places bands and points along the cycle', () => {
    const g = chartGeometry(cycle);
    expect(g.length).toBe(28);
    expect(g.period).toEqual({ x0: 0, x1: 5 / 28 });
    expect(g.lateLuteal).toEqual({ x0: 18 / 28, x1: 1 });
    expect(g.points).toEqual([
      { x: 3.5 / 28, y: 0 },
      { x: 18.5 / 28, y: 1 },
    ]);
  });

  it('sorts the alerts into slots', () => {
    const a = (code: string): PmddAlert => ({ code, title: code, body: null, severity: 'info', hotlines: [] });
    expect(pmddNotes([a('pmdd_pattern_luteal'), a('pmdd_crisis')])).toEqual({
      pattern: a('pmdd_pattern_luteal'),
      needsTwo: null,
      crisis: a('pmdd_crisis'),
    });
    expect(pmddNotes([a('pmdd_not_enough_data'), a('pmdd_needs_two_cycles')]).needsTwo?.code).toBe('pmdd_needs_two_cycles');
  });
});
