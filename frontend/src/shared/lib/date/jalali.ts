import dayjs from 'dayjs';
import jalaliday from 'jalaliday';

import type { Locale } from '@/shared/i18n';

// This module is the ONLY place allowed to touch the date library directly
// (CLAUDE.md §7). Everything else imports the helpers below from
// `@/shared/lib/date`. Persian digits are handled here, not in components.
dayjs.extend(jalaliday);

const JALALI_MONTHS: Record<Locale, readonly string[]> = {
  fa: [
    'فروردین', 'اردیبهشت', 'خرداد', 'تیر', 'مرداد', 'شهریور',
    'مهر', 'آبان', 'آذر', 'دی', 'بهمن', 'اسفند',
  ],
  en: [
    'Farvardin', 'Ordibehesht', 'Khordad', 'Tir', 'Mordad', 'Shahrivar',
    'Mehr', 'Aban', 'Azar', 'Dey', 'Bahman', 'Esfand',
  ],
};

const PERSIAN_DIGITS = ['۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'];

function toPersianDigits(value: string): string {
  return value.replace(/[0-9]/g, (digit) => PERSIAN_DIGITS[Number(digit)]);
}

function localizeDigits(value: string, locale: Locale): string {
  return locale === 'fa' ? toPersianDigits(value) : value;
}

export interface JalaliParts {
  /** Jalali year, e.g. 1403. */
  year: number;
  /** Jalali month, 1–12 (1 = Farvardin). */
  month: number;
  /** Day of month, 1–31. */
  day: number;
}

/** Break a date down into its Jalali (Shamsi) year/month/day parts. */
export function toJalali(date: Date): JalaliParts {
  const jalali = dayjs(date).calendar('jalali');
  return {
    year: Number(jalali.format('YYYY')),
    month: Number(jalali.format('M')),
    day: Number(jalali.format('D')),
  };
}

/**
 * Long Jalali date, e.g. «۹ خرداد ۱۴۰۵» (fa) or "9 Khordad 1405" (en).
 * User-facing dates are always Jalali (CLAUDE.md §7).
 */
export function formatJalali(date: Date, locale: Locale): string {
  const { year, month, day } = toJalali(date);
  const monthName = JALALI_MONTHS[locale][month - 1];
  return localizeDigits(`${day} ${monthName} ${year}`, locale);
}

/** Numeric Jalali date, e.g. «۱۴۰۵/۰۳/۰۹» (fa) or "1405/03/09" (en). */
export function formatJalaliNumeric(date: Date, locale: Locale): string {
  const { year, month, day } = toJalali(date);
  const pad = (value: number) => String(value).padStart(2, '0');
  return localizeDigits(`${year}/${pad(month)}/${pad(day)}`, locale);
}

/**
 * Short Jalali date without the year, e.g. «۹ خرداد» (fa) or "9 Khordad" (en).
 * Used where the year is implied by surrounding context (e.g. the calendar).
 */
export function formatJalaliDayMonth(date: Date, locale: Locale): string {
  const { month, day } = toJalali(date);
  const monthName = JALALI_MONTHS[locale][month - 1];
  return localizeDigits(`${day} ${monthName}`, locale);
}

/** Month name + year for a Jalali month, e.g. «دی ۱۴۰۳» (fa) / "Dey 1403" (en). */
export function formatJalaliMonthLabel(year: number, month: number, locale: Locale): string {
  const monthName = JALALI_MONTHS[locale][month - 1];
  return localizeDigits(`${monthName} ${year}`, locale);
}

/** Today, normalized to the start of the day. */
export function today(): Date {
  return dayjs().startOf('day').toDate();
}

/**
 * Serialize a date to the API's Gregorian `YYYY-MM-DD` format (CLAUDE.md §7:
 * Gregorian may exist at the API boundary, never shown to the user). Date-keyed
 * endpoints like `/health-logs/{date}` expect exactly this shape.
 */
export function toApiDate(date: Date): string {
  return dayjs(date).format('YYYY-MM-DD');
}

/**
 * Parse the API's Gregorian `YYYY-MM-DD` back into a start-of-day Date — the
 * inverse of {@link toApiDate}. Used when a date crosses back in from a route
 * query (e.g. opening the log editor on a specific calendar day).
 */
export function fromApiDate(value: string): Date {
  return dayjs(value).startOf('day').toDate();
}

/** Today's Jalali parts. */
export function todayJalali(): JalaliParts {
  return toJalali(today());
}

/** One day inside a rendered Jalali month grid. */
export interface JalaliMonthCell {
  /** Day of month, 1–31. */
  day: number;
  /** The underlying Gregorian date this cell represents (start of day). */
  date: Date;
}

/**
 * Lay a Jalali month out as weeks for a **Saturday-first** (شنبه) grid — the
 * order Iranian calendars use. Leading/trailing blanks are `null`. This is the
 * one place that knows how a Jalali month maps onto a weekday grid, so calendar
 * UIs never touch the date library themselves (CLAUDE.md §7).
 */
export function jalaliMonthMatrix(year: number, month: number): (JalaliMonthCell | null)[][] {
  const first = dayjs().calendar('jalali').year(year).month(month - 1).date(1);
  const daysInMonth = first.daysInMonth();
  // `day()` is 0=Sunday…6=Saturday; shift so Saturday becomes column 0.
  const offset = (first.day() + 1) % 7;

  const cells: (JalaliMonthCell | null)[] = [];
  for (let i = 0; i < offset; i += 1) cells.push(null);
  for (let d = 1; d <= daysInMonth; d += 1) {
    cells.push({ day: d, date: first.date(d).startOf('day').toDate() });
  }
  while (cells.length % 7 !== 0) cells.push(null);

  const weeks: (JalaliMonthCell | null)[][] = [];
  for (let i = 0; i < cells.length; i += 7) weeks.push(cells.slice(i, i + 7));
  return weeks;
}

/** Step a Jalali (year, month) pair by whole months, wrapping the year. */
export function shiftJalaliMonth(
  year: number,
  month: number,
  delta: number,
): { year: number; month: number } {
  const zeroBased = (month - 1) + delta;
  const nextYear = year + Math.floor(zeroBased / 12);
  const nextMonth = ((zeroBased % 12) + 12) % 12;
  return { year: nextYear, month: nextMonth + 1 };
}

/** A new date `days` days after `date` (use a negative number to subtract). */
export function addDays(date: Date, days: number): Date {
  return dayjs(date).add(days, 'day').toDate();
}

/** Whole calendar days between two dates (a − b). */
export function diffInDays(a: Date, b: Date): number {
  return dayjs(a).startOf('day').diff(dayjs(b).startOf('day'), 'day');
}
