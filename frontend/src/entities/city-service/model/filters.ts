import type { DirectoryParams } from '../api/keys';
import type { CityServiceCatalogOption } from '../api/schema';

/** The filter sheet's value (CB-DIR-07). `age` is a place age-range code from the options catalog. */
export interface DirectoryFilters {
  age: string | null;
  openNow: boolean;
  amenities: string[];
  sort: 'rank' | 'distance' | 'rating';
}

export const EMPTY_FILTERS: DirectoryFilters = { age: null, openNow: false, amenities: [], sort: 'rank' };

/** Number of active filters (the sort is not a filter), for the badge on the «فیلترها» button. */
export const filterCount = (f: DirectoryFilters): number => (f.age ? 1 : 0) + (f.openNow ? 1 : 0) + f.amenities.length;

/** A catalog age range → one `age_months` to search with (the middle of its span; `meta` = {min_months,max_months}). */
export function ageMonthsOf(code: string | null, catalog: CityServiceCatalogOption[]): number | null {
  if (!code) return null;
  const meta = catalog.find((o) => o.code === code)?.meta as { min_months?: unknown; max_months?: unknown } | undefined;
  const min = typeof meta?.min_months === 'number' ? meta.min_months : null;
  const max = typeof meta?.max_months === 'number' ? meta.max_months : null;
  if (min === null && max === null) return null;
  return Math.min(216, Math.max(0, Math.round(((min ?? max ?? 0) + (max ?? min ?? 0)) / 2)));
}

/** Filters → request params. The position is only sent when the user tapped «near me» (never stored). */
export function toDirectoryParams(
  f: DirectoryFilters,
  catalog: CityServiceCatalogOption[],
  near: { lat: number; lng: number } | null,
): Pick<DirectoryParams, 'ageMonths' | 'openNow' | 'amenities' | 'sort' | 'lat' | 'lng'> {
  return {
    ageMonths: ageMonthsOf(f.age, catalog),
    openNow: f.openNow,
    amenities: f.amenities,
    sort: f.sort === 'distance' && !near ? 'rank' : f.sort,
    ...(near ? { lat: near.lat, lng: near.lng } : {}),
  };
}

export interface SlotWhen {
  day: 'today' | 'tomorrow' | 'date';
  date: Date;
}

/** `next_slot_at` → «امروز» / «فردا» / a date, relative to `now`. Null for a missing or unparsable value. */
export function slotWhen(iso: string | null, now: Date): SlotWhen | null {
  if (!iso) return null;
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return null;
  const startOf = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  const diff = Math.round((startOf(date) - startOf(now)) / 86_400_000);
  if (diff < 0) return null;
  return { day: diff === 0 ? 'today' : diff === 1 ? 'tomorrow' : 'date', date };
}
