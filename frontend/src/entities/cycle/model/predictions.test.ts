import { describe, expect, it } from 'vitest';

import { deriveCyclePredictions, normalizePhase } from './predictions';
import type { CycleCalculation } from './types';

function makeCalc(overrides: Partial<CycleCalculation> = {}): CycleCalculation {
  return {
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
    ...overrides,
  };
}

describe('normalizePhase', () => {
  it('passes through known phases', () => {
    expect(normalizePhase('ovulation')).toBe('ovulation');
    expect(normalizePhase('period')).toBe('period');
  });

  it('falls back to luteal for unknown backend strings', () => {
    expect(normalizePhase('some_future_phase')).toBe('luteal');
    expect(normalizePhase('')).toBe('luteal');
  });
});

describe('deriveCyclePredictions', () => {
  it('derives day offsets from the calculation', () => {
    const p = deriveCyclePredictions(makeCalc());
    expect(p.daysUntilNextPeriod).toBe(20); // 28 - 8
    expect(p.daysUntilOvulation).toBe(6); // 14 - 8
    expect(p.daysUntilFertileWindow).toBe(2); // 6 - 4 lead days
    expect(p.fertilityPercent).toBe(20); // rounded
    expect(p.cycleDay).toBe(8);
    expect(p.cycleLength).toBe(28);
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
