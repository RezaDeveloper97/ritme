import { z } from 'zod';

import {
  ALL_WEEKDAYS,
  MEDICATION_DURATIONS,
  MEDICATION_FORMS,
  type Medication,
  type MedicationDuration,
  type MedicationForm,
  type Weekday,
} from '@/entities/care-reminder';
import type { MedicationInput } from '@/features/manage-medication';

/*
 * Pure form logic for v13_AddMedication — state shape, the times ↔ count
 * rule, the weekday summary, zod validation mirroring the Go request rules
 * (backend-go/internal/care/request.go) and 422 → field mapping.
 *
 * Privacy (§11): nothing here logs; values only travel to the API.
 */

export const MAX_TIMES = 4;
export const TIMES_OPTIONS = [1, 2, 3, 4] as const;
/** Slot n's default when the count grows (Scope §2). */
export const DEFAULT_TIMES = ['08:00', '20:00', '14:00', '23:00'] as const;
export const AMOUNT_MIN = 1;
export const AMOUNT_MAX = 10;

export interface MedicationFormState {
  title: string;
  dose: string;
  unit: string;
  form: MedicationForm;
  times: string[];
  weekdays: Weekday[];
  amount: number;
  /** `Y-m-d`. */
  startsOn: string;
  duration: MedicationDuration;
  /** `Y-m-d` or null. */
  endsOn: string | null;
  notify: boolean;
  notes: string;
}

export type FormField =
  | 'title'
  | 'dose'
  | 'unit'
  | 'form'
  | 'times'
  | 'weekdays'
  | 'amount'
  | 'startsOn'
  | 'duration'
  | 'endsOn'
  | 'notes';

/**
 * The duration a new medication starts with: «تا پایان بارداری» for a pregnant
 * user (the `v13` artboard's default), else «بدون تاریخ پایان».
 */
export function defaultDuration(pregnancy: boolean): MedicationDuration {
  return pregnancy ? 'pregnancy_end' : 'ongoing';
}

export function emptyForm(todayApi: string, pregnancy = false): MedicationFormState {
  return {
    title: '',
    dose: '',
    unit: 'mg',
    form: 'tablet',
    times: [DEFAULT_TIMES[0]],
    weekdays: [...ALL_WEEKDAYS],
    amount: 1,
    startsOn: todayApi,
    duration: defaultDuration(pregnancy),
    endsOn: null,
    notify: true,
    notes: '',
  };
}

export function fromMedication(med: Medication, todayApi: string): MedicationFormState {
  return {
    title: med.title,
    dose: med.dose ?? '',
    unit: med.unit || 'mg',
    form: med.form,
    times: med.times.length ? [...med.times] : [DEFAULT_TIMES[0]],
    weekdays: med.weekdays.length ? [...med.weekdays] : [...ALL_WEEKDAYS],
    amount: med.amount || 1,
    startsOn: med.startsOn ?? todayApi,
    duration: med.duration,
    endsOn: med.duration === 'until_date' ? med.endsOn : null,
    notify: med.notify,
    notes: med.notes ?? '',
  };
}

// ── Times ↔ count ────────────────────────────────────────────────

/**
 * Resize the slot list to `count` (1–4). Shrinking drops the last rows;
 * growing appends the first default (08:00, 20:00, 14:00, 23:00 in that
 * order) that isn't already taken, so a new row never duplicates one.
 */
export function setTimesCount(times: readonly string[], count: number): string[] {
  const n = Math.min(MAX_TIMES, Math.max(1, Math.round(count)));
  if (times.length >= n) return times.slice(0, n);
  const next = [...times];
  while (next.length < n) {
    const free = DEFAULT_TIMES.find((slot) => !next.includes(slot));
    next.push(free ?? DEFAULT_TIMES[next.length] ?? '08:00');
  }
  return next;
}

export function replaceTime(times: readonly string[], index: number, value: string): string[] {
  return times.map((slot, i) => (i === index ? value : slot));
}

