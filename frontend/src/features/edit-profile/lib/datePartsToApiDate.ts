import { type DateParts, partsToDate, toApiDate } from '@/shared/lib/date';
import type { Locale } from '@/shared/i18n';

/**
 * Convert user-picked calendar parts into the API's Gregorian `YYYY-MM-DD`.
 *
 * The parts come from the wheels in whichever calendar the user's locale reads
 * (Jalali for `fa`, Gregorian for `en`), so the locale has to travel with them —
 * the same numbers mean different days in each calendar. The shared date layer
 * is the only module allowed to touch the date library (CLAUDE.md §7); it also
 * clamps a day that overflows the month (31 Esfand, 31 February) instead of
 * silently rolling into the next month.
 */
export function datePartsToApiDate(parts: DateParts, locale: Locale): string {
  return toApiDate(partsToDate(parts, locale));
}
