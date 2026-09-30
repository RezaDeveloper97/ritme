import { describe, expect, it } from 'vitest';

import { fromApiDate } from '@/shared/lib/date';

import { gregorianMonthsBetween } from './months';

describe('gregorianMonthsBetween', () => {
  it('lists every Gregorian month a Jalali two-month span touches', () => {
    // 1 Shahrivar – 30 Mehr 1405 ≈ 23 Aug – 22 Oct 2026.
    expect(gregorianMonthsBetween(fromApiDate('2026-08-23'), fromApiDate('2026-10-22'))).toEqual([
      { year: 2026, month: 8 },
      { year: 2026, month: 9 },
      { year: 2026, month: 10 },
    ]);
  });

  it('handles one day and a year boundary', () => {
    expect(gregorianMonthsBetween(fromApiDate('2026-12-31'), fromApiDate('2026-12-31'))).toEqual([
      { year: 2026, month: 12 },
    ]);
    expect(gregorianMonthsBetween(fromApiDate('2026-12-30'), fromApiDate('2027-01-02'))).toEqual([
      { year: 2026, month: 12 },
      { year: 2027, month: 1 },
    ]);
  });
});
