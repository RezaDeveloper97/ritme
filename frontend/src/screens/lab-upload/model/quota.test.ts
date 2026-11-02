import { describe, expect, it } from 'vitest';

import { labPaywalled, labQuotaNote } from './quota';

describe('lab quota note', () => {
  it('free: one per month', () => {
    const n = labQuotaNote({ state: 'available', limit: 1, remaining: 1 }, false);
    expect(n).toEqual({ key: 'available', remaining: 1, limit: 1 });
    expect(labPaywalled(n, false)).toBe(false);
  });
  it('free spent -> paywall', () => {
    const n = labQuotaNote({ state: 'spent', limit: 1 }, false);
    expect(n).toEqual({ key: 'spent', limit: 1 });
    expect(labPaywalled(n, false)).toBe(true);
  });
  it('plus spent does not paywall', () => {
    expect(labPaywalled(labQuotaNote({ state: 'spent', limit: 10 }, true), true)).toBe(false);
  });
  it('plus available uses its own copy; unlimited/locked/unknown show nothing', () => {
    expect(labQuotaNote({ state: 'available', limit: 10, remaining: 7 }, true)?.key).toBe('plusAvailable');
    expect(labQuotaNote({ state: 'unlimited' }, true)).toBeNull();
    expect(labQuotaNote({ state: 'locked' }, false)).toBeNull();
    expect(labQuotaNote(null, false)).toBeNull();
  });
});
