import { describe, expect, it } from 'vitest';

import { deriveDayHighlights } from './highlights';
import type { CycleCalculation } from './types';

const base: CycleCalculation = {
  calculationDate: '2026-07-24',
  cycleDay: 10,
  phase: 'follicular',
  subphase: null,
  estimatedOvulationDay: 15,
  cycleLength: 28,
  isFertileWindow: false,
  isPmsWindow: false,
  isPeriodTomorrow: false,
  fertilityPercent: 5,
  cycleVariability: null,
  dailyTips: [],
};

describe('deriveDayHighlights', () => {
  it('marks a period day', () => {
    expect(deriveDayHighlights({ ...base, phase: 'menstruation' })).toEqual(['period']);
  });

  it('prefers ovulation over the plain fertile badge', () => {
    expect(deriveDayHighlights({ ...base, phase: 'ovulation', isFertileWindow: true })).toEqual(['ovulation']);
  });

  it('shows fertile when in the window but not ovulating', () => {
    expect(deriveDayHighlights({ ...base, isFertileWindow: true })).toEqual(['fertile']);
  });

  it('stacks PMS and an imminent period in the luteal phase', () => {
    expect(deriveDayHighlights({ ...base, phase: 'luteal', isPmsWindow: true, isPeriodTomorrow: true }))
      .toEqual(['pms', 'period_tomorrow']);
  });

  it('returns nothing notable for a plain follicular day', () => {
    expect(deriveDayHighlights(base)).toEqual([]);
  });
});
