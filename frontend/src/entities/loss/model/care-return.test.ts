import { afterEach, describe, expect, it, vi } from 'vitest';

import { LOSS_CARE_WINDOW_DAYS, hideLossCareRow, isLossCareOpen, isLossCareRowHidden } from './care-return';
import type { LossEvent } from './types';

const loss = (over: Partial<LossEvent> = {}): LossEvent => ({
  id: 7,
  type: 'early_miscarriage',
  occurredOn: null,
  notifyCompanion: false,
  companionNotified: false,
  contentStopped: true,
  nextStep: null,
  createdAt: '2026-10-01T23:30:00+03:30',
  ...over,
});

describe('isLossCareOpen', () => {
  it('is closed without a loss', () => {
    expect(isLossCareOpen(null, '2026-10-04')).toBe(false);
  });

  it('counts the window from the Tehran day she recorded it', () => {
    expect(LOSS_CARE_WINDOW_DAYS).toBe(60);
    expect(isLossCareOpen(loss(), '2026-10-01')).toBe(true);
    expect(isLossCareOpen(loss(), '2026-11-29')).toBe(true); // day 59
    expect(isLossCareOpen(loss(), '2026-11-30')).toBe(false); // day 60
    expect(isLossCareOpen(loss(), '2026-09-30')).toBe(false);
  });

  it('falls back to the approximate date, and stays closed with neither', () => {
    expect(isLossCareOpen(loss({ createdAt: null, occurredOn: '2026-09-20' }), '2026-10-04')).toBe(true);
    expect(isLossCareOpen(loss({ createdAt: null }), '2026-10-04')).toBe(false);
  });
});

describe('hidden home row', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('remembers only the hidden loss id', () => {
    const store = new Map<string, string>();
    vi.stubGlobal('window', {
      localStorage: { getItem: (k: string) => store.get(k) ?? null, setItem: (k: string, v: string) => void store.set(k, v) },
    });
    expect(isLossCareRowHidden(7)).toBe(false);
    hideLossCareRow(7);
    expect(isLossCareRowHidden(7)).toBe(true);
    expect(isLossCareRowHidden(8)).toBe(false); // a newer loss shows the row again
    expect([...store.entries()]).toEqual([['ritme_care_return_hidden', '7']]);
  });

  it('shows the row when storage is unavailable', () => {
    vi.stubGlobal('window', {
      localStorage: {
        getItem: () => {
          throw new Error('blocked');
        },
        setItem: () => {
          throw new Error('blocked');
        },
      },
    });
    expect(() => hideLossCareRow(7)).not.toThrow();
    expect(isLossCareRowHidden(7)).toBe(false);
  });
});
