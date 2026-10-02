import { shiftMonth } from '@/shared/lib/date';

/** A calendar month in the locale's calendar (Jalali for fa). */
export interface PickedMonth {
  year: number;
  month: number;
}

function key({ year, month }: PickedMonth): number {
  return year * 12 + month;
}

/** Whether the picker may step forward: the last period is never in the future. */
export function canStepForward(picked: PickedMonth | null, now: PickedMonth): boolean {
  return picked !== null && key(picked) < key(now);
}

/**
 * Step the last-period picker. Unset → «back» picks last month and «forward»
 * the current one (nothing is guessed before she touches it). Never past the
 * current month: the API refuses a future last period.
 */
export function stepMonth(picked: PickedMonth | null, now: PickedMonth, delta: number): PickedMonth | null {
  const base = picked ?? now;
  const next = shiftMonth(base.year, base.month, picked ? delta : Math.min(0, delta));
  return key(next) <= key(now) ? next : picked;
}
