import type {
  DoseUnit,
  Meal,
  MedicationDuration,
  MedicationFrequency,
} from '@/entities/care-reminder';

/** One dose block as the form submits it. */
export interface DoseInput {
  /** `HH:MM`. */
  time: string;
  amount: number;
  unit: DoseUnit;
  meal: Meal | null;
}

/** What the medication form (frequency + dose blocks flow) submits. */
export interface MedicationInput {
  title: string;
  /** `Y-m-d`. */
  startsOn: string;
  frequency: MedicationFrequency;
  /** 1–6 blocks, distinct times. */
  doses: DoseInput[];
  duration: MedicationDuration;
  /** `Y-m-d`; required when `duration === 'until_date'`. */
  endsOn?: string | null;
  notify: boolean;
  notes?: string | null;
}

export type MedicationPatch = Partial<MedicationInput> & { isActive?: boolean };

/**
 * camelCase input → the API body. Undefined fields are omitted so a PUT stays
 * partial. The legacy `times` / `weekdays` / `form` / `dose` / `unit` / `amount`
 * are never sent: the server derives them from `doses`. `ends_on` is sent only
 * for `until_date`.
 */
export function toMedicationBody(input: MedicationPatch): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  if (input.title !== undefined) body.title = input.title.trim();
  if (input.startsOn !== undefined) body.starts_on = input.startsOn;
  if (input.frequency !== undefined) {
    body.frequency = { every: input.frequency.every, unit: input.frequency.unit };
  }
  if (input.doses !== undefined) {
    body.doses = input.doses.map((d) => ({
      time: d.time,
      amount: d.amount,
      unit: d.unit,
      meal: d.meal ?? null,
    }));
  }
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
