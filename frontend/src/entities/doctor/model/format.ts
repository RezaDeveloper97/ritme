import type { Locale } from '@/shared/i18n';
import { formatNumber, formatWeekdayDayMonth, fromApiDate } from '@/shared/lib/date';

/** Prices travel as integer rials (QUESTIONS #120); the UI shows toman. */
export const toman = (rials: number): number => Math.round(rials / 10);

/** «۲۵۰٬۰۰۰» — grouped toman in the locale's digits. */
export function formatToman(rials: number, locale: Locale): string {
  return formatNumber(toman(rials).toLocaleString('en-US'), locale).replace(/,/g, locale === 'fa' ? '٬' : ',');
}

/** «HH:MM» in the locale's digits. */
export const formatClock = (hhmm: string, locale: Locale): string => formatNumber(hhmm, locale);

/** The doctor's first slot is "today" / "tomorrow" / a date — classify by whole days from `now`. */
export function dayOffset(isoDate: string, now: Date): number {
  const [y, m, d] = isoDate.split('-').map(Number);
  const a = Date.UTC(y, m - 1, d);
  const b = Date.UTC(now.getFullYear(), now.getMonth(), now.getDate());
  return Math.round((a - b) / 86_400_000);
}

/** `Y-m-d H:i` — the wall-clock format of POST /telemed/bookings starts_at. */
export const startsAtParam = (date: string, time: string): string => `${date} ${time}`;

/** True once the video link may be opened (10 min before the start until the end). */
export function videoWindowOpen(opensAt: string, closesAt: string, now: number): boolean {
  return now >= Date.parse(opensAt) && now < Date.parse(closesAt);
}

/** «شنبه، ۸ مهر · ۱۸:۰۰» — the visit's wall-clock slot in the locale's calendar. */
export function whenLabel(date: string, time: string, locale: Locale): string {
  return `${formatWeekdayDayMonth(fromApiDate(date), locale)} · ${formatClock(time, locale)}`;
}
