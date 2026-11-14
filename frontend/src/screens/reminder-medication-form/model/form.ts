import { z } from 'zod';

import {
  DAY_EVERY_MAX,
  DOSE_UNITS,
  HOUR_EVERY_OPTIONS,
  MEALS,
  MEDICATION_DURATIONS,
  type DoseUnit,
  type FrequencyUnit,
  type Meal,
  type Medication,
  type MedicationDuration,
  type MedicationForm,
  type MedicationFrequency,
} from '@/entities/care-reminder';
import type { MedicationInput } from '@/features/manage-medication';

/*
 * Pure form logic for the medication form (frequency + dose blocks) — state
 * shape, frequency/dose helpers, zod validation mirroring the Go request rules
 * (backend-go/internal/care/request.go) and 422 → field mapping.
 *
 * Privacy (§11): nothing here logs; values only travel to the API.
 */

export const MAX_DOSES = 6;
export const AMOUNT_MIN = 1;
export const AMOUNT_MAX = 99;
/** The wheel's unit order (server enum order). */
export const DOSE_UNIT_OPTIONS: readonly DoseUnit[] = DOSE_UNITS;
export const FREQUENCY_UNIT_OPTIONS: readonly FrequencyUnit[] = ['day', 'hour'];

/** One editable dose block; a field is null until the user picks it. */
export interface DoseBlock {
  /** Stable React key (blocks can be removed from the middle). */
  key: string;
  /** `HH:MM` or null. */
  time: string | null;
  amount: number | null;
  unit: DoseUnit | null;
  meal: Meal | null;
}

export interface MedicationFormState {
  title: string;
  /** `Y-m-d`. */
  startsOn: string;
  frequency: MedicationFrequency;
  doses: DoseBlock[];
  duration: MedicationDuration;
  /** `Y-m-d` or null. */
  endsOn: string | null;
  notify: boolean;
  notes: string;
}

export type FormField =
  | 'title'
  | 'startsOn'
  | 'frequency'
  | 'doses'
  | 'duration'
  | 'endsOn'
  | 'notes';

/** Per-block errors: `time` covers the clock tile, `amount` the dose tile. */
export type DoseFieldErrors = Record<number, { time?: string; amount?: string }>;

let blockSeq = 0;
function nextKey(): string {
  blockSeq += 1;
  return `dose-${blockSeq}`;
}

export function emptyDose(): DoseBlock {
  return { key: nextKey(), time: null, amount: null, unit: null, meal: null };
}

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
    startsOn: todayApi,
    frequency: { every: 1, unit: 'day' },
    doses: [emptyDose()],
    duration: defaultDuration(pregnancy),
    endsOn: null,
    notify: true,
    notes: '',
  };
}

/** Old `form` → the closest dose unit (for a medication saved before dose blocks). */
const FORM_TO_UNIT: Record<MedicationForm, DoseUnit> = {
  tablet: 'pill',
  capsule: 'capsule',
  drops: 'drop',
  syrup: 'ml',
  injection: 'injection',
};

const clampAmount = (n: number) => Math.min(AMOUNT_MAX, Math.max(AMOUNT_MIN, Math.round(n) || AMOUNT_MIN));

export function fromMedication(med: Medication, todayApi: string): MedicationFormState {
  const frequency: MedicationFrequency = med.frequency ?? { every: 1, unit: 'day' };
  let doses: DoseBlock[];
  if (med.doses.length > 0) {
    doses = med.doses.map((d) => ({
      key: nextKey(),
      time: d.time,
      amount: clampAmount(d.amount),
      unit: d.unit,
      meal: d.meal,
    }));
  } else {
    const times = med.times.length ? med.times : ['08:00'];
    doses = times.slice(0, MAX_DOSES).map((time) => ({
      key: nextKey(),
      time,
      amount: clampAmount(med.amount || 1),
      unit: FORM_TO_UNIT[med.form] ?? 'pill',
      meal: null,
    }));
  }
  return {
    title: med.title,
    startsOn: med.startsOn ?? todayApi,
    frequency,
    doses,
    duration: med.duration,
    endsOn: med.duration === 'until_date' ? med.endsOn : null,
    notify: med.notify,
    notes: med.notes ?? '',
  };
}

// ── Frequency ────────────────────────────────────────────────────

/** The numbers the «هر …» wheel offers for a unit. */
export function everyOptions(unit: FrequencyUnit): number[] {
  return unit === 'hour'
    ? [...HOUR_EVERY_OPTIONS]
    : Array.from({ length: DAY_EVERY_MAX }, (_, i) => i + 1);
}

/** Keep `every` legal after the unit changes (e.g. 5 days → 4 hours). */
export function snapEvery(unit: FrequencyUnit, every: number): number {
  const options = everyOptions(unit);
  if (options.includes(every)) return every;
  let best = options[0]!;
  for (const o of options) if (Math.abs(o - every) < Math.abs(best - every)) best = o;
  return best;
}

// ── Dose blocks ──────────────────────────────────────────────────

export function isDoseComplete(d: DoseBlock): boolean {
  return d.time !== null && d.amount !== null && d.unit !== null;
}

/** Append an empty block (no-op at the cap). */
export function addDose(doses: readonly DoseBlock[]): DoseBlock[] {
  return doses.length >= MAX_DOSES ? [...doses] : [...doses, emptyDose()];
}

/** Remove a block; the first block (and a sole block) can't be removed. */
export function removeDose(doses: readonly DoseBlock[], key: string): DoseBlock[] {
  if (doses.length <= 1 || doses[0]?.key === key) return [...doses];
  return doses.filter((d) => d.key !== key);
}

export function patchDose(doses: readonly DoseBlock[], key: string, patch: Partial<Omit<DoseBlock, 'key'>>): DoseBlock[] {
  return doses.map((d) => (d.key === key ? { ...d, ...patch } : d));
}

