import { describe, expect, it } from 'vitest';

import type { CycleReport } from '@/entities/analysis';

import { chronological, cycleVerdict, dayLayout, excludedCycles, periodVerdict, variationVerdict } from './cycle';

const base: CycleReport = {
  range: { key: '6m', from: '2026-03-26', to: '2026-09-23', days: 182 },
  ready: true,
  basedOnCycles: 4,
  figo: { applies: true, cycleMin: 24, cycleMax: 38, periodMax: 8, variationMax: 7 },
  cycleLength: { median: 29, status: 'normal' },
  periodLength: { median: 5, status: 'normal' },
  variation: { days: 2, max: 7, status: 'regular', cyclesNeeded: 3 },
  typical: { cycleLength: 29, ovulationDay: 16, fertileStartDay: 11, fertileEndDay: 16, days: { period: 5, follicular: 5, fertile: 6, luteal: 13 } },
  current: null,
  cycles: [
    { start: '2026-08-16', end: '2026-09-13', length: 29, periodDays: 5, inFigoRange: true, counted: true, excluded: null },
    { start: '2026-04-13', end: '2026-05-20', length: 38, periodDays: 5, inFigoRange: true, counted: false, excluded: 'outlier' },
  ],
};

describe('analysis-cycle model', () => {
  it('lays out a cycle like the engine', () => {
    expect(dayLayout(29, 5)).toEqual({ periodDays: 5, ovulationDay: 16, fertileStart: 11, fertileEnd: 16 });
    expect(dayLayout(22, 9)).toEqual({ periodDays: 9, ovulationDay: 10, fertileStart: 10, fertileEnd: 10 });
  });

  it('grades the tiles', () => {
    expect(cycleVerdict(base)).toBe('good');
    expect(periodVerdict(base)).toBe('good');
    expect(variationVerdict(base)).toBe('good');
    const teen = { ...base, figo: { ...base.figo, applies: false }, cycleLength: { median: 22, status: 'frequent' as const } };
    expect(cycleVerdict(teen)).toBe('none');
    expect(cycleVerdict({ ...base, cycleLength: { median: 41, status: 'infrequent' } })).toBe('warn');
    expect(variationVerdict({ ...base, variation: { ...base.variation, status: 'not_enough_data' } })).toBe('none');
  });

  it('orders and filters cycles', () => {
    expect(chronological(base.cycles).map((c) => c.length)).toEqual([38, 29]);
    expect(excludedCycles(base.cycles).map((c) => c.excluded)).toEqual(['outlier']);
  });
});
