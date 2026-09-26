import type { CheckupRecord } from '@/entities/checkup';

/** `?type=` → a checkup type id, or null for «همه». */
export function parseTypeParam(value: string | undefined): number | null {
  if (!value || !/^\d{1,12}$/.test(value)) return null;
  const id = Number(value);
  return id > 0 ? id : null;
}

/**
 * «با پیوست» shows records whose report is actually on THIS device — the
 * server flag alone may point at a file stored on another phone.
 */
export function withLocalAttachment(
  records: readonly CheckupRecord[],
  localIds: ReadonlySet<number> | undefined,
): CheckupRecord[] {
  return records.filter((r) => localIds?.has(r.id) ?? r.hasAttachment);
}

/**
 * The `checkup-mark-done` sheet arg (`"<typeId>-<recordId>"` = edit). Kept in
 * step with `screens/checkup-mark-done` — sibling screens can't import each
 * other (FSD §3.3).
 */
export function editRecordSheetArg(record: Pick<CheckupRecord, 'checkupTypeId' | 'id'>): string {
  return `${record.checkupTypeId}-${record.id}`;
}

export const MARK_DONE_SHEET = 'checkup-mark-done';
