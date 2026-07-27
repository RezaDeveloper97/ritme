import { describe, expect, it } from 'vitest';

import { calcToPhase, cycleDayMarker, deriveCyclePredictions, normalizePhase } from './predictions';
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

describe('normalizePhase', () => {
  it('passes through known phases', () => {
    expect(normalizePhase('ovulation')).toBe('ovulation');
    expect(normalizePhase('period')).toBe('period');
  });

  it('maps the backend "menstruation" phase to "period"', () => {
    expect(normalizePhase('menstruation')).toBe('period');
  });

  it('falls back to luteal for unknown backend strings', () => {
    expect(normalizePhase('some_future_phase')).toBe('luteal');
    expect(normalizePhase('')).toBe('luteal');
  });
});

describe('calcToPhase', () => {
  it('surfaces the fertile window as its own phase', () => {
    expect(calcToPhase(makeCalc({ phase: 'follicular', isFertileWindow: true }))).toBe('fertile');
  });

  it('lets period and ovulation win over the fertile flag', () => {
    expect(calcToPhase(makeCalc({ phase: 'menstruation', isFertileWindow: true }))).toBe('period');
    expect(calcToPhase(makeCalc({ phase: 'ovulation', isFertileWindow: true }))).toBe('ovulation');
  });

  it('passes non-fertile phases through', () => {
    expect(calcToPhase(makeCalc({ phase: 'luteal', isFertileWindow: false }))).toBe('luteal');
    expect(calcToPhase(makeCalc({ phase: 'follicular', isFertileWindow: false }))).toBe('follicular');
  });
});

describe('cycleDayMarker', () => {
  it('marks period and ovulation from the phase', () => {
    expect(cycleDayMarker(makeCalc({ phase: 'menstruation' }))).toBe('period');
    expect(cycleDayMarker(makeCalc({ phase: 'ovulation' }))).toBe('ovulation');
  });

  it('marks the fertile and PMS windows', () => {
    expect(cycleDayMarker(makeCalc({ phase: 'follicular', isFertileWindow: true }))).toBe('fertile');
    expect(cycleDayMarker(makeCalc({ phase: 'luteal', isPmsWindow: true }))).toBe('pms');
  });

  it('prioritizes period/ovulation over the window flags', () => {
    expect(cycleDayMarker(makeCalc({ phase: 'ovulation', isFertileWindow: true }))).toBe('ovulation');
    expect(cycleDayMarker(makeCalc({ phase: 'menstruation', isPmsWindow: true }))).toBe('period');
  });

  it('returns null for a plain follicular/luteal day', () => {
    expect(cycleDayMarker(makeCalc({ phase: 'follicular' }))).toBeNull();
    expect(cycleDayMarker(makeCalc({ phase: 'luteal' }))).toBeNull();
  });
});

describe('deriveCyclePredictions', () => {
  it('derives day offsets from the calculation', () => {
    const p = deriveCyclePredictions(makeCalc());
    expect(p.daysUntilNextPeriod).toBe(21); // 28 - 8 + 1 (next period is cycle day 29)
    expect(p.daysUntilOvulation).toBe(6); // 14 - 8
    expect(p.daysUntilFertileWindow).toBe(1); // 6 - 5 lead days
    expect(p.fertilityPercent).toBe(20); // rounded
    expect(p.cycleDay).toBe(8);
    expect(p.cycleLength).toBe(28);
  });

  it('derives the PMS window as the days before the next period', () => {
    const p = deriveCyclePredictions(makeCalc()); // next period in 20 days
    expect(p.daysUntilPmsEnd).toBe(20); // day before next period
    expect(p.daysUntilPmsStart).toBe(17); // 4-day window
  });

  it('clamps the PMS window to today when a period is imminent', () => {
    const p = deriveCyclePredictions(makeCalc({ cycleDay: 28, cycleLength: 28 }));
    expect(p.daysUntilPmsEnd).toBe(0);
    expect(p.daysUntilPmsStart).toBe(0);
  });

  it('never reports a negative days-until-next-period', () => {
    const p = deriveCyclePredictions(makeCalc({ cycleDay: 30, cycleLength: 28 }));
    expect(p.daysUntilNextPeriod).toBe(0);
  });

  it('reports a negative ovulation offset once ovulation has passed', () => {
    const p = deriveCyclePredictions(
      makeCalc({ cycleDay: 20, estimatedOvulationDay: 14 }),
    );
    expect(p.daysUntilOvulation).toBe(-6);
  });

  it('clamps the fertility probability to 0–100', () => {
    expect(deriveCyclePredictions(makeCalc({ fertilityPercent: 140 })).fertilityPercent).toBe(100);
    expect(deriveCyclePredictions(makeCalc({ fertilityPercent: -5 })).fertilityPercent).toBe(0);
  });
});
