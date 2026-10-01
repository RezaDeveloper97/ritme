import { describe, expect, it } from 'vitest';

import type { PeriodReport } from '@/entities/analysis';

import { flowColumns, hasFlow, levelOf } from './period';

const report = (over: Partial<PeriodReport> = {}): PeriodReport => ({
  range: { key: '6m', from: '2026-03-26', to: '2026-09-23', days: 182 },
  ready: true,
  basedOnPeriods: 3,
  periodMax: 8,
  periodLength: { median: 5, status: 'normal' },
  peak: { day: 2, level: 'heavy' },
  spotting: { days: 0, cycles: 0, ofCycles: 3 },
  average: [
    { day: 1, score: 2, level: 'medium' },
    { day: 2, score: 3, level: 'heavy' },
  ],
  periods: [{ start: '2026-09-14', length: 6, closed: true, isCurrent: true, days: [] }],
  coSymptoms: { ready: false, periodsNeeded: 3, items: [] },
  ...over,
});

describe('analysis-period model', () => {
  it('maps flow to levels', () => {
    expect(levelOf('light')).toBe(1);
    expect(levelOf('very_heavy')).toBe(4);
    expect(levelOf(null)).toBeNull();
  });

  it('sizes the grid and detects flow', () => {
    expect(
      flowColumns(
        report({ periods: [{ start: 'x', length: 6, closed: true, isCurrent: false, days: Array.from({ length: 6 }, (_, i) => ({ day: i + 1, date: 'd', flow: null })) }] }),
      ),
    ).toBe(6);
    expect(hasFlow(report())).toBe(true);
    expect(hasFlow(report({ average: [{ day: 1, score: null, level: null }] }))).toBe(false);
  });
});
