import { describe, expect, it } from 'vitest';

import {
  addDays,
  birthYearRange,
  convertParts,
  daysInCalendarMonth,
  diffInDays,
  formatDayMonth,
  formatDecimal,
  formatLongDate,
  formatNumericDate,
  formatWeekday,
  formatWeekdayDayMonth,
  monthMatrix,
  partsToDate,
  toApiDate,
  toParts,
  weekdayKeys,
  weekOf,
} from './calendar';

// Dates are built with local Y/M/D (month is 0-based in the Date constructor)
// to avoid UTC-parsing surprises across timezones.

describe('toParts', () => {
  it('maps Nowruz 1403 (2024-03-20) to 1/1/1403 in fa', () => {
    expect(toParts(new Date(2024, 2, 20), 'fa')).toEqual({ year: 1403, month: 1, day: 1 });
  });

  it('maps 2020-12-31 to 1399/10/11 in fa', () => {
    expect(toParts(new Date(2020, 11, 31), 'fa')).toEqual({ year: 1399, month: 10, day: 11 });
  });

  it('stays Gregorian in en', () => {
    expect(toParts(new Date(2024, 2, 20), 'en')).toEqual({ year: 2024, month: 3, day: 20 });
  });
});

describe('formatLongDate', () => {
  it('formats a long Persian (Jalali) date with Persian month name and digits', () => {
    expect(formatLongDate(new Date(2024, 2, 20), 'fa')).toBe('۱ فروردین ۱۴۰۳');
  });

  it('formats a Gregorian date with English month name for en', () => {
    expect(formatLongDate(new Date(2024, 2, 20), 'en')).toBe('20 March 2024');
  });
});

describe('formatDayMonth', () => {
  it('drops the year in each calendar', () => {
    expect(formatDayMonth(new Date(2024, 2, 20), 'fa')).toBe('۱ فروردین');
    expect(formatDayMonth(new Date(2024, 2, 20), 'en')).toBe('20 March');
  });
});

describe('formatNumericDate', () => {
  it('zero-pads and uses Persian digits for fa', () => {
    expect(formatNumericDate(new Date(2024, 2, 20), 'fa')).toBe('۱۴۰۳/۰۱/۰۱');
  });

  it('zero-pads a Gregorian date for en', () => {
    expect(formatNumericDate(new Date(2020, 11, 31), 'en')).toBe('2020/12/31');
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
    expect(toParts(later, 'fa')).toEqual({ year: 1403, month: 2, day: 1 });
  });

  it('counts whole days between two dates', () => {
    expect(diffInDays(new Date(2024, 2, 30), new Date(2024, 2, 20))).toBe(10);
  });
});

describe('monthMatrix (jalali)', () => {
  const cellsOf = (year: number, month: number) =>
    monthMatrix(year, month, 'fa').flat().filter((c): c is NonNullable<typeof c> => c !== null);

  it('lays a month out on its own days, not the current one', () => {
    const esfand = cellsOf(1404, 12);
    expect(esfand).toHaveLength(30); // 1404 is a leap year → Esfand has 30 days
    expect(toParts(esfand[0]!.date, 'fa')).toEqual({ year: 1404, month: 12, day: 1 });
    expect(toParts(esfand.at(-1)!.date, 'fa')).toEqual({ year: 1404, month: 12, day: 30 });
  });

  it('gives adjacent months across a year boundary distinct days', () => {
    // Regression: the plugin's setters wrote Gregorian fields, so Esfand 1404
    // and Farvardin 1405 both rendered as Farvardin — one tap toggled both.
    const esfand = cellsOf(1404, 12);
    const farvardin = cellsOf(1405, 1);
    expect(farvardin).toHaveLength(31);
    expect(toApiDate(farvardin[0]!.date)).toBe('2026-03-21');
    expect(toApiDate(esfand[0]!.date)).toBe('2026-02-19');
    const overlap = new Set(esfand.map((c) => toApiDate(c.date)));
    expect(farvardin.some((c) => overlap.has(toApiDate(c.date)))).toBe(false);
  });

  it('pads the first week so day 1 lands on its real weekday (Saturday-first)', () => {
    const firstWeek = monthMatrix(1405, 1, 'fa')[0]!;
    // 1405/01/01 = 2026-03-21, a Saturday → no leading blanks.
    expect(firstWeek[0]?.day).toBe(1);
    expect(monthMatrix(1404, 12, 'fa')[0]!.findIndex((c) => c !== null)).toBe(5); // 2026-02-19 is a Thursday
  });
});

