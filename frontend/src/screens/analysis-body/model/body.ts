import type { WeekdayKey } from '@/shared/lib/date';

/** The API's weekday order (backend-go body.go: Saturday first). */
const API_ORDER: readonly WeekdayKey[] = ['sat', 'sun', 'mon', 'tue', 'wed', 'thu', 'fri'];

/** Re-order the Saturday-first averages into the locale's week (`weekdayKeys(locale)`). */
export function weekdayValues(
  bySaturdayFirst: ReadonlyArray<number | null>,
  order: readonly WeekdayKey[],
): Array<{ key: WeekdayKey; value: number | null }> {
  return order.map((key) => ({ key, value: bySaturdayFirst[API_ORDER.indexOf(key)] ?? null }));
}

/** Index of the largest value (the bar drawn solid), -1 when there is none. */
export function maxIndex(values: ReadonlyArray<number | null>): number {
  let at = -1;
  values.forEach((v, i) => {
    if (v !== null && v > 0 && (at === -1 || v > (values[at] as number))) at = i;
  });
  return at;
}

/** «−۰٫۶» style signed one-decimal parts (U+2212 minus, as the artboards print it). */
export function signed(value: number): { sign: '−' | '+' | ''; abs: string } {
  const rounded = Math.round(value * 10) / 10;
  if (rounded === 0) return { sign: '', abs: '0' };
  return { sign: rounded < 0 ? '−' : '+', abs: Math.abs(rounded).toFixed(1).replace(/\.0$/, '') };
}

/** The headline weight: the latest 7-day average (what the card title promises), else the latest weigh-in. */
export function headlineWeight(points: ReadonlyArray<{ value: number | null; avg7: number | null }>, current: number | null): number | null {
  for (let i = points.length - 1; i >= 0; i--) {
    if (points[i].avg7 !== null) return points[i].avg7;
  }
  return current;
}
