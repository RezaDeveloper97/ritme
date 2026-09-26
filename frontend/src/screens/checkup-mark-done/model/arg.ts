/**
 * The `checkup-mark-done` sheet argument: `"<typeId>"` to record a new visit,
 * `"<typeId>-<recordId>"` to edit an existing record. Ids only — never health
 * data in the URL (§11).
 */
export interface MarkDoneTarget {
  typeId: number;
  recordId: number | null;
}

const ARG = /^(\d{1,12})(?:-(\d{1,12}))?$/;

export function parseMarkDoneArg(arg: string | undefined): MarkDoneTarget | null {
  const m = arg ? ARG.exec(arg) : null;
  if (!m) return null;
  const typeId = Number(m[1]);
  const recordId = m[2] === undefined ? null : Number(m[2]);
  if (typeId <= 0 || (recordId !== null && recordId <= 0)) return null;
  return { typeId, recordId };
}

export function markDoneArg(typeId: number, recordId?: number | null): string {
  return recordId ? `${typeId}-${recordId}` : String(typeId);
}
