import type { BloodPressureValue, MonthlyCalendar, MonthlyMetricKey } from '@/entities/analysis';
import type { Locale } from '@/shared/i18n';
import { calendarSystem, convertParts, shiftMonth, todayParts } from '@/shared/lib/date';

/** A calendar month (1-based month) in some calendar. */
export interface YearMonth {
  year: number;
  month: number;
}

/** How far back the month stepper goes (the hub's longest range is a year; two give a year-on-year look). */
export const MONTHS_BACK = 24;

const YM = /^(\d{4})-(0[1-9]|1[0-2])$/;

/** `YYYY-MM` → parts, or null for anything the API would refuse as a format error. */
export function parseYm(ym: string): YearMonth | null {
  const m = YM.exec(ym);
  return m ? { year: Number(m[1]), month: Number(m[2]) } : null;
}

export function formatYm({ year, month }: YearMonth): string {
  return `${String(year).padStart(4, '0')}-${String(month).padStart(2, '0')}`;
}

/** A locale that reads dates in `calendar` (the date utils pick the calendar from the locale). */
function localeOf(calendar: MonthlyCalendar): Locale {
  return calendar === 'jalali' ? 'fa' : 'en';
}

const index = ({ year, month }: YearMonth) => year * 12 + (month - 1);

/**
 * The month the screen shows, in the reader's calendar. A link built in the
 * other calendar (e.g. a fa link opened after switching to en) is re-expressed
 * as the reader's month that contains its first day — month names come from the
 * locale, so a mixed pair could not be titled. Parses through `convertParts`
 * (jalaliday setters write Gregorian fields; never chain them).
 */
export function resolveMonth(ym: YearMonth, calendar: MonthlyCalendar, locale: Locale): YearMonth {
  const own = calendarSystem(locale);
  if (own === calendar) return ym;
  const p = convertParts({ ...ym, day: 1 }, localeOf(calendar), locale);
  return { year: p.year, month: p.month };
}

export interface MonthBounds {
  /** The running month (today) — the latest allowed. */
  current: YearMonth;
  /** The earliest allowed. */
  earliest: YearMonth;
}

export function monthBounds(locale: Locale): MonthBounds {
  const { year, month } = todayParts(locale);
  const current = { year, month };
  return { current, earliest: shiftMonth(year, month, -(MONTHS_BACK - 1)) };
}

export type MonthPlacement = 'future' | 'too_old' | 'ok';

export function placeMonth(ym: YearMonth, bounds: MonthBounds): MonthPlacement {
  if (index(ym) > index(bounds.current)) return 'future';
  if (index(ym) < index(bounds.earliest)) return 'too_old';
  return 'ok';
}

/** The neighbour `delta` months away, or null when it falls outside the bounds (no future months). */
export function neighbour(ym: YearMonth, delta: number, bounds: MonthBounds): YearMonth | null {
  const next = shiftMonth(ym.year, ym.month, delta);
  return placeMonth(next, bounds) === 'ok' ? next : null;
}

/**
 * Colour of a «تغییر» cell: turquoise when the change reads well or is just a
 * measurement, amber when it reads worse (fewer logged days, less sleep, fewer
 * good-mood days, higher blood pressure — as the artboard paints them), muted
 * when nothing moved.
 */
export type DeltaTone = 'data' | 'warm' | 'flat';

const HIGHER_IS_BETTER: ReadonlySet<MonthlyMetricKey> = new Set(['days_logged', 'sleep', 'good_mood']);

export function deltaTone(key: MonthlyMetricKey, delta: number | BloodPressureValue): DeltaTone {
  if (typeof delta !== 'number') {
    if (delta.systolic === 0 && delta.diastolic === 0) return 'flat';
    return delta.systolic > 0 || delta.diastolic > 0 ? 'warm' : 'data';
  }
  if (delta === 0) return 'flat';
  if (HIGHER_IS_BETTER.has(key)) return delta > 0 ? 'data' : 'warm';
  return 'data';
}

/** «−۱» / «+۰٫۳» parts: U+2212 minus as the artboards print it; zero has no sign. */
export function signed(value: number, places: number): { sign: '−' | '+' | ''; abs: string } {
  const factor = 10 ** places;
  const rounded = Math.round(value * factor) / factor;
  const abs = Math.abs(rounded).toFixed(places).replace(/\.0+$/, '');
  if (rounded === 0) return { sign: '', abs };
  return { sign: rounded < 0 ? '−' : '+', abs };
}
