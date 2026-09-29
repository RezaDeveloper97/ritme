import { describe, expect, it } from 'vitest';

import { chanceFraction, ttcPhase } from './ttc';

describe('ttc-today model', () => {
  it('maps the engine main phase onto the legend', () => {
    expect(ttcPhase('menstrual')).toBe('period');
    expect(ttcPhase('follicular')).toBe('follicular');
    expect(ttcPhase('fertile')).toBe('fertile');
    expect(ttcPhase('luteal')).toBe('luteal');
    expect(ttcPhase('period_expected')).toBe('luteal');
    expect(ttcPhase('unknown')).toBeNull();
    expect(ttcPhase(null)).toBeNull();
  });

  it('fills the donut by level, empty when unknown', () => {
    expect(chanceFraction('high')).toBe(0.8);
    expect(chanceFraction('peak')).toBe(1);
    expect(chanceFraction('low')).toBeLessThan(chanceFraction('medium'));
    expect(chanceFraction('unknown')).toBe(0);
    expect(chanceFraction(null)).toBe(0);
  });
});
