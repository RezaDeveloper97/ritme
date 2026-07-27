import { describe, expect, it } from 'vitest';

import { weeklyWellbeingTone, wellbeingTrend } from './tone';

describe('weeklyWellbeingTone', () => {
  it('treats "nothing logged" as its own tone, not a bad week', () => {
    expect(weeklyWellbeingTone(null)).toBe('none');
  });

  it('buckets scores from low to great', () => {
    expect(weeklyWellbeingTone(20)).toBe('low');
    expect(weeklyWellbeingTone(44)).toBe('low');
    expect(weeklyWellbeingTone(45)).toBe('mixed');
    expect(weeklyWellbeingTone(64)).toBe('mixed');
    expect(weeklyWellbeingTone(65)).toBe('good');
    expect(weeklyWellbeingTone(79)).toBe('good');
    expect(weeklyWellbeingTone(80)).toBe('great');
    expect(weeklyWellbeingTone(100)).toBe('great');
  });
});

describe('wellbeingTrend', () => {
  it('has no direction without a comparison', () => {
    expect(wellbeingTrend(null)).toBeNull();
  });

  it('ignores swings small enough to be noise', () => {
    expect(wellbeingTrend(4)).toBeNull();
    expect(wellbeingTrend(-4)).toBeNull();
  });

  it('reports the direction of a real change', () => {
    expect(wellbeingTrend(12)).toBe('up');
    expect(wellbeingTrend(-12)).toBe('down');
  });
});
