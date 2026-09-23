import type { CheckupResult } from '@/entities/checkup';

/** What the MarkDone sheet (v14_MarkDone) and the self-exam «این ماه انجام دادم» submit. */
export interface CheckupRecordInput {
  /** `Y-m-d`, not in the future (the server re-validates). */
  doneOn: string;
  result: CheckupResult;
  /** Self-exam finding keys (⊆ the type's `finding_options`). */
  findings?: string[];
  note?: string | null;
  /** `Y-m-d` — the «تغییر» override of the computed next due date; null clears it. */
  nextDueOn?: string | null;
}

export type CheckupRecordPatch = Partial<CheckupRecordInput>;

/**
 * camelCase input → the API body. Undefined fields are omitted so a PUT stays
 * partial; findings are de-duplicated; a blank note becomes null.
 * `has_attachment` is decided by the mutation (it owns the on-device file), so
 * it is passed in rather than read from the form.
 */
export function toCheckupRecordBody(
  input: CheckupRecordPatch,
  hasAttachment?: boolean,
): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  if (input.doneOn !== undefined) body.done_on = input.doneOn;
  if (input.result !== undefined) body.result = input.result;
  if (input.findings !== undefined) body.findings = [...new Set(input.findings)];
  if (input.note !== undefined) body.note = input.note?.trim() || null;
  if (input.nextDueOn !== undefined) body.next_due_on = input.nextDueOn || null;
  if (hasAttachment !== undefined) body.has_attachment = hasAttachment;
  return body;
}

/**
 * Self-exam: «چیزی متفاوت نبود» (an exclusive option) clears the others and
 * any other finding clears it. Returns the new selection after tapping `key`.
 */
export function toggleFinding(
  selected: readonly string[],
  key: string,
  exclusiveKeys: readonly string[],
): string[] {
  if (selected.includes(key)) return selected.filter((k) => k !== key);
  if (exclusiveKeys.includes(key)) return [key];
  return [...selected.filter((k) => !exclusiveKeys.includes(k)), key];
}

/** Self-exam result: any real finding → `follow_up`, else `normal` (T-M4-09). */
export function selfExamResult(
  findings: readonly string[],
  exclusiveKeys: readonly string[],
): CheckupResult {
  return findings.some((k) => !exclusiveKeys.includes(k)) ? 'follow_up' : 'normal';
}
