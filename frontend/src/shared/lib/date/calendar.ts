import dayjs from 'dayjs';
import jalaliday from 'jalaliday';

import type { Locale } from '@/shared/i18n';

// This module is the ONLY place allowed to touch the date library directly
// (CLAUDE.md §7). Everything else imports the helpers below from
// `@/shared/lib/date`. Persian digits are handled here, not in components.
//
// The calendar *system* follows the locale: Persian users get Jalali (Shamsi),
// English users get the Gregorian calendar they actually think in. Every helper
// that produces or consumes calendar parts therefore takes the locale — there
// is no ambient "current calendar", so a component can never render a month
// grid in one calendar while labelling it with the other.
dayjs.extend(jalaliday);

/** Which calendar a locale reads dates in. */
export type CalendarSystem = 'jalali' | 'gregorian';

/** fa → Jalali (Shamsi), en → Gregorian. */
export function calendarSystem(locale: Locale): CalendarSystem {
  return locale === 'fa' ? 'jalali' : 'gregorian';
}

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

const GREGORIAN_MONTHS: Record<Locale, readonly string[]> = {
  fa: [
    'ژانویه', 'فوریه', 'مارس', 'آوریل', 'مه', 'ژوئن',
    'ژوئیه', 'اوت', 'سپتامبر', 'اکتبر', 'نوامبر', 'دسامبر',
  ],
  en: [
    'January', 'February', 'March', 'April', 'May', 'June',
    'July', 'August', 'September', 'October', 'November', 'December',
  ],
};

function monthNames(locale: Locale): readonly string[] {
  return calendarSystem(locale) === 'jalali' ? JALALI_MONTHS[locale] : GREGORIAN_MONTHS[locale];
}

const PERSIAN_DIGITS = ['۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'];

function toPersianDigits(value: string): string {
  return value.replace(/[0-9]/g, (digit) => PERSIAN_DIGITS[Number(digit)]);
}

function localizeDigits(value: string, locale: Locale): string {
  return locale === 'fa' ? toPersianDigits(value) : value;
}

/** Localize the digits of an already-formatted string (Persian digits in fa). */
export function formatNumber(value: string | number, locale: Locale): string {
  return localizeDigits(String(value), locale);
}

/**
 * A calendar date broken into parts **of the locale's calendar** — Jalali for
 * `fa`, Gregorian for `en`. Parts are meaningless without the locale that
 * produced them; pass both around together.
 */
export interface DateParts {
  /** Year in the active calendar, e.g. 1403 (Jalali) or 2025 (Gregorian). */
  year: number;
  /** Month, 1–12 (1 = Farvardin / January). */
  month: number;
  /** Day of month, 1–31. */
  day: number;
}

/** dayjs instance in the calendar the locale uses. */
function inCalendar(date: Date, locale: Locale) {
  const d = dayjs(date);
  return calendarSystem(locale) === 'jalali' ? d.calendar('jalali') : d;
}

/** Break a date down into its year/month/day parts in the locale's calendar. */
export function toParts(date: Date, locale: Locale): DateParts {
  const d = inCalendar(date, locale);
  return {
    year: Number(d.format('YYYY')),
    month: Number(d.format('M')),
    day: Number(d.format('D')),
  };
}

/**
 * Long date, e.g. «۹ خرداد ۱۴۰۵» (fa) or "9 June 2026" (en).
 * Always rendered in the calendar the user's locale reads (CLAUDE.md §7).
 */
export function formatLongDate(date: Date, locale: Locale): string {
  const { year, month, day } = toParts(date, locale);
  return localizeDigits(`${day} ${monthNames(locale)[month - 1]} ${year}`, locale);
}

/** Numeric date, e.g. «۱۴۰۵/۰۳/۰۹» (fa) or "2026/06/09" (en). */
export function formatNumericDate(date: Date, locale: Locale): string {
  const { year, month, day } = toParts(date, locale);
  const pad = (value: number) => String(value).padStart(2, '0');
  return localizeDigits(`${year}/${pad(month)}/${pad(day)}`, locale);
}

/**
 * Short date without the year, e.g. «۹ خرداد» (fa) or "9 June" (en).
 * Used where the year is implied by surrounding context (e.g. the calendar).
 */
