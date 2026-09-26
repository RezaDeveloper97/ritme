import type { Appointment, RemindBefore } from '@/entities/care-reminder';
import { type IcsEvent } from '@/shared/lib/ics';

/*
 * Pure view-state for the appointment detail (v13_AppointmentDetail) — no
 * React, no locale, so countdown, pregnancy week, maps link and the calendar
 * file are unit-tested on their own.
 */

export const REMIND_BEFORE_MINUTES: Record<RemindBefore, number> = {
  '1h': 60,
  '3h': 180,
  '1d': 1440,
  '2d': 2880,
};

export interface ScheduledParts {
  /** `Y-m-d`. */
  date: string;
  /** `HH:MM`. */
  time: string;
}

/** Tehran wall-clock `Y-m-d H:i:s` → date + `HH:MM`; null when absent/garbled. */
export function splitScheduled(value: string | null): ScheduledParts | null {
  const m = /^(\d{4}-\d{2}-\d{2})[ T](\d{2}:\d{2})/.exec(value ?? '');
  return m ? { date: m[1]!, time: m[2]! } : null;
}

/** Whole calendar days from `today` to the appointment date (both `Y-m-d`). */
export function daysUntil(date: string, today: string): number {
  const a = Date.parse(`${date}T00:00:00Z`);
  const b = Date.parse(`${today}T00:00:00Z`);
  return Math.round((a - b) / 86_400_000);
}

/**
 * Gestational week on the appointment date, from today's gestational age
 * (`totalDays`, as the pregnancy entity reports it) shifted by the countdown.
 * Null when outside 1..42 or unknown.
 */
export function pregnancyWeekAt(totalDaysToday: number | null | undefined, daysAhead: number): number | null {
  if (totalDaysToday == null || !Number.isFinite(totalDaysToday)) return null;
  const week = Math.floor((totalDaysToday + daysAhead) / 7) + 1;
  return week >= 1 && week <= 42 ? week : null;
}

/** «مسیریابی» — a generic maps search for an in-person address; null otherwise. */
export function mapsHref(appt: Pick<Appointment, 'kind' | 'location'>): string | null {
  const place = appt.location?.trim();
  if (appt.kind !== 'in_person' || !place) return null;
  return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(place)}`;
}

/** The `.ics` event for «افزودن به تقویم» (alarm = remind_before); null without a time. */
export function appointmentIcsEvent(
  appt: Appointment,
  summary: string,
  description: string | null,
): IcsEvent | null {
  if (!splitScheduled(appt.scheduledAt)) return null;
  return {
    uid: `appointment-${appt.id}@ritme.app`,
    title: summary,
    description,
    location: appt.location,
    start: appt.scheduledAt as string,
    durationMinutes: 60,
    alarmMinutesBefore: appt.isActive ? REMIND_BEFORE_MINUTES[appt.remindBefore] : null,
  };
}

/** When the reminder fires (wall clock), for «۱ روز قبل · ۴ مهر ساعت ۱۰:۳۰». */
export function reminderMoment(when: ScheduledParts, before: RemindBefore): ScheduledParts {
  const [y, mo, d] = when.date.split('-').map(Number);
  const [h, mi] = when.time.split(':').map(Number);
  const t = new Date(Date.UTC(y!, mo! - 1, d!, h!, mi! - REMIND_BEFORE_MINUTES[before]));
  const pad = (n: number) => String(n).padStart(2, '0');
  return {
    date: `${t.getUTCFullYear()}-${pad(t.getUTCMonth() + 1)}-${pad(t.getUTCDate())}`,
    time: `${pad(t.getUTCHours())}:${pad(t.getUTCMinutes())}`,
  };
}
