import { describe, expect, it } from 'vitest';

import type { CycleView } from '@/entities/cycle';

import { availableTabs, phaseDayRange } from './tabs';

const view = (mainPhase: CycleView['mainPhase'], ovulation: string | null = '2026-09-28') =>
  ({
    mainPhase,
    anchors: { currentPeriodStart: '2026-09-14', estimatedOvulationDate: ovulation },
    metrics: { effectiveCycleLength: 29, effectivePeriodLength: 5, cycleVariability: null },
  }) as unknown as CycleView;

describe('availableTabs', () => {
  it('keeps only tabs with copy, in order', () => {
    expect(availableTabs({ nutrition: 'x', hormonal_changes: 'y', sex_tips: ' ' })).toEqual(['body', 'nutrition']);
  });
});

describe('phaseDayRange', () => {
  it('places each main phase on the cycle', () => {
    expect(phaseDayRange(view('menstrual'))).toEqual({ from: 1, to: 5 });
    expect(phaseDayRange(view('follicular'))).toEqual({ from: 6, to: 9 });
    expect(phaseDayRange(view('fertile'))).toEqual({ from: 10, to: 15 });
    expect(phaseDayRange(view('luteal'))).toEqual({ from: 16, to: 29 });
  });
  it('falls back to length − 14 without an ovulation anchor', () => {
    expect(phaseDayRange(view('luteal', null))).toEqual({ from: 16, to: 29 });
  });
  it('has no range for an expected period or unknown phase', () => {
    expect(phaseDayRange(view('period_expected'))).toBeNull();
    expect(phaseDayRange(null)).toBeNull();
  });
});