export function formatDayMonth(date: Date, locale: Locale): string {
  const { month, day } = toParts(date, locale);
  return localizeDigits(`${day} ${monthNames(locale)[month - 1]}`, locale);
}

/** Name of a month (1–12) in the locale's calendar. */
export function monthName(month: number, locale: Locale): string {
  return monthNames(locale)[month - 1];
}

/** All twelve month names of the locale's calendar, in order. */
export function allMonthNames(locale: Locale): readonly string[] {
  return monthNames(locale);
}

/** A year with localized digits, e.g. «۱۴۰۵» (fa) / "2026" (en). */
export function formatYear(year: number, locale: Locale): string {
  return localizeDigits(String(year), locale);
}

/** Month name + year, e.g. «دی ۱۴۰۳» (fa) / "December 2025" (en). */
export function formatMonthLabel(year: number, month: number, locale: Locale): string {
  return localizeDigits(`${monthNames(locale)[month - 1]} ${year}`, locale);
}

/** Today, normalized to the start of the day. */
export function today(): Date {
  return dayjs().startOf('day').toDate();
}

/**
 * Serialize a date to the API's Gregorian `YYYY-MM-DD` format (CLAUDE.md §7:
 * Gregorian may exist at the API boundary, never shown raw to the user).
 * Date-keyed endpoints like `/health-logs/{date}` expect exactly this shape.
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

/** Today's parts in the locale's calendar. */
export function todayParts(locale: Locale): DateParts {
  return toParts(today(), locale);
}

/** One day inside a rendered month grid. */
export interface MonthCell {
  /** Day of month, 1–31. */
  day: number;
  /** The underlying absolute date this cell represents (start of day). */
  date: Date;
}

/**
 * The first day of a (year, month) in the locale's calendar, as a dayjs.
 *
 * Jalali dates are built by *parsing* rather than with `.year()/.month()/
 * .date()`: the plugin's setters write the Gregorian fields even in jalali
 * mode, so chaining them lands in a different month. The `{ jalali: true }`
 * parse option is real but missing from the plugin's type declarations, hence
 * the cast.
 */
function firstOfMonth(year: number, month: number, locale: Locale) {
  if (calendarSystem(locale) === 'jalali') {
    const option = { jalali: true } as unknown as Parameters<typeof dayjs>[1];
    return dayjs(`${year}/${month}/1`, option).calendar('jalali').startOf('day');
  }
  return dayjs(new Date(year, month - 1, 1)).startOf('day');
}

/** Number of days in a month (Esfand is 29 or 30; February 28 or 29). */
export function daysInCalendarMonth(year: number, month: number, locale: Locale): number {
  return firstOfMonth(year, month, locale).daysInMonth();
}

/**
 * Weekday column order for the locale's calendar: **Saturday-first** (شنبه) for
 * Jalali — the order Iranian calendars use — and **Sunday-first** for
 * Gregorian. Keys match the `weekdays.*` message namespaces.
 */
export const WEEKDAY_KEYS = ['sat', 'sun', 'mon', 'tue', 'wed', 'thu', 'fri'] as const;
export type WeekdayKey = (typeof WEEKDAY_KEYS)[number];

const GREGORIAN_WEEKDAY_KEYS = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat'] as const;

export function weekdayKeys(locale: Locale): readonly WeekdayKey[] {
  return calendarSystem(locale) === 'jalali' ? WEEKDAY_KEYS : GREGORIAN_WEEKDAY_KEYS;
}

const WEEKDAY_SHORT: Record<Locale, Record<WeekdayKey, string>> = {
  fa: { sat: 'ش', sun: 'ی', mon: 'د', tue: 'س', wed: 'چ', thu: 'پ', fri: 'ج' },
  en: { sat: 'Sa', sun: 'Su', mon: 'Mo', tue: 'Tu', wed: 'We', thu: 'Th', fri: 'Fr' },
};

/**
 * Short weekday headers already in the locale's column order. For screens that
 * own a `weekdays.*` message namespace, prefer {@link weekdayKeys} + the
 * translator; this is for `shared` components with no namespace of their own.
 */