export function toSlot(hour: number, minute: number): string {
  return `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}`;
}

export function parseSlot(slot: string): { hour: number; minute: number } {
  const [h, m] = slot.split(':').map(Number);
  return {
    hour: Number.isFinite(h) ? Math.min(23, Math.max(0, h)) : 0,
    minute: Number.isFinite(m) ? Math.min(59, Math.max(0, m)) : 0,
  };
}

/** The slot as the form's time button shows it: 24-hour, no leading zero (`08:00` → `8:00`). */
export function slotLabel(slot: string): string {
  const { hour, minute } = parseSlot(slot);
  return `${hour}:${String(minute).padStart(2, '0')}`;
}

export type SlotPeriod = 'morning' | 'noon' | 'evening' | 'night';

/** 05–11 morning, 12–14 noon, 15–18 evening, 19–04 night (same as the reminders list). */
export function slotPeriod(slot: string): SlotPeriod {
  const { hour } = parseSlot(slot);
  if (hour >= 5 && hour < 12) return 'morning';
  if (hour >= 12 && hour < 15) return 'noon';
  if (hour >= 15 && hour < 19) return 'evening';
  return 'night';
}

// ── Weekdays ─────────────────────────────────────────────────────

export function toggleWeekday(days: readonly Weekday[], day: Weekday): Weekday[] {
  const next = days.includes(day) ? days.filter((d) => d !== day) : [...days, day];
  return [...next].sort((a, b) => a - b);
}

/** Display order: Saturday-first for the Jalali calendar, Sunday-first otherwise. */
export function weekdayOrder(saturdayFirst: boolean): Weekday[] {
  return saturdayFirst ? [...ALL_WEEKDAYS] : [1, 2, 3, 4, 5, 6, 0];
}

export type WeekdaySummary =
  | { kind: 'everyDay' }
  | { kind: 'none' }
  | { kind: 'days'; count: number; days: Weekday[] };

/** All seven on = «هر روز»; otherwise the count (and the days, sorted). */
export function weekdaySummary(days: readonly Weekday[]): WeekdaySummary {
  const unique = [...new Set(days)].sort((a, b) => a - b);
  if (unique.length === 0) return { kind: 'none' };
  if (unique.length === ALL_WEEKDAYS.length) return { kind: 'everyDay' };
  return { kind: 'days', count: unique.length, days: unique };
}

// ── Validation (mirrors request.go) ──────────────────────────────

/** i18n keys under `care.medicationForm.errors`. */
export type FormErrorKey =
  | 'titleRequired'
  | 'tooLong'
  | 'timesDuplicate'
  | 'timesInvalid'
  | 'weekdaysRequired'
  | 'amountRange'
  | 'dateInvalid'
  | 'endsOnRequired'
  | 'endsOnBeforeStart';

const API_DATE = /^\d{4}-\d{2}-\d{2}$/;
const SLOT = /^([01]\d|2[0-3]):[0-5]\d$/;

export const medicationFormSchema = z
  .object({
    title: z.string().trim().min(1, 'titleRequired').max(255, 'tooLong'),
    dose: z.string().trim().max(50, 'tooLong'),
    unit: z.string().trim().max(50, 'tooLong'),
    form: z.enum(MEDICATION_FORMS),
    times: z
      .array(z.string().regex(SLOT, 'timesInvalid'))
      .min(1, 'timesInvalid')
      .max(MAX_TIMES, 'timesInvalid')
      .refine((times) => new Set(times).size === times.length, 'timesDuplicate'),
    weekdays: z.array(z.number().int().min(0).max(6)).min(1, 'weekdaysRequired').max(7),
    amount: z.number().int('amountRange').min(AMOUNT_MIN, 'amountRange').max(AMOUNT_MAX, 'amountRange'),
    startsOn: z.string().regex(API_DATE, 'dateInvalid'),
    duration: z.enum(MEDICATION_DURATIONS),
    endsOn: z.string().regex(API_DATE, 'dateInvalid').nullable(),
    notify: z.boolean(),
    notes: z.string().max(2000, 'tooLong'),
  })
  .superRefine((value, ctx) => {
    if (value.duration !== 'until_date') return;
    if (!value.endsOn) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['endsOn'], message: 'endsOnRequired' });
    } else if (value.endsOn < value.startsOn) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['endsOn'], message: 'endsOnBeforeStart' });
    }
  });

