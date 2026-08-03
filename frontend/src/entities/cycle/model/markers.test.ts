import { describe, expect, it } from 'vitest';

import { markerIntensityByDate } from './markers';
import type { CycleCalculation } from './types';

function makeCalc(overrides: Partial<CycleCalculation> = {}): CycleCalculation {
  return {
    calculationDate: '2026-01-08',
    cycleDay: 8,
    phase: 'follicular',
    subphase: null,
    estimatedOvulationDay: 14,
    cycleLength: 28,
    isFertileWindow: false,
    isPmsWindow: false,
    isPeriodTomorrow: false,
    fertilityPercent: 19.89,
    cycleVariability: 'regular',
    dailyTips: [],
    ...overrides,
  };
}

describe('markerIntensityByDate', () => {
  it('grades days within a marker group from faint to strong by fertilityPercent', () => {
    const map = markerIntensityByDate([
      makeCalc({ calculationDate: '2026-01-09', phase: 'follicular', isFertileWindow: true, fertilityPercent: 10 }),
      makeCalc({ calculationDate: '2026-01-10', phase: 'follicular', isFertileWindow: true, fertilityPercent: 15 }),
      makeCalc({ calculationDate: '2026-01-11', phase: 'follicular', isFertileWindow: true, fertilityPercent: 22 }),
    ]);
    expect(map.get('2026-01-09')).toBe('faint');
    expect(map.get('2026-01-10')).toBe('medium');
    expect(map.get('2026-01-11')).toBe('strong');
  });

  it('leaves period and PMS days ungraded (base tint)', () => {
    const map = markerIntensityByDate([
      makeCalc({ calculationDate: '2026-01-01', phase: 'period', fertilityPercent: 1 }),
      makeCalc({ calculationDate: '2026-01-03', phase: 'period', fertilityPercent: 3 }),
      makeCalc({ calculationDate: '2026-01-25', phase: 'luteal', isPmsWindow: true, fertilityPercent: 4 }),
      makeCalc({ calculationDate: '2026-01-27', phase: 'luteal', isPmsWindow: true, fertilityPercent: 8 }),
    ]);
    // Absent from the map → consumers fall back to 'medium', the base tint.
    expect(map.size).toBe(0);
  });

  it('falls back to the graded days\' overall range for a group with no spread', () => {
    const map = markerIntensityByDate([
      makeCalc({ calculationDate: '2026-01-12', phase: 'follicular', isFertileWindow: true, fertilityPercent: 20 }),
      makeCalc({ calculationDate: '2026-01-14', phase: 'ovulation', fertilityPercent: 33 }),
    ]);
    // The lone ovulation day sits at the top of the overall graded range.
    expect(map.get('2026-01-14')).toBe('strong');
  });

  it('uses medium when every graded probability is effectively identical', () => {
    const map = markerIntensityByDate([
      makeCalc({ calculationDate: '2026-01-12', phase: 'follicular', isFertileWindow: true, fertilityPercent: 30 }),
      makeCalc({ calculationDate: '2026-01-13', phase: 'follicular', isFertileWindow: true, fertilityPercent: 30.1 }),
    ]);
    expect(map.get('2026-01-12')).toBe('medium');
    expect(map.get('2026-01-13')).toBe('medium');
  });

  it('skips days without a marker', () => {
    const map = markerIntensityByDate([
      makeCalc({ calculationDate: '2026-01-08', phase: 'follicular', fertilityPercent: 12 }),
    ]);
    expect(map.size).toBe(0);
  });
});
