import { describe, expect, it } from 'vitest';

import type { CycleCalculation, CycleView } from '@/entities/cycle';

import { needsPeriodData } from './cycle-data';

// Only the fields the schedule reads; the rest of the payload is irrelevant here.
const view = (currentPeriodStart: string | null) =>
  ({
    anchors: { currentPeriodStart, predictedNextPeriodStart: null, estimatedOvulationDate: null },
    forecast: null,
    metrics: { effectiveCycleLength: 28 },
    effectiveValues: { cycleLength: 28, periodDuration: 5 },
  }) as unknown as CycleView;

describe('needsPeriodData', () => {
  it('is true when the API has no period anchor and no cycle day (after leaving pregnancy)', () => {
    expect(needsPeriodData(view(null), null)).toBe(true);
    expect(needsPeriodData(null, null)).toBe(true);
  });

  it('is false once a period start is on record', () => {
    expect(needsPeriodData(view('2026-09-13'), null)).toBe(false);
  });

  it('is false when only the legacy calculation knows the cycle day', () => {
    const calc = { calculationDate: '2026-09-28', cycleDay: 16, cycleLength: 28 } as unknown as CycleCalculation;
    expect(needsPeriodData(null, calc)).toBe(false);
  });
});