describe('monthMatrix (gregorian)', () => {
  it('lays a Gregorian month out on a Sunday-first grid', () => {
    const march = monthMatrix(2024, 3, 'en');
    const cells = march.flat().filter((c): c is NonNullable<typeof c> => c !== null);
    expect(cells).toHaveLength(31);
    expect(toApiDate(cells[0]!.date)).toBe('2024-03-01');
    // 2024-03-01 is a Friday → five leading blanks on a Sunday-first grid.
    expect(march[0]!.findIndex((c) => c !== null)).toBe(5);
  });

  it('handles a February leap day', () => {
    expect(daysInCalendarMonth(2024, 2, 'en')).toBe(29);
    expect(daysInCalendarMonth(2023, 2, 'en')).toBe(28);
  });
});

describe('daysInCalendarMonth (jalali)', () => {
  it('knows Esfand is 29 days in a common year and 30 in a leap year', () => {
    expect(daysInCalendarMonth(1403, 12, 'fa')).toBe(29);
    expect(daysInCalendarMonth(1404, 12, 'fa')).toBe(30);
  });
});

describe('weekdayKeys', () => {
  it('starts the week on Saturday in Jalali and Sunday in Gregorian', () => {
    expect(weekdayKeys('fa')[0]).toBe('sat');
    expect(weekdayKeys('en')[0]).toBe('sun');
  });
});

describe('partsToDate / convertParts', () => {
  it('round-trips parts through an absolute date', () => {
    expect(toApiDate(partsToDate({ year: 1403, month: 1, day: 1 }, 'fa'))).toBe('2024-03-20');
    expect(toApiDate(partsToDate({ year: 2024, month: 3, day: 20 }, 'en'))).toBe('2024-03-20');
  });

  it('clamps a day that overflows the month instead of rolling over', () => {
    // Esfand 1403 has 29 days; 31 must clamp to the 29th, not spill into Farvardin.
    expect(toParts(partsToDate({ year: 1403, month: 12, day: 31 }, 'fa'), 'fa'))
      .toEqual({ year: 1403, month: 12, day: 29 });
    expect(toParts(partsToDate({ year: 2023, month: 2, day: 31 }, 'en'), 'en'))
      .toEqual({ year: 2023, month: 2, day: 28 });
  });

  it('re-expresses parts in the other calendar', () => {
    expect(convertParts({ year: 1403, month: 1, day: 1 }, 'fa', 'en'))
      .toEqual({ year: 2024, month: 3, day: 20 });
    expect(convertParts({ year: 2024, month: 3, day: 20 }, 'en', 'fa'))
      .toEqual({ year: 1403, month: 1, day: 1 });
  });

  it('is a no-op when both locales share a calendar', () => {
    const parts = { year: 1403, month: 1, day: 1 };
    expect(convertParts(parts, 'fa', 'fa')).toBe(parts);
  });
});

describe('birthYearRange', () => {
  it('spans 60 years in whichever calendar the locale reads', () => {
    for (const locale of ['fa', 'en'] as const) {
      const { min, max } = birthYearRange(locale);
      expect(max - min).toBe(59);
    }
  });
});

describe('weekOf', () => {
  // 2025-10-04 is a Saturday.
  const saturday = new Date(2025, 9, 4);

  it('runs Saturday → Friday in the Jalali grid', () => {
    const week = weekOf(new Date(2025, 9, 7), 'fa');
    expect(week).toHaveLength(7);
    expect(toApiDate(week[0])).toBe('2025-10-04');
    expect(toApiDate(week[6])).toBe('2025-10-10');
  });

  it('runs Sunday → Saturday in the Gregorian grid', () => {
    const week = weekOf(saturday, 'en');
    expect(toApiDate(week[0])).toBe('2025-09-28');
    expect(toApiDate(week[6])).toBe('2025-10-04');
  });

  it('carries days of both months for a week that straddles them', () => {
    const week = weekOf(new Date(2025, 9, 1), 'en');
    expect(week.map(toApiDate)).toContain('2025-09-30');
    expect(week.map(toApiDate)).toContain('2025-10-01');
  });
});

describe('formatWeekdayDayMonth', () => {
  // 2024-03-20 is a Wednesday — 1 Farvardin 1403.
  const nowruz = new Date(2024, 2, 20);

  it('reads «weekday، day month» in the Jalali calendar for fa', () => {
    expect(formatWeekdayDayMonth(nowruz, 'fa')).toBe('چهارشنبه، ۱ فروردین');
  });

  it('reads "Weekday, day Month" in the Gregorian calendar for en', () => {
    expect(formatWeekdayDayMonth(nowruz, 'en')).toBe('Wednesday, 20 March');
  });
});

describe('formatWeekday / formatDecimal', () => {
  it('names the weekday in the locale', () => {
    const sat = new Date(2026, 9, 10); // Saturday
    expect(formatWeekday(sat, 'fa')).toBe('شنبه');
    expect(formatWeekday(sat, 'en')).toBe('Saturday');
  });
  it('localizes digits and the decimal separator', () => {
    expect(formatDecimal('1.6', 'fa')).toBe('۱٫۶');
    expect(formatDecimal(62.5, 'en')).toBe('62.5');
  });
});
