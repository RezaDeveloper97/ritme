import { describe, expect, it } from 'vitest';

import { resolveSwipeIndex } from './swipe';

const base = { width: 300, index: 1, count: 3 };

describe('resolveSwipeIndex', () => {
  it('advances to the next slide on a decisive left drag', () => {
    // |deltaX| clears max(40, 300*0.15=45)
    expect(resolveSwipeIndex({ ...base, deltaX: -60 })).toBe(2);
  });

  it('goes back on a decisive right drag', () => {
    expect(resolveSwipeIndex({ ...base, deltaX: 60 })).toBe(0);
  });

  it('ignores jitters below the threshold', () => {
    expect(resolveSwipeIndex({ ...base, deltaX: -20 })).toBe(1);
    expect(resolveSwipeIndex({ ...base, deltaX: 20 })).toBe(1);
  });

  it('honours the absolute floor over a tiny width fraction', () => {
    // width*0.15 = 15, but the 40px floor wins, so a 30px drag is ignored.
    expect(resolveSwipeIndex({ ...base, width: 100, deltaX: -30 })).toBe(1);
    expect(resolveSwipeIndex({ ...base, width: 100, deltaX: -50 })).toBe(2);
  });

  it('clamps at the first and last slide', () => {
    expect(resolveSwipeIndex({ ...base, index: 0, deltaX: 100 })).toBe(0);
    expect(resolveSwipeIndex({ ...base, index: 2, deltaX: -100 })).toBe(2);
  });

  it('is a no-op without a measurable width or with a single slide', () => {
    expect(resolveSwipeIndex({ ...base, width: 0, deltaX: -100 })).toBe(1);
    expect(resolveSwipeIndex({ ...base, count: 1, index: 0, deltaX: -100 })).toBe(0);
  });
});
