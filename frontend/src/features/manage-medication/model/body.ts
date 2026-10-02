import type { MedicationDuration, MedicationForm, Weekday } from '@/entities/care-reminder';

/** What the medication form (v13_AddMedication) submits. */
export interface MedicationInput {
  title: string;
  dose: string;
  unit: string;
  form: MedicationForm;
  /** 1–4 `HH:MM` slots. */
  times: string[];
  weekdays: Weekday[];
  amount: number;
  /** `Y-m-d`. */
  startsOn: string;
  duration: MedicationDuration;
  /** `Y-m-d`; required when `duration === 'until_date'`. */
  endsOn?: string | null;
  notify: boolean;
  notes?: string | null;
}

export type MedicationPatch = Partial<MedicationInput> & { isActive?: boolean };

/**
 * camelCase input → the API body. Fields go flat (`dose`, `times`, …), the
 * server packs them into `reminders.meta`; undefined fields are omitted so a
 * PUT stays partial. Times are de-duplicated and sorted, weekdays sorted,
 * and `ends_on` is sent only for `until_date`.
 */
export function toMedicationBody(input: MedicationPatch): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  if (input.title !== undefined) body.title = input.title.trim();
  if (input.dose !== undefined) body.dose = input.dose.trim();
  if (input.unit !== undefined) body.unit = input.unit.trim();
  if (input.form !== undefined) body.form = input.form;
  if (input.times !== undefined) body.times = [...new Set(input.times)].sort();
  if (input.weekdays !== undefined) body.weekdays = [...new Set(input.weekdays)].sort((a, b) => a - b);
  if (input.amount !== undefined) body.amount = input.amount;
  if (input.startsOn !== undefined) body.starts_on = input.startsOn;
  if (input.duration !== undefined) {
    body.duration = input.duration;
    body.ends_on = input.duration === 'until_date' ? (input.endsOn ?? null) : null;
  } else if (input.endsOn !== undefined) {
    body.ends_on = input.endsOn;
  }
  if (input.notify !== undefined) body.notify = input.notify;
  if (input.notes !== undefined) body.notes = input.notes?.trim() || null;
  if (input.isActive !== undefined) body.is_active = input.isActive;
  return body;
}

/**
 * «ثبت برای …» (B-N4-06): names the owner a companion with edit records for
 * (`for_user_id`, B-N4-02). null/undefined = the viewer's own record, body unchanged.
 */
export function withForUser(body: Record<string, unknown>, forUserId: number | null | undefined): Record<string, unknown> {
  return forUserId ? { ...body, for_user_id: forUserId } : body;
}
