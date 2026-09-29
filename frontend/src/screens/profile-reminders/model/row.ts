import type { Appointment, Medication } from '@/entities/care-reminder';

/*
 * The legacy reminders sheet (Profile → یادآورها) lists the `GET /reminders`
 * rows. Medications and appointments made on the care screens (M3) carry more
 * than those legacy columns say, so the sheet reads the matching care row to
 * describe them the way /reminders does (stage regression B, B-1…B-3).
 */

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
