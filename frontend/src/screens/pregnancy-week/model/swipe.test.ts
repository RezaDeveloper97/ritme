import { describe, expect, it } from 'vitest';

import { swipeTarget } from './swipe';

describe('swipeTarget', () => {
  it('advances with reading direction', () => {
    expect(swipeTarget(8, 100, 0, 'rtl')).toBe(9);
    expect(swipeTarget(8, -100, 0, 'rtl')).toBe(7);
    expect(swipeTarget(8, -100, 0, 'ltr')).toBe(9);
    expect(swipeTarget(8, 100, 0, 'ltr')).toBe(7);
  });

  it('ignores short or vertical gestures', () => {
    expect(swipeTarget(8, 30, 0, 'rtl')).toBeNull();
    expect(swipeTarget(8, 80, 90, 'rtl')).toBeNull();
  });

  it('stops at the 1 / 42 edges', () => {
    expect(swipeTarget(42, 100, 0, 'rtl')).toBeNull();
    expect(swipeTarget(1, -100, 0, 'rtl')).toBeNull();
  });
});
