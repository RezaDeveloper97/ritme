import { describe, expect, it } from 'vitest';

import { resolveSwipeIndex, wrapIndex } from './carousel';

const base = { width: 300, index: 1, count: 3 };

describe('resolveSwipeIndex', () => {
  it('advances to the next slide on a decisive left drag', () => {
    // width*0.15 = 45, floor 40 → trigger 45; -60 clears it.
    expect(resolveSwipeIndex({ ...base, deltaX: -60 })).toBe(2);
  });

  it('goes to the previous slide on a decisive right drag', () => {
    expect(resolveSwipeIndex({ ...base, deltaX: 60 })).toBe(0);
  });

  it('stays put when the drag is below the threshold', () => {
    expect(resolveSwipeIndex({ ...base, deltaX: -20 })).toBe(1);
    expect(resolveSwipeIndex({ ...base, deltaX: 30 })).toBe(1);
  });

  it('clamps at the last slide (no wrap-around)', () => {
    expect(resolveSwipeIndex({ ...base, index: 2, deltaX: -100 })).toBe(2);
  });

  it('clamps at the first slide', () => {
    expect(resolveSwipeIndex({ ...base, index: 0, deltaX: 100 })).toBe(0);
  });

  it('never moves with a single banner', () => {
    expect(resolveSwipeIndex({ width: 300, index: 0, count: 1, deltaX: -200 })).toBe(0);
  });

  it('respects the absolute minimum on very narrow viewports', () => {
    // width*0.15 = 15, but the 40px floor wins, so a 30px drag does nothing.
    expect(resolveSwipeIndex({ width: 100, index: 0, count: 3, deltaX: -30 })).toBe(0);
    expect(resolveSwipeIndex({ width: 100, index: 0, count: 3, deltaX: -45 })).toBe(1);
  });
});

describe('wrapIndex', () => {
  it('advances and wraps around for auto-play', () => {
    expect(wrapIndex(0, 3)).toBe(1);
    expect(wrapIndex(2, 3)).toBe(0);
  });

  it('handles an empty set safely', () => {
    expect(wrapIndex(0, 0)).toBe(0);
  });
});
