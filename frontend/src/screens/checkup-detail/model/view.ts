import { diffInDays, fromApiDate, today } from '@/shared/lib/date';

/** Whole months since `lastDoneOn` (≈30-day months), null when never done. */
export function monthsSince(lastDoneOn: string | null, now: Date = today()): number | null {
  if (!lastDoneOn) return null;
  const days = diffInDays(now, fromApiDate(lastDoneOn));
  return days < 0 ? 0 : Math.floor(days / 30);
}

/**
 * The `checkup-mark-done` sheet arg (`"<typeId>"` / `"<typeId>-<recordId>"`).
 * Kept in step with `screens/checkup-mark-done` — sibling screens can't import
 * each other (FSD §3.3).
 */
export function markDoneSheetArg(typeId: number, recordId?: number): string {
  return recordId ? `${typeId}-${recordId}` : String(typeId);
}

export const MARK_DONE_SHEET = 'checkup-mark-done';
