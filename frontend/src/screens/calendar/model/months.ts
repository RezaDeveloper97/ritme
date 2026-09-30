import { addDays, diffInDays, toApiDate } from '@/shared/lib/date';

/** Gregorian `{year, month}` pairs a date range touches, in order. */
export function gregorianMonthsBetween(first: Date, last: Date): { year: number; month: number }[] {
  const out: { year: number; month: number }[] = [];
  const seen = new Set<string>();
  const span = Math.max(0, diffInDays(last, first));
  // Step a week at a time (plus the last day): a month can't hide between two probes.
  for (let i = 0; i <= span + 7; i += 7) {
    const iso = toApiDate(addDays(first, Math.min(i, span)));
    const key = iso.slice(0, 7);
    if (seen.has(key)) continue;
    seen.add(key);
    out.push({ year: Number(key.slice(0, 4)), month: Number(key.slice(5, 7)) });
  }
  return out;
}