export function weekdayLabels(locale: Locale): string[] {
  return weekdayKeys(locale).map((key) => WEEKDAY_SHORT[locale][key]);
}

/** Column index (0–6) of `date` in the locale's weekday grid. */
function weekdayColumn(day: number, locale: Locale): number {
  // dayjs `day()` is 0=Sunday…6=Saturday.
  return calendarSystem(locale) === 'jalali' ? (day + 1) % 7 : day;
}

/**
 * Lay a month out as weeks for the locale's weekday grid (Saturday-first in
 * Jalali, Sunday-first in Gregorian). Leading/trailing blanks are `null`. This
 * is the one place that knows how a month maps onto a weekday grid, so calendar
 * UIs never touch the date library themselves (CLAUDE.md §7).
 */
export function monthMatrix(
  year: number,
  month: number,
  locale: Locale,
): (MonthCell | null)[][] {
  // Build the month from a *parsed* first-of-month and then step whole days.
  // The jalaliday `.year()/.month()/.date()` setters operate on the Gregorian
  // fields even in jalali mode, so chaining them silently lands in the wrong
  // month (Esfand 1404 resolved to Farvardin 1405, rendering two identical
  // grids). Stepping days is the only setter-free way to walk a month.
  const first = firstOfMonth(year, month, locale);
  const total = first.daysInMonth();
  const offset = weekdayColumn(first.day(), locale);

  const cells: (MonthCell | null)[] = [];
  for (let i = 0; i < offset; i += 1) cells.push(null);
  for (let d = 1; d <= total; d += 1) {
    cells.push({ day: d, date: first.add(d - 1, 'day').startOf('day').toDate() });
  }
  while (cells.length % 7 !== 0) cells.push(null);

  const weeks: (MonthCell | null)[][] = [];
  for (let i = 0; i < cells.length; i += 7) weeks.push(cells.slice(i, i + 7));
  return weeks;
}

/** Step a (year, month) pair by whole months, wrapping the year. */
export function shiftMonth(
  year: number,
  month: number,
  delta: number,
): { year: number; month: number } {
  const zeroBased = (month - 1) + delta;
  const nextYear = year + Math.floor(zeroBased / 12);
  const nextMonth = ((zeroBased % 12) + 12) % 12;
  return { year: nextYear, month: nextMonth + 1 };
}

/**
 * Turn user-picked calendar parts into an absolute date. A day that overflows
 * the month (31 Esfand, 31 February) clamps to the month's last day rather than
 * silently rolling into the next month.
 */
export function partsToDate({ year, month, day }: DateParts, locale: Locale): Date {
  // Guard the 1-based month contract: an out-of-range month would make the date
  // library parse a nonsense string and quietly return a wrong date rather than
  // fail (that is how 0-based wheel indexes went unnoticed).
  const safeMonth = Math.min(12, Math.max(1, month));
  const first = firstOfMonth(year, safeMonth, locale);
  const safeDay = Math.min(Math.max(1, day), first.daysInMonth());
  return first.add(safeDay - 1, 'day').startOf('day').toDate();
}

/** Re-express parts entered in one locale's calendar in another's. */
export function convertParts(parts: DateParts, from: Locale, to: Locale): DateParts {
  if (calendarSystem(from) === calendarSystem(to)) return parts;
  return toParts(partsToDate(parts, from), to);
}

/**
 * Selectable year span for a birthday wheel in the locale's calendar: the last
 * 60 years, ending 6 years before today (nobody picking a birthday here is a
 * toddler). Keeps the Jalali and Gregorian wheels covering the same real span.
 */
export function birthYearRange(locale: Locale): { min: number; max: number } {
  const max = todayParts(locale).year - 6;
  return { min: max - 59, max };
}

/** A new date `days` days after `date` (use a negative number to subtract). */
export function addDays(date: Date, days: number): Date {
  return dayjs(date).add(days, 'day').toDate();
}

/** Whole calendar days between two dates (a − b). */
export function diffInDays(a: Date, b: Date): number {
  return dayjs(a).startOf('day').diff(dayjs(b).startOf('day'), 'day');
}
