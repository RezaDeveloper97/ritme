import { describe, expect, it } from 'vitest';

import { cycleDayInfo } from './phase';
import type { CycleConfig } from './types';

// A clean 28-day cycle starting 2025-01-01; period 5 days. Ovulation index =
// 28 - 14 = 14 (the 15th day). Fertile window = days 10…16 (indices 9…15).
const config: CycleConfig = {
  startDate: new Date(2025, 0, 1),
  cycleLength: 28,
  periodLength: 5,
};

const on = (y: number, m: number, d: number) => cycleDayInfo(new Date(y, m - 1, d), config);

describe('cycleDayInfo', () => {
  it('marks the first days as the period with low conception chance', () => {
    expect(on(2025, 1, 1)).toEqual({ cycleDay: 1, phase: 'period', pregnancyChance: 'low' });
    expect(on(2025, 1, 5)).toEqual({ cycleDay: 5, phase: 'period', pregnancyChance: 'low' });
  });

  it('enters the follicular phase after the period', () => {
    expect(on(2025, 1, 6)).toEqual({ cycleDay: 6, phase: 'follicular', pregnancyChance: 'low' });
  });

  it('flags the fertile window before ovulation', () => {
    expect(on(2025, 1, 14)).toEqual({ cycleDay: 14, phase: 'fertile', pregnancyChance: 'medium' });
  });

  it('flags ovulation day with the highest conception chance', () => {
    expect(on(2025, 1, 15)).toEqual({ cycleDay: 15, phase: 'ovulation', pregnancyChance: 'high' });
  });

  it('enters the luteal phase after ovulation', () => {
    expect(on(2025, 1, 20)).toEqual({ cycleDay: 20, phase: 'luteal', pregnancyChance: 'low' });
  });

  it('wraps forward into the next predicted cycle', () => {
    // Day 29 is day 1 of the next cycle → period again.
    expect(on(2025, 1, 29)).toEqual({ cycleDay: 1, phase: 'period', pregnancyChance: 'low' });
  });

  it('wraps backward into the previous cycle', () => {
    // The day before the start is the last day (28) of the prior cycle.
    expect(on(2024, 12, 31)).toEqual({ cycleDay: 28, phase: 'luteal', pregnancyChance: 'low' });
  });
});
