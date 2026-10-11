import type { CityServiceService, CityServiceSlot } from '@/entities/city-service';

/* CB-DIR-11 — pure helpers of the desktop place page. Same formatting rules as the app's place page (CB-DIR-08);
   duplicated here on purpose: screens never import each other (FSD) and these are tiny. */

/** A venue-declared price in toman as `{ unit, value }`: thousands under a million, millions above. */
export function priceParts(toman: number): { unit: 'thousand' | 'million'; value: number } {
  if (toman >= 1_000_000) return { unit: 'million', value: Math.round((toman / 1_000_000) * 10) / 10 };
  return { unit: 'thousand', value: Math.round(toman / 1000) };
}

/** Splits an API timestamp (Tehran wall clock) into day + `HH:mm`. */
export function slotParts(startsAt: string): { day: string; time: string } | null {
  const m = /^(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2})/.exec(startsAt);
  return m ? { day: m[1], time: m[2] } : null;
}

/** Months → «۶ ماه» / «۲ سال» pieces. */
export function monthsLabelParts(months: number): { unit: 'month' | 'year'; n: number } {
  return months >= 24 && months % 12 === 0 ? { unit: 'year', n: months / 12 } : { unit: 'month', n: months };
}

/** The cheapest declared price of a venue's services. */
export function minPrice(services: CityServiceService[]): number | null {
  const prices = services.map((s) => s.price_toman).filter((p): p is number => p !== null && p > 0);
  return prices.length ? Math.min(...prices) : null;
}

/** A slot can be picked while it has a seat. */
export const bookable = (s: CityServiceSlot): boolean => !s.is_full && s.seats_left > 0;

/** Only http(s) goes out as a link (the server validates too). */
export function safeHttp(url: string | null): string | null {
  if (!url) return null;
  try {
    const u = new URL(url);
    return u.protocol === 'http:' || u.protocol === 'https:' ? u.toString() : null;
  } catch {
    return null;
  }
}

/** Neshan location link for a place. */
export const neshanHref = (lat: number, lng: number): string => `https://neshan.org/maps/@${lat},${lng},17z`;

/** A `tel:` href from a venue phone (digits and a leading +). */
export const telHref = (phone: string): string => `tel:${phone.replace(/[^\d+]/g, '')}`;

/** `/places` ↔ app route map: where the header nav, CTA and the <1024px redirect lead. */
export const PLACES_HOME = '/places';
export const APP_DIRECTORY = '/city-services';
