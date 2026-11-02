/** OKR status copy helpers (T-M10-34, SPEC v6 §5.7, A-07). Mirrors the backend rule; the API stays the source of truth. */

import type { KrStatus } from '../api/okr';

/** On track while progress >= elapsed% - PACE_SLACK_POINTS; done at 100 %; otherwise at risk. */
export const PACE_SLACK_POINTS = 10;

const DAY = 86_400_000;

/** Share (0..100) of the period gone by `now`; a period that has not started is 0, an ended one 100. */
export function elapsedPct(startsOn: string, endsOn: string, now: Date): number | null {
  const start = Date.parse(`${startsOn}T00:00:00Z`);
  const end = Date.parse(`${endsOn}T00:00:00Z`);
  if (!Number.isFinite(start) || !Number.isFinite(end) || end < start) return null;
  const today = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate());
  if (today < start) return 0;
  const total = (end - start) / DAY + 1;
  return Math.round(Math.min(1, ((today - start) / DAY + 1) / total) * 100);
}

/** The lowest progress (percent) that still counts as on track at `elapsed`; 0 when the period has barely begun. */
export function paceFloor(elapsed: number): number {
  return Math.max(0, Math.round(elapsed - PACE_SLACK_POINTS));
}

export interface StatusHint {
  /** i18n key suffix under okr.statusHint. */
  key: KrStatus | 'behind';
  elapsed: number | null;
  floor: number | null;
}

/** Why a key result has its status, for the line under it. */
export function statusHint(status: KrStatus, progress: number | null, elapsed: number | null): StatusHint {
  if (status === 'at_risk' && progress != null && elapsed != null) return { key: 'behind', elapsed, floor: paceFloor(elapsed) };
  return { key: status, elapsed, floor: elapsed == null ? null : paceFloor(elapsed) };
}
