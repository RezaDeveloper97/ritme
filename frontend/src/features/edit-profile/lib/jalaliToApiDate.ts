import { type JalaliParts, jalaliMonthMatrix, toApiDate } from '@/shared/lib/date';

/**
 * Convert user-picked Jalali parts into the API's Gregorian `YYYY-MM-DD`.
 *
 * The shared date layer is the only module allowed to touch the date library
 * (CLAUDE.md §7), so the conversion goes through `jalaliMonthMatrix`, which
 * already maps every Jalali day of a month onto its Gregorian date. If the
 * picked day overflows the month (e.g. 31 Esfand), it clamps to the month's
 * last day rather than silently rolling into the next month.
 */
export function jalaliPartsToApiDate({ year, month, day }: JalaliParts): string {
  const cells = jalaliMonthMatrix(year, month)
    .flat()
    .filter((cell): cell is NonNullable<typeof cell> => cell !== null);
  const cell = cells.find((c) => c.day === day) ?? cells[cells.length - 1];
  return toApiDate(cell.date);
}
