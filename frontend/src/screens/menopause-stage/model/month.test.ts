import { describe, expect, it } from 'vitest';

import { canStepForward, stepMonth } from './month';

const now = { year: 1405, month: 7 };

describe('last-period month picker', () => {
  it('starts from the current month when unset', () => {
    expect(stepMonth(null, now, -1)).toEqual({ year: 1405, month: 6 });
    expect(stepMonth(null, now, 1)).toEqual(now);
  });

  it('steps back across the year', () => {
    expect(stepMonth({ year: 1405, month: 1 }, now, -1)).toEqual({ year: 1404, month: 12 });
  });

  it('never goes past the current month', () => {
    expect(stepMonth(now, now, 1)).toEqual(now);
    expect(canStepForward(now, now)).toBe(false);
    expect(canStepForward({ year: 1405, month: 6 }, now)).toBe(true);
    expect(canStepForward(null, now)).toBe(false);
  });
});
