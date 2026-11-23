import { DOSE_UNITS, type DoseUnit, type Meal, type Medication, type Weekday } from './types';

/** How a medication's weekdays are worded on every reminders surface. */
export type MedicationSchedule =
  | { kind: 'everyDay' }
  | { kind: 'everyOtherDay' }
  | { kind: 'daysPerWeek'; count: number };

/**
 * All seven (or none) = every day; three or more days each two apart
 * (ش/د/چ/ج, ی/س/پ) = «یک روز در میان»; anything else is a count. The one rule
 * for the /reminders screen and the legacy Profile → یادآورها sheet (whose
 * `recurrence` column can only say `daily` / `weekly`).
 */
export function medicationSchedule(weekdays: readonly number[]): MedicationSchedule {
  const days = [...new Set(weekdays)].sort((a, b) => a - b);
  if (days.length === 0 || days.length >= 7) return { kind: 'everyDay' };
  if (days.length >= 3 && days.every((d, i) => i === 0 || d - days[i - 1]! === 2)) {
    return { kind: 'everyOtherDay' };
  }
  return { kind: 'daysPerWeek', count: days.length };
}

/** How a medication's `frequency` is worded («هر روز» / «هر ۳ روز» / «هر ۲ ساعت»). */
export type FrequencySummary =
  | { kind: 'everyDay' }
  | { kind: 'everyDays'; count: number }
  | { kind: 'everyHours'; count: number };

export function frequencySummary(frequency: { every: number; unit: 'day' | 'hour' }): FrequencySummary {
  if (frequency.unit === 'hour') return { kind: 'everyHours', count: frequency.every };
  return frequency.every <= 1 ? { kind: 'everyDay' } : { kind: 'everyDays', count: frequency.every };
}

/** A today-dose's amount line parts, or null when the API sent no amount (old medication). */
export function doseDetail(dose: {
  amount: number | null;
  unit: string | null;
  unitLabel: string | null;
  meal: Meal | null;
}): { amount: number; unit: DoseUnit | null; unitLabel: string | null; meal: Meal | null } | null {
  if (dose.amount === null) return null;
  const unit = (DOSE_UNITS as readonly string[]).includes(dose.unit ?? '') ? (dose.unit as DoseUnit) : null;
  if (!unit && !dose.unitLabel) return null;
  return { amount: dose.amount, unit, unitLabel: dose.unitLabel, meal: dose.meal };
}

// ── Next dose ────────────────────────────────────────────────────

const LOOKAHEAD_DAYS = 31;

/** `Y-m-d` (optionally with a time part) → local midnight of that day, null when unparseable. */
function dayStart(value: string | null): Date | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(value ?? '');
  return m ? new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3])) : null;
}

/** Whether `day` (any time on it) is on the medication's calendar — the day rules of {@link nextDoseAt}, slots aside. */
function isScheduledDay(
  med: Pick<Medication, 'weekdays' | 'frequency' | 'startsOn' | 'endsOn'>,
  day: Date,
): boolean {
  const day0 = new Date(day.getFullYear(), day.getMonth(), day.getDate());
  const start = dayStart(med.startsOn);
  const end = dayStart(med.endsOn);
  if (start && day0 < start) return false;
  if (end && day0 > end) return false;
  const everyDays = med.frequency && med.frequency.unit === 'day' && med.frequency.every > 1 ? med.frequency.every : 1;
  if (everyDays > 1 && start) {
    const since = Math.round((day0.getTime() - start.getTime()) / 86_400_000);
    if (since % everyDays !== 0) return false;
  }
  const weekdays = med.frequency ? [] : [...new Set(med.weekdays)];
  return !(weekdays.length > 0 && weekdays.length < 7 && !weekdays.includes(((day0.getDay() + 1) % 7) as Weekday));
}

/**
 * Whether an active medication with at least one time slot has a dose on `day` (the calendar's day list). Same day
 * rules as {@link nextDoseAt}: start/end dates, «every N days» from the start, else the weekdays.
 */
export function isDoseDay(
  med: Pick<Medication, 'isActive' | 'times' | 'weekdays' | 'frequency' | 'startsOn' | 'endsOn'>,
  day: Date,
): boolean {
  return med.isActive && med.times.length > 0 && isScheduledDay(med, day);
}

/**
 * The next moment a medication is due, or null when there is none within ~31
 * days (inactive, ended, no slots). Pure wall-clock arithmetic on `now`'s own
 * calendar day, like the rest of the slice (the API sends Tehran wall-clock
 * dates and the app runs on the user's device clock).
 *
 * A day is a dose day when it is on/after `startsOn`, on/before `endsOn`, and:
 * `frequency` every N days → whole days since `startsOn` divisible by N (hour
 * frequencies and N = 1 are every day); without `frequency`, `weekdays`
 * (Saturday = 0; empty = every day). A slot at exactly `now` counts as due.
 */
export function nextDoseAt(
  med: Pick<Medication, 'isActive' | 'times' | 'weekdays' | 'frequency' | 'startsOn' | 'endsOn'>,
  now: Date,
): Date | null {
  if (!med.isActive) return null;
  const slots = med.times
    .map((s) => /^(\d{1,2}):(\d{2})/.exec(s))
    .flatMap((m) => (m ? [{ h: Number(m[1]), min: Number(m[2]) }] : []))
    .sort((a, b) => a.h - b.h || a.min - b.min);
  if (slots.length === 0) return null;

  const end = dayStart(med.endsOn);
  const today0 = new Date(now.getFullYear(), now.getMonth(), now.getDate());

  for (let i = 0; i <= LOOKAHEAD_DAYS; i++) {
    const day = new Date(today0.getFullYear(), today0.getMonth(), today0.getDate() + i);
    if (end && day > end) return null;
    if (!isScheduledDay(med, day)) continue;
    for (const s of slots) {
      const at = new Date(day.getFullYear(), day.getMonth(), day.getDate(), s.h, s.min);
      if (at.getTime() >= now.getTime()) return at;
    }
  }
  return null;
}

export type RelativeDose =
  | { kind: 'now' }
  | { kind: 'minutes'; n: number }
  | { kind: 'hours'; n: number }
  | { kind: 'days'; n: number };

/** «همین حالا» (< 1 min) · minutes (< 60) · hours (< 24 h) · days, rounded down, min 1. */
export function relativeDose(at: Date, now: Date): RelativeDose {
  const mins = Math.floor((at.getTime() - now.getTime()) / 60_000);
  if (mins < 1) return { kind: 'now' };
  if (mins < 60) return { kind: 'minutes', n: mins };
  if (mins < 24 * 60) return { kind: 'hours', n: Math.floor(mins / 60) };
  return { kind: 'days', n: Math.floor(mins / (24 * 60)) };
}
