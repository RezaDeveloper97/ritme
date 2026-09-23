/**
 * The admin's one date layer (frontend/CLAUDE.md §7, in miniature). The API
 * sends ISO 8601 in Asia/Tehran (`2026-09-23T13:00:00+03:30`) or calendar dates
 * (`2026-09-23`); nothing outside this file formats a date.
 *
 * The calendar follows the UI locale: `fa` → Jalali (Persian calendar, Persian
 * digits), every other locale → Gregorian. `locale === 'fa'` is the one
 * sanctioned locale check (Jalali is genuinely Persian-specific).
 */
const TIME_ZONE = 'Asia/Tehran';

function intlLocale(locale: string): string {
  return locale === 'fa' ? 'fa-IR-u-ca-persian' : `${locale}-u-ca-gregory`;
}

function parse(value: string | null | undefined): Date | null {
  if (!value) return null;
  // A bare calendar date is a Tehran date, not UTC midnight.
  const iso = /^\d{4}-\d{2}-\d{2}$/.test(value) ? `${value}T12:00:00+03:30` : value;
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? null : date;
}

/** `۱۴۰۵/۰۷/۰۱ ۱۳:۰۰` (fa) or `09/23/2026 13:00` (en). Empty string for null. */
export function formatDateTime(value: string | null | undefined, locale: string): string {
  const date = parse(value);
  if (!date) return '';
  const base = { timeZone: TIME_ZONE } as const;
  // Date and time formatted apart and joined by a space: Intl's own joiner is a
  // Latin comma even in fa, which reads wrongly inside RTL text.
  const day = new Intl.DateTimeFormat(intlLocale(locale), {
    ...base,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(date);
  const time = new Intl.DateTimeFormat(intlLocale(locale), {
    ...base,
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  }).format(date);
  return `${day} ${time}`;
}

/** `۱ مهر ۱۴۰۵` (fa) or `Sep 23, 2026` (en). */
export function formatDate(value: string | null | undefined, locale: string): string {
  const date = parse(value);
  if (!date) return '';
  return new Intl.DateTimeFormat(intlLocale(locale), {
    timeZone: TIME_ZONE,
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(date);
}

/** Hour of day in Tehran (0–23), for greetings and the like. */
export function tehranHour(now: Date = new Date()): number {
  const hour = new Intl.DateTimeFormat('en-US', {
    timeZone: TIME_ZONE,
    hour: 'numeric',
    hourCycle: 'h23',
  }).format(now);
  return Number(hour);
}