/** Tapping the selected meal again clears it. */
export function toggleMeal(current: Meal | null, tapped: Meal): Meal | null {
  return current === tapped ? null : tapped;
}

/** The save button's gate: a name and at least one complete dose. */
export function canSubmit(state: MedicationFormState): boolean {
  return state.title.trim() !== '' && state.doses.some(isDoseComplete);
}

// ── Slots ────────────────────────────────────────────────────────

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

// ── Validation (mirrors request.go) ──────────────────────────────

/** i18n keys under `care.medicationForm.errors`. */
export type FormErrorKey =
  | 'titleRequired'
  | 'tooLong'
  | 'timesDuplicate'
  | 'timesInvalid'
  | 'doseRequired'
  | 'frequencyInvalid'
  | 'dosesRange'
  | 'amountRange'
  | 'dateInvalid'
  | 'endsOnRequired'
  | 'endsOnBeforeStart';

const API_DATE = /^\d{4}-\d{2}-\d{2}$/;
const SLOT = /^([01]\d|2[0-3]):[0-5]\d$/;

const doseSchema = z.object({
  time: z.string({ message: 'timesInvalid' }).regex(SLOT, 'timesInvalid'),
  amount: z
    .number({ message: 'amountRange' })
    .int('amountRange')
    .min(AMOUNT_MIN, 'amountRange')
    .max(AMOUNT_MAX, 'amountRange'),
  unit: z.enum(DOSE_UNITS, { message: 'amountRange' }),
  meal: z.enum(MEALS).nullable(),
});

export const medicationFormSchema = z
  .object({
    title: z.string().trim().min(1, 'titleRequired').max(255, 'tooLong'),
    startsOn: z.string().regex(API_DATE, 'dateInvalid'),
    frequency: z
      .object({ every: z.number().int(), unit: z.enum(['day', 'hour']) })
      .superRefine((f, ctx) => {
        const ok =
          f.unit === 'hour'
            ? (HOUR_EVERY_OPTIONS as readonly number[]).includes(f.every)
            : f.every >= 1 && f.every <= DAY_EVERY_MAX;
        if (!ok) ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'frequencyInvalid' });
      }),
    doses: z
      .array(doseSchema)
      .min(1, 'doseRequired')
      .max(MAX_DOSES, 'dosesRange')
      .refine((list) => new Set(list.map((d) => d.time)).size === list.length, 'timesDuplicate'),
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

export interface FormErrors {
  fields: Partial<Record<FormField, FormErrorKey>>;
  doses: Record<number, { time?: FormErrorKey; amount?: FormErrorKey }>;
}

export type ValidationResult =
  | { ok: true; input: MedicationInput }
  | { ok: false; errors: FormErrors };

/** Validate and convert to the feature's `MedicationInput`. */
export function validateForm(state: MedicationFormState): ValidationResult {
  const parsed = medicationFormSchema.safeParse(state);
  if (!parsed.success) {
    const errors: FormErrors = { fields: {}, doses: {} };
    for (const issue of parsed.error.issues) {
      const [head, index, leaf] = issue.path;
      const key = issue.message as FormErrorKey;
      if (head === 'doses' && typeof index === 'number') {
        const slot = (errors.doses[index] ??= {});
        const which = leaf === 'time' ? 'time' : 'amount';
        slot[which] ??= key;
      } else if (typeof head === 'string') {
        const field = head as FormField;
        errors.fields[field] ??= key;
      }
    }
    // A duplicate time is a list-level issue: pin it on the later block's clock tile.
    if (errors.fields.doses === 'timesDuplicate') {
      const seen = new Set<string>();
      state.doses.forEach((d, i) => {
        if (d.time && seen.has(d.time)) errors.doses[i] = { ...errors.doses[i], time: 'timesDuplicate' };
        if (d.time) seen.add(d.time);
      });
      delete errors.fields.doses;
    }
    return { ok: false, errors };
  }
  const v = parsed.data;
  return {
    ok: true,
    input: {
      title: v.title,
      startsOn: v.startsOn,
      frequency: v.frequency,
      doses: v.doses,
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
  starts_on: 'startsOn',
  frequency: 'frequency',
  doses: 'doses',
  duration: 'duration',
  ends_on: 'endsOn',
  notes: 'notes',
};

/**
 * `{errors: {"doses.1.time": ["…"], "frequency.every": ["…"]}}` → first
 * (already localized) message per form field / dose block. Keys the form
 * doesn't render are returned under `unknown` so the screen can still show something.
 */
export function mapServerErrors(body: unknown): {
  fields: FieldErrors;
  doses: DoseFieldErrors;
  unknown: string | null;
} {
  const fields: FieldErrors = {};
  const doses: DoseFieldErrors = {};
  let unknown: string | null = null;
  const errors =
    body && typeof body === 'object' && 'errors' in body
      ? (body as { errors: unknown }).errors
      : null;
  if (!errors || typeof errors !== 'object') return { fields, doses, unknown };
  for (const [key, value] of Object.entries(errors as Record<string, unknown>)) {
    const message = Array.isArray(value) ? value.find((m) => typeof m === 'string') : value;
    if (typeof message !== 'string') continue;
    const [head, index, leaf] = key.split('.');
    if (head === 'doses' && index !== undefined && /^\d+$/.test(index)) {
      const slot = (doses[Number(index)] ??= {});
      const which = leaf === 'time' ? 'time' : 'amount';
      slot[which] ??= message;
      continue;
    }
    const field = SERVER_FIELDS[head ?? ''];
    if (field) {
      fields[field] ??= message;
    } else if (!unknown) {
      unknown = message;
    }
  }
  return { fields, doses, unknown };
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
