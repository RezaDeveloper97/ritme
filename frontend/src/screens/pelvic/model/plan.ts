import type { PelvicLeak, PelvicUtiSymptom, PelvicWeekDay } from '@/entities/pelvic';
import type { Locale } from '@/shared/i18n';
import { fromApiDate, toApiDate, weekOf } from '@/shared/lib/date';
import type { WeekDotState } from '@/shared/ui';

/**
 * The seven dots of today's week in the locale's column order (Jalali Saturday → Friday, the API's own week;
 * Gregorian Sunday → Saturday). Trained → done, an earlier untrained day → missed, today untrained / later → future.
 * Days the API did not send (the Gregorian Saturday edge) are «not trained» when past.
 */
export function weekDotStates(
  days: readonly PelvicWeekDay[],
  today: string,
  locale: Locale,
): { states: WeekDotState[]; todayIndex: number } {
  const done = new Set(days.filter((d) => d.done).map((d) => d.date));
  const week = weekOf(fromApiDate(today), locale).map(toApiDate);
  const states = week.map((date): WeekDotState => {
    if (done.has(date)) return 'done';
    return date < today ? 'missed' : 'future';
  });
  return { states, todayIndex: week.indexOf(today) };
}

/** Whole minutes of a session, at least one. */
export function sessionMinutes(sessionSec: number): number {
  return Math.max(1, Math.round(sessionSec / 60));
}

/** A tap on the selected leak answer clears it; any other tap selects. */
export function nextLeak(current: PelvicLeak | null, tapped: PelvicLeak): PelvicLeak | null {
  return current === tapped ? null : tapped;
}

/** Toggles one UTI symptom, keeping the API order. */
export function toggleSymptom(
  current: readonly PelvicUtiSymptom[],
  symptom: PelvicUtiSymptom,
  order: readonly PelvicUtiSymptom[],
): PelvicUtiSymptom[] {
  const on = new Set(current);
  if (on.has(symptom)) on.delete(symptom);
  else on.add(symptom);
  return order.filter((s) => on.has(s));
}
