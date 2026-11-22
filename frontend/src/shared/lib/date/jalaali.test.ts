import { describe, expect, it } from 'vitest';

import {
  daysInCalendarMonth,
  monthMatrix,
  partsToDate,
  toApiDate,
  toParts,
  formatLongDate,
} from './calendar';
import { isLeapJalaaliYear, jalaaliMonthLength, toGregorian, toJalaali } from './jalaali';

// T-M12-21b. The frontend must agree with the backend's Jalali converter
// (backend-go/internal/checkups/engine/jalali.go, a port of jalaali-js) to the
// day. jalaliday 2.3.0 made 1404 leap instead of 1403 and shifted every day of
// 1404 by one, so most rows below fail on the old library.

const local = (iso: string) => {
  const [y, m, d] = iso.split('-').map(Number);
  return new Date(y, m - 1, d);
};

/**
 * First day of every Jalali month 1399–1411, dumped from the backend with
 * `engine.FromJalali(jy, jm, 1)` (go test overlay, 2026-11-18). Do not edit by
 * hand: regenerate from the Go code if the backend algorithm ever changes.
 */
const BACKEND_MONTH_STARTS: readonly (readonly [number, readonly string[]])[] = [
  [1399, ['2020-03-20', '2020-04-20', '2020-05-21', '2020-06-21', '2020-07-22', '2020-08-22', '2020-09-22', '2020-10-22', '2020-11-21', '2020-12-21', '2021-01-20', '2021-02-19']],
  [1400, ['2021-03-21', '2021-04-21', '2021-05-22', '2021-06-22', '2021-07-23', '2021-08-23', '2021-09-23', '2021-10-23', '2021-11-22', '2021-12-22', '2022-01-21', '2022-02-20']],
  [1401, ['2022-03-21', '2022-04-21', '2022-05-22', '2022-06-22', '2022-07-23', '2022-08-23', '2022-09-23', '2022-10-23', '2022-11-22', '2022-12-22', '2023-01-21', '2023-02-20']],
  [1402, ['2023-03-21', '2023-04-21', '2023-05-22', '2023-06-22', '2023-07-23', '2023-08-23', '2023-09-23', '2023-10-23', '2023-11-22', '2023-12-22', '2024-01-21', '2024-02-20']],
  [1403, ['2024-03-20', '2024-04-20', '2024-05-21', '2024-06-21', '2024-07-22', '2024-08-22', '2024-09-22', '2024-10-22', '2024-11-21', '2024-12-21', '2025-01-20', '2025-02-19']],
  [1404, ['2025-03-21', '2025-04-21', '2025-05-22', '2025-06-22', '2025-07-23', '2025-08-23', '2025-09-23', '2025-10-23', '2025-11-22', '2025-12-22', '2026-01-21', '2026-02-20']],
  [1405, ['2026-03-21', '2026-04-21', '2026-05-22', '2026-06-22', '2026-07-23', '2026-08-23', '2026-09-23', '2026-10-23', '2026-11-22', '2026-12-22', '2027-01-21', '2027-02-20']],
  [1406, ['2027-03-21', '2027-04-21', '2027-05-22', '2027-06-22', '2027-07-23', '2027-08-23', '2027-09-23', '2027-10-23', '2027-11-22', '2027-12-22', '2028-01-21', '2028-02-20']],
  [1407, ['2028-03-20', '2028-04-20', '2028-05-21', '2028-06-21', '2028-07-22', '2028-08-22', '2028-09-22', '2028-10-22', '2028-11-21', '2028-12-21', '2029-01-20', '2029-02-19']],
  [1408, ['2029-03-20', '2029-04-20', '2029-05-21', '2029-06-21', '2029-07-22', '2029-08-22', '2029-09-22', '2029-10-22', '2029-11-21', '2029-12-21', '2030-01-20', '2030-02-19']],
  [1409, ['2030-03-21', '2030-04-21', '2030-05-22', '2030-06-22', '2030-07-23', '2030-08-23', '2030-09-23', '2030-10-23', '2030-11-22', '2030-12-22', '2031-01-21', '2031-02-20']],
  [1410, ['2031-03-21', '2031-04-21', '2031-05-22', '2031-06-22', '2031-07-23', '2031-08-23', '2031-09-23', '2031-10-23', '2031-11-22', '2031-12-22', '2032-01-21', '2032-02-20']],
  [1411, ['2032-03-20', '2032-04-20', '2032-05-21', '2032-06-21', '2032-07-22', '2032-08-22', '2032-09-22', '2032-10-22', '2032-11-21', '2032-12-21', '2033-01-20', '2033-02-19']],
];

