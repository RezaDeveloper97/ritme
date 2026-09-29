import type { CheckupDetail } from '@/entities/checkup';
import { diffInDays, fromApiDate, today } from '@/shared/lib/date';

/** Whole months since `lastDoneOn` (≈30-day months), null when never done. */
export function monthsSince(lastDoneOn: string | null, now: Date = today()): number | null {
  if (!lastDoneOn) return null;
  const days = diffInDays(now, fromApiDate(lastDoneOn));
  return days < 0 ? 0 : Math.floor(days / 30);
}

export type HeroRelative =
  | { kind: 'never' }
  | { kind: 'done'; lastDoneOn: string; /** Months past the due date; null unless overdue. */ overdueMonths: number | null };

/**
 * The hero's second line (v14_CheckupDetail «حدود ۶ ماه گذشته · آخرین بار
 * فروردین ۱۴۰۱»): how far past the *due date* an overdue item is — never the
 * time since the last visit, which misstated it (audit C1).
 */
export function heroRelative(
  detail: Pick<CheckupDetail, 'status' | 'lastDoneOn' | 'nextDueOn'>,
  now: Date = today(),
): HeroRelative {
  if (!detail.lastDoneOn) return { kind: 'never' };
  const months = detail.status === 'overdue' ? monthsSince(detail.nextDueOn, now) : null;
  return { kind: 'done', lastDoneOn: detail.lastDoneOn, overdueMonths: months && months > 0 ? months : null };
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
