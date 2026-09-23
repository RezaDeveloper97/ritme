import type { CheckupPerformer } from '@/entities/checkup';

/** What the custom-checkup form (`/checkups/custom/new`, T-M4-07) submits. */
export interface CustomCheckupInput {
  title: string;
  /** One of `CUSTOM_INTERVAL_MONTHS` in the UI; the API validates the range. */
  intervalMonths: number;
  performedBy: CheckupPerformer;
  note?: string | null;
  /** `Y-m-d`, optional — seeds the first record so the next due date is known. */
  lastDoneOn?: string | null;
}

export type CustomCheckupPatch = Partial<CustomCheckupInput>;

/**
 * camelCase input → the API body; undefined fields are omitted so a PUT stays
 * partial, blanks become null.
 */
export function toCustomCheckupBody(input: CustomCheckupPatch): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  if (input.title !== undefined) body.title = input.title.trim();
  if (input.intervalMonths !== undefined) body.interval_months = Math.trunc(input.intervalMonths);
  if (input.performedBy !== undefined) body.performed_by = input.performedBy;
  if (input.note !== undefined) body.note = input.note?.trim() || null;
  if (input.lastDoneOn !== undefined) body.last_done_on = input.lastDoneOn || null;
  return body;
}
