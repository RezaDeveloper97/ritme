import type { Appointment, Medication } from '@/entities/care-reminder';

/*
 * The legacy reminders sheet (Profile → یادآورها) lists the `GET /reminders`
 * rows. Medications and appointments made on the care screens (M3) carry more
 * than those legacy columns say, so the sheet reads the matching care row to
 * describe them the way /reminders does (stage regression B, B-1…B-3).
 */

export type RowSchedule =
  | { kind: 'everyDay' }
  | { kind: 'everyOtherDay' }
  | { kind: 'daysPerWeek'; count: number };

/**
 * A medication's weekdays as /reminders words them: all seven (or none) = every
 * day; three or more days each two apart (ش/د/چ/ج, ی/س/پ) = «یک روز در میان»;
 * anything else is a count. The legacy `recurrence` column can only say
 * `daily` / `weekly`, which misnamed an alternate-day plan «هفتگی».
 * Same rule as `screens/reminders` (`medicationSchedule`).
 */
export function rowSchedule(weekdays: readonly number[]): RowSchedule {
  const days = [...new Set(weekdays)].sort((a, b) => a - b);
  if (days.length === 0 || days.length >= 7) return { kind: 'everyDay' };
  if (days.length >= 3 && days.every((d, i) => i === 0 || d - days[i - 1]! === 2)) {
    return { kind: 'everyOtherDay' };
  }
  return { kind: 'daysPerWeek', count: days.length };
}

/** `"08:00"` / `"08:00:00"` → `"8:00"`: the hour without a leading zero, as the care screens show it. */
export function clockLabel(time: string): string {
  const match = /^(\d{1,2}):(\d{2})/.exec(time);
  if (!match) return time;
  return `${Number(match[1])}:${match[2]}`;
}

/** An appointment's «who، specialty» parts (the legacy subtitle joins them with « · »). */
export function appointmentParts(appt: Pick<Appointment, 'withWhom' | 'specialty'>): string[] {
  return [appt.withWhom, appt.specialty].filter((p): p is string => !!p && p.trim() !== '');
}

/** A cancelled appointment's reminder can't be switched back on (the API answers 422). */
export function isCancelled(appt: Pick<Appointment, 'status'> | undefined): boolean {
  return appt?.status === 'cancelled';
}

/** Care rows by the legacy reminder id (`GET /reminders` ids are strings). */
export function byReminderId<T extends Pick<Medication | Appointment, 'id'>>(
  rows: readonly T[] | undefined,
): Map<string, T> {
  return new Map((rows ?? []).map((r) => [String(r.id), r]));
}
