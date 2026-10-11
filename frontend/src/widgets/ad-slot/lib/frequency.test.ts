import { beforeEach, describe, expect, it, vi } from 'vitest';

import { MAX_PER_DAY, capReached, pathAllowsAds, recordShown } from './frequency';

describe('ad frequency cap', () => {
  beforeEach(() => {
    const store = new Map<string, string>();
    vi.stubGlobal('window', {
      localStorage: {
        getItem: (k: string) => store.get(k) ?? null,
        setItem: (k: string, v: string) => void store.set(k, v),
      },
    });
  });

  it('caps an ad after MAX_PER_DAY impressions and resets the next day', () => {
    const day1 = new Date(2026, 9, 11, 9);
    for (let i = 0; i < MAX_PER_DAY; i++) {
      expect(capReached(7, day1)).toBe(false);
      recordShown(7, day1);
    }
    expect(capReached(7, day1)).toBe(true);
    expect(capReached(8, day1)).toBe(false);
    expect(capReached(7, new Date(2026, 9, 12, 9))).toBe(false);
  });

  it('never allows ads on teen / loss / emergency / health paths', () => {
    for (const p of ['/teen', '/fa/loss/care', '/emergency', '/en/health-record/labs', '/doctor/chat', '/consent']) {
      expect(pathAllowsAds(p)).toBe(false);
    }
    expect(pathAllowsAds('/home')).toBe(true);
    expect(pathAllowsAds('/fa/home')).toBe(true);
  });
});
