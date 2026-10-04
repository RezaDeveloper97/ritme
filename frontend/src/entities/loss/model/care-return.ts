import type { LossEvent } from './types';

/*
 * The calm way back into «مراقبت از خودت» (`/loss/care`, CB-LOSS-03b) after she has left it.
 *
 * Default window: LOSS_CARE_WINDOW_DAYS (60) days from the day she recorded the loss (Tehran day of `createdAt`,
 * else the approximate `occurredOn`). Inside it a quiet row shows on her cycle / TTC home (dismissible, per loss)
 * and in /profile/mode (not dismissible — the way back after hiding the home row). It ends earlier when she erases
 * the record (`DELETE /loss` → `GET /loss` has no loss). Never shown to companions: they never load her home or
 * profile, and `GET /loss` answers only for her.
 */
export const LOSS_CARE_WINDOW_DAYS = 60;

const DAY_MS = 86_400_000;

function dayNumber(isoDay: string): number | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(isoDay);
  return m ? Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3])) / DAY_MS : null;
}

/** Whether the re-entry to /loss/care is still offered on `todayIso` (`Y-m-d`, Tehran). */
export function isLossCareOpen(loss: LossEvent | null | undefined, todayIso: string): boolean {
  if (!loss) return false;
  // `createdAt` carries the Tehran offset, so its first 10 characters are her local day.
  const start = dayNumber(loss.createdAt ?? loss.occurredOn ?? '');
  const now = dayNumber(todayIso);
  if (start === null || now === null) return false;
  const age = now - start;
  return age >= 0 && age < LOSS_CARE_WINDOW_DAYS;
}

/*
 * «پنهان کن» on the home row: a per-device convenience that keeps only the loss row's numeric id (never its type,
 * date or anything she wrote — §11) under a neutral key. Every access is guarded; blocked storage just shows the
 * row again.
 */
const HIDDEN_KEY = 'ritme_care_return_hidden';

export function isLossCareRowHidden(lossId: number): boolean {
  try {
    return window.localStorage.getItem(HIDDEN_KEY) === String(lossId);
  } catch {
    return false;
  }
}

export function hideLossCareRow(lossId: number): void {
  try {
    window.localStorage.setItem(HIDDEN_KEY, String(lossId));
  } catch {
    // storage unavailable — the row comes back on the next visit
  }
}