export type FieldErrors = Partial<Record<FormField, string>>;

export type ValidationResult =
  | { ok: true; input: MedicationInput }
  | { ok: false; errors: Partial<Record<FormField, FormErrorKey>> };

/** Validate and convert to the feature's `MedicationInput`. */
export function validateForm(state: MedicationFormState): ValidationResult {
  const parsed = medicationFormSchema.safeParse(state);
  if (!parsed.success) {
    const errors: Partial<Record<FormField, FormErrorKey>> = {};
    for (const issue of parsed.error.issues) {
      const field = issue.path[0] as FormField | undefined;
      if (field && !errors[field]) errors[field] = issue.message as FormErrorKey;
    }
    return { ok: false, errors };
  }
  const v = parsed.data;
  return {
    ok: true,
    input: {
      title: v.title,
      dose: v.dose,
      unit: v.unit,
      form: v.form,
      times: v.times,
      weekdays: v.weekdays as Weekday[],
      amount: v.amount,
      startsOn: v.startsOn,
      duration: v.duration,
      endsOn: v.duration === 'until_date' ? v.endsOn : null,
      notify: v.notify,
      notes: v.notes.trim() || null,
    },
  };
}

// ── Server 422 → fields ──────────────────────────────────────────

const SERVER_FIELDS: Record<string, FormField> = {
  title: 'title',
  dose: 'dose',
  unit: 'unit',
  form: 'form',
  times: 'times',
  weekdays: 'weekdays',
  amount: 'amount',
  starts_on: 'startsOn',
  duration: 'duration',
  ends_on: 'endsOn',
  notes: 'notes',
};

/**
 * `{errors: {"times.1": ["…"], "ends_on": ["…"]}}` → first (already localized)
 * message per form field. Keys the form doesn't render are returned under
 * `unknown` so the screen can still show something.
 */
export function mapServerErrors(body: unknown): { fields: FieldErrors; unknown: string | null } {
  const fields: FieldErrors = {};
  let unknown: string | null = null;
  const errors =
    body && typeof body === 'object' && 'errors' in body
      ? (body as { errors: unknown }).errors
      : null;
  if (!errors || typeof errors !== 'object') return { fields, unknown };
  for (const [key, value] of Object.entries(errors as Record<string, unknown>)) {
    const message = Array.isArray(value) ? value.find((m) => typeof m === 'string') : value;
    if (typeof message !== 'string') continue;
    const field = SERVER_FIELDS[key.split('.')[0]];
    if (field) {
      if (!fields[field]) fields[field] = message;
    } else if (!unknown) {
      unknown = message;
    }
  }
  return { fields, unknown };
}

// ── Navigation ───────────────────────────────────────────────────

/** Where save/delete return to: the home card or the reminders hub (default). */
export function returnHref(from: string | undefined): '/home' | '/reminders' | '/companion' {
  if (from === 'home') return '/home';
  // B-N4-06: «افزودن برای …» on the companion panel home.
  if (from === 'companion') return '/companion';
  return '/reminders';
}

/** Durations offered: «تا پایان بارداری» only in pregnancy mode (or if already stored). */
export function durationOptions(
  pregnancy: boolean,
  current: MedicationDuration,
): MedicationDuration[] {
  return MEDICATION_DURATIONS.filter(
    (d) => d !== 'pregnancy_end' || pregnancy || current === 'pregnancy_end',
  );
}
