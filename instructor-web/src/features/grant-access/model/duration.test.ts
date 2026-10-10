import { describe, expect, it } from 'vitest';

import { untilRange, untilValid } from './duration';

describe('untilValid', () => {
  const now = new Date(2026, 9, 10);
  it('accepts today … +5 years only', () => {
    expect(untilRange(now)).toEqual({ min: '2026-10-10', max: '2031-10-10' });
    expect(untilValid('2026-10-10', now)).toBe(true);
    expect(untilValid('2031-10-10', now)).toBe(true);
    expect(untilValid('2026-10-09', now)).toBe(false);
    expect(untilValid('2031-10-11', now)).toBe(false);
    expect(untilValid('', now)).toBe(false);
  });
});
