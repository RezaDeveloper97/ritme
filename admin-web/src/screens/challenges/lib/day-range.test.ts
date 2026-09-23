import { describe, expect, it } from 'vitest';

import { dayRange } from './day-range';

describe('dayRange', () => {
  it('classifies the cycle-day window', () => {
    expect(dayRange(null, null)).toEqual({ kind: 'all' });
    expect(dayRange(3, 3)).toEqual({ kind: 'single', day: 3 });
    expect(dayRange(3, 7)).toEqual({ kind: 'range', from: 3, to: 7 });
    expect(dayRange(3, null)).toEqual({ kind: 'from', from: 3 });
    expect(dayRange(null, 7)).toEqual({ kind: 'to', to: 7 });
  });
});
