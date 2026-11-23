import { describe, expect, it } from 'vitest';

import { fertileLevel, fertileLevelAt } from './markers';
import type { CycleDayMarker } from './types';

describe('fertileLevel', () => {
  it('peaks on ovulation and fades toward the window edges', () => {
    expect(fertileLevel(0)).toBe(4);
    expect(fertileLevel(1)).toBe(3);
    expect(fertileLevel(2)).toBe(3);
    expect(fertileLevel(3)).toBe(2);
    expect(fertileLevel(4)).toBe(2);
    expect(fertileLevel(5)).toBe(1);
    expect(fertileLevel(-1)).toBe(1);
  });

  it('reads an unknown distance as the middle step', () => {
    expect(fertileLevel(null)).toBe(2);
  });
});

describe('fertileLevelAt', () => {
  // A window of five fertile days, ovulation on index 5, one fertile day after it.
  const run: (CycleDayMarker | null)[] = [null, 'fertile', 'fertile', 'fertile', 'fertile', 'ovulation', 'fertile', null];
  const at = (i: number) => fertileLevelAt((k) => run[i + k] ?? null);

  it('grades each day by its distance to ovulation', () => {
    expect(at(1)).toBe(2);
    expect(at(2)).toBe(2);
    expect(at(3)).toBe(3);
    expect(at(4)).toBe(3);
    expect(at(5)).toBe(4);
    expect(at(6)).toBe(1);
  });

  it('is null outside the fertile window', () => {
    expect(at(0)).toBeNull();
    expect(at(7)).toBeNull();
  });

  it('falls back to the middle step when ovulation is not in reach', () => {
    expect(fertileLevelAt((k) => (k === 0 ? 'fertile' : null))).toBe(2);
  });
});