describe('Gregorian → Jalali matches the official calendar (the backend facts)', () => {
  it.each([
    // The bug rows: the whole of 1404 was one day off.
    ['2025-03-20', 1403, 12, 30], // last day of leap 1403 (old: 1404-01-01)
    ['2025-03-21', 1404, 1, 1], // Nowruz 1404 (old: 1404-01-02)
    ['2025-09-23', 1404, 7, 1], // first day of Mehr 1404 (old: 1404-07-02)
    ['2026-03-20', 1404, 12, 29], // last day of common 1404 (old: 1404-12-30)
    ['2026-03-21', 1405, 1, 1], // Nowruz 1405
    // Leap-year boundaries either side.
    ['2021-03-20', 1399, 12, 30], // 1399 leap
    ['2021-03-21', 1400, 1, 1],
    ['2024-03-19', 1402, 12, 29], // 1402 common
    ['2024-03-20', 1403, 1, 1],
    ['2029-03-19', 1407, 12, 29], // 1407 common
    ['2030-03-20', 1408, 12, 30], // 1408 leap
    ['2030-03-21', 1409, 1, 1],
    // Month boundaries inside the year (31 → 30-day months).
    ['2025-09-22', 1404, 6, 31],
    ['2025-03-01', 1403, 12, 11],
  ])('%s → %i-%i-%i', (iso, year, month, day) => {
    expect(toParts(local(iso), 'fa')).toEqual({ year, month, day });
  });

  it('formats the corrected day in Persian', () => {
    expect(formatLongDate(local('2025-03-21'), 'fa')).toBe('۱ فروردین ۱۴۰۴');
    expect(formatLongDate(local('2025-03-20'), 'fa')).toBe('۳۰ اسفند ۱۴۰۳');
  });
});

describe('leap years', () => {
  it.each([1399, 1403, 1408])('%i is leap: Esfand has 30 days', (year) => {
    expect(isLeapJalaaliYear(year)).toBe(true);
    expect(daysInCalendarMonth(year, 12, 'fa')).toBe(30);
  });

  it.each([1400, 1401, 1402, 1404, 1405, 1406, 1407, 1409, 1410])(
    '%i is common: Esfand has 29 days',
    (year) => {
      expect(isLeapJalaaliYear(year)).toBe(false);
      expect(daysInCalendarMonth(year, 12, 'fa')).toBe(29);
    },
  );

  it('fixes the month lengths for the other eleven months', () => {
    expect(jalaaliMonthLength(1404, 1)).toBe(31);
    expect(jalaaliMonthLength(1404, 6)).toBe(31);
    expect(jalaaliMonthLength(1404, 7)).toBe(30);
    expect(jalaaliMonthLength(1404, 11)).toBe(30);
  });
});

describe('invalid / out-of-range input', () => {
  it('clamps 30 Esfand of a common year to the 29th instead of spilling into Nowruz', () => {
    const d = partsToDate({ year: 1404, month: 12, day: 30 }, 'fa');
    expect(toApiDate(d)).toBe('2026-03-20');
    expect(toParts(d, 'fa')).toEqual({ year: 1404, month: 12, day: 29 });
  });

  it('keeps 30 Esfand of a leap year', () => {
    expect(toApiDate(partsToDate({ year: 1403, month: 12, day: 30 }, 'fa'))).toBe('2025-03-20');
  });

  it('clamps day 0 to the 1st and month 13 to Esfand', () => {
    expect(toApiDate(partsToDate({ year: 1404, month: 1, day: 0 }, 'fa'))).toBe('2025-03-21');
    expect(toParts(partsToDate({ year: 1404, month: 13, day: 1 }, 'fa'), 'fa'))
      .toEqual({ year: 1404, month: 12, day: 1 });
  });

  it('refuses a year outside the algorithm\'s exact range rather than guessing', () => {
    expect(() => toGregorian(3178, 1, 1)).toThrow(RangeError);
    expect(() => toJalaali(500, 1, 1)).toThrow(RangeError);
  });
});

describe('parity with the backend month boundaries, 1399–1411', () => {
  for (const [year, starts] of BACKEND_MONTH_STARTS) {
    it(`${year}: every month starts and ends where the backend says`, () => {
      starts.forEach((iso, i) => {
        const month = i + 1;
        expect(toApiDate(partsToDate({ year, month, day: 1 }, 'fa'))).toBe(iso);
        expect(toParts(local(iso), 'fa')).toEqual({ year, month, day: 1 });

        const nextStart = i < 11 ? starts[i + 1] : BACKEND_MONTH_STARTS.find(([y]) => y === year + 1)?.[1][0];
        if (!nextStart) return;
        const length = Math.round((local(nextStart).getTime() - local(iso).getTime()) / 86_400_000);
        expect(daysInCalendarMonth(year, month, 'fa')).toBe(length);

        const cells = monthMatrix(year, month, 'fa').flat().filter((c) => c !== null);
        expect(cells).toHaveLength(length);
        expect(toApiDate(cells[0]!.date)).toBe(iso);
      });
    });
  }
});

describe('round trips', () => {
  it('Gregorian → Jalali → Gregorian is the identity for every day of 1399–1411', () => {
    const start = local('2020-03-20');
    const end = local('2033-03-20');
    let previous = toParts(addOne(start, -1), 'fa');
    for (let d = start; d < end; d = addOne(d, 1)) {
      const parts = toParts(d, 'fa');
      expect(toApiDate(partsToDate(parts, 'fa'))).toBe(toApiDate(d));
      // Consecutive days are consecutive Jalali days (no gaps, no repeats).
      const sameMonth = parts.year === previous.year && parts.month === previous.month;
      if (sameMonth) expect(parts.day).toBe(previous.day + 1);
      else expect(parts.day).toBe(1);
      previous = parts;
    }
  });

  it('the pure converters invert each other', () => {
    expect(toJalaali(2025, 3, 21)).toEqual({ year: 1404, month: 1, day: 1 });
    expect(toGregorian(1404, 1, 1)).toEqual({ year: 2025, month: 3, day: 21 });
    expect(toGregorian(1404, 12, 29)).toEqual({ year: 2026, month: 3, day: 20 });
  });
});

function addOne(date: Date, days: number): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate() + days);
}
