import type { CityServiceService, CityServiceSlot } from '@/entities/city-service';
import { diffInDays, fromApiDate, today } from '@/shared/lib/date';

/* CB-DIR-08 — pure helpers of the place page (formatting rules; no React). */

/** A venue-declared price in toman as `{ unit, value }`: thousands under a million, millions above. */
export function priceParts(toman: number): { unit: 'thousand' | 'million'; value: number } {
  if (toman >= 1_000_000) return { unit: 'million', value: Math.round((toman / 1_000_000) * 10) / 10 };
  return { unit: 'thousand', value: Math.round(toman / 1000) };
}

/** Splits an API timestamp (`2026-10-12T17:00:00…` or `2026-10-12 17:00:00`, Tehran wall clock) into day + `HH:mm`. */
export function slotParts(startsAt: string): { day: string; time: string } | null {
  const m = /^(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2})/.exec(startsAt);
  return m ? { day: m[1], time: m[2] } : null;
}

/** 0 = today, 1 = tomorrow, otherwise another number (the caller prints the weekday). */
export function slotDayOffset(day: string, now: Date = today()): number {
  return diffInDays(fromApiDate(day), now);
}

/** Months → «۶ ماه» / «۲ سال» pieces. */
export function monthsLabelParts(months: number): { unit: 'month' | 'year'; n: number } {
  return months >= 24 && months % 12 === 0 ? { unit: 'year', n: months / 12 } : { unit: 'month', n: months };
}

/** The cheapest declared price of a venue's services (the sticky bar's «از …»). */
export function minPrice(services: CityServiceService[]): number | null {
  const prices = services.map((s) => s.price_toman).filter((p): p is number => p !== null && p > 0);
  return prices.length ? Math.min(...prices) : null;
}

/** A slot can be picked while it has a seat. */
export const bookable = (s: CityServiceSlot): boolean => !s.is_full && s.seats_left > 0;

/** Only http(s) goes out as a link (the server validates too). */
export function safeWebsite(url: string | null): string | null {
  if (!url) return null;
  try {
    const u = new URL(url);
    return u.protocol === 'http:' || u.protocol === 'https:' ? u.toString() : null;
  } catch {
    return null;
  }
}

/** Neshan location link for a place. TODO(CB-DIR-08): confirm the final Neshan deep-link format. */
export const neshanHref = (lat: number, lng: number): string => `https://neshan.org/maps/@${lat},${lng},17z`;
