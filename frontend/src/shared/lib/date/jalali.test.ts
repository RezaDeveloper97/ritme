import { describe, expect, it } from 'vitest';

import {
  addDays,
  diffInDays,
  formatJalali,
  formatJalaliNumeric,
  toApiDate,
  toJalali,
} from './jalali';

// Dates are built with local Y/M/D (month is 0-based in the Date constructor)
// to avoid UTC-parsing surprises across timezones.

describe('toJalali', () => {
  it('maps Nowruz 1403 (2024-03-20) to 1/1/1403', () => {
    expect(toJalali(new Date(2024, 2, 20))).toEqual({ year: 1403, month: 1, day: 1 });
  });

  it('maps 2020-12-31 to 1399/10/11', () => {
    expect(toJalali(new Date(2020, 11, 31))).toEqual({ year: 1399, month: 10, day: 11 });
  });
});

describe('formatJalali', () => {
  it('formats a long Persian date with Persian month name and digits', () => {
    expect(formatJalali(new Date(2024, 2, 20), 'fa')).toBe('۱ فروردین ۱۴۰۳');
  });

  it('formats a long English-transliterated date with Latin digits', () => {
    expect(formatJalali(new Date(2024, 2, 20), 'en')).toBe('1 Farvardin 1403');
  });
});

describe('formatJalaliNumeric', () => {
  it('zero-pads and uses Persian digits for fa', () => {
    expect(formatJalaliNumeric(new Date(2024, 2, 20), 'fa')).toBe('۱۴۰۳/۰۱/۰۱');
  });

  it('zero-pads with Latin digits for en', () => {
    expect(formatJalaliNumeric(new Date(2020, 11, 31), 'en')).toBe('1399/10/11');
  });
});

describe('toApiDate', () => {
  it('serializes to a Gregorian YYYY-MM-DD string with Latin digits', () => {
    expect(toApiDate(new Date(2024, 2, 20))).toBe('2024-03-20');
  });

  it('zero-pads month and day', () => {
    expect(toApiDate(new Date(2020, 0, 5))).toBe('2020-01-05');
  });
});

describe('addDays / diffInDays', () => {
  it('adds days across a Jalali month boundary', () => {
    const start = new Date(2024, 2, 20); // 1403/01/01
    const later = addDays(start, 31); // crosses Farvardin (31 days)
    expect(toJalali(later)).toEqual({ year: 1403, month: 2, day: 1 });
  });

  it('counts whole days between two dates', () => {
    expect(diffInDays(new Date(2024, 2, 30), new Date(2024, 2, 20))).toBe(10);
  });
});
