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
  it('has no fertile or luteal range without an engine ovulation (no length − 14 guess)', () => {
    expect(phaseDayRange(view('luteal', null))).toBeNull();
    expect(phaseDayRange(view('fertile', null))).toBeNull();
    expect(phaseDayRange(view('follicular', null))).toEqual({ from: 6, to: 29 });
  });

  // SPEC §2.36: O = L − 14, fertile = O−5 … O (kept after the period's end).
  it.each([
    { L: 21, O: 7, fertile: { from: 6, to: 7 } },
    { L: 28, O: 14, fertile: { from: 9, to: 14 } },
    { L: 35, O: 21, fertile: { from: 16, to: 21 } },
  ])('matches the §2.36 table for a $L-day cycle', ({ L, O, fertile }) => {
    const start = '2026-09-01';
    const ov = new Date(Date.UTC(2026, 8, O)).toISOString().slice(0, 10);
    const v = (mainPhase: CycleView['mainPhase']) =>
      ({
        mainPhase,
        anchors: { currentPeriodStart: start, estimatedOvulationDate: ov },
        metrics: { effectiveCycleLength: L, effectivePeriodLength: 5, cycleVariability: null },
      }) as unknown as CycleView;
    expect(phaseDayRange(v('fertile'))).toEqual(fertile);
    expect(phaseDayRange(v('luteal'))).toEqual({ from: O + 1, to: L });
  });
  it('has no range for an expected period or unknown phase', () => {
    expect(phaseDayRange(view('period_expected'))).toBeNull();
    expect(phaseDayRange(null)).toBeNull();
  });
});
