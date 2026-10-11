import type { CityServiceBooking, CityServiceCatalogOption } from '../api/schema';

/** Label of a place age-range code from the options catalog, falling back to nothing (the card then shows no chip). */
export function ageRangeLabel(codes: string[], catalog: CityServiceCatalogOption[]): string | null {
  const names = codes.map((c) => catalog.find((o) => o.code === c)?.name).filter((n): n is string => Boolean(n));
  return names.length > 0 ? names.slice(0, 2).join(' · ') : null;
}

/** The earliest still-open request (requested / confirmed) whose day is today or later — «رزرو بعدی». */
export function nextBooking(bookings: CityServiceBooking[], today: string): CityServiceBooking | null {
  const open = bookings.filter((b) => (b.status === 'requested' || b.status === 'confirmed') && b.preferred_date >= today);
  open.sort((a, b) => a.preferred_date.localeCompare(b.preferred_date));
  return open[0] ?? null;
}
