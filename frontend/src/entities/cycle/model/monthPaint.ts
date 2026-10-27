import { addDays, diffInDays, fromApiDate, toApiDate } from '@/shared/lib/date';

import { PMS_WINDOW_DAYS } from './predictions';
import type {
  CycleAnchors,
  CycleDayMarker,
  CycleFutureCycle,
  CycleMonthDay,
  MonthPeriodKind,
} from './types';

/**
 * Future cycles the engine predicts (E-05); nothing is painted beyond the 6th.
 * The server already stops there — this is the client's own guard.
 */
export const MAX_FUTURE_CYCLES = 6;

/** How one calendar day is painted from v6 month data. */
export interface MonthDayPaint {
  /** The coloured marker, or null for a neutral day. */
  marker: CycleDayMarker | null;
  /** `logged` = solid period; `assumed`/`predicted` = soft period; null = not a period day. */
  periodKind: MonthPeriodKind | null;
  /** A day past the predicted start with no bleeding logged (`period_expected`, days_late > 0). */
  late: boolean;
}

/** `period_expected` and past the predicted start: the period is late (E-04). */
export function isLateDay(day: CycleMonthDay | undefined): boolean {
  return !!day && day.mainPhase === 'period_expected' && (day.daysLate ?? 0) > 0;
}

/**
 * The day map of the month payload's `future_cycles` (k ≤ 6): predicted period,
 * the PMS run before each period, the fertile window and ovulation. Replaces the
 * client's own cycle roll-forward — the server decides how many cycles exist,
 * and none at all while the period is more than 14 days late.
 * Fertile days and ovulation are left out when `suppressed`.
 */
export function futureCycleDayMap(
  futureCycles: readonly CycleFutureCycle[],
  suppressed = false,
): Map<string, MonthDayPaint> {
  const out = new Map<string, MonthDayPaint>();
  const set = (iso: string, paint: MonthDayPaint, overwrite = false) => {
    if (overwrite || !out.has(iso)) out.set(iso, paint);
  };
  const range = (from: string, to: string): string[] => {
    const first = fromApiDate(from);
    const count = diffInDays(fromApiDate(to), first) + 1;
    return Array.from({ length: Math.max(0, count) }, (_, i) => toApiDate(addDays(first, i)));
  };

  for (const c of futureCycles) {
    if (c.k < 1 || c.k > MAX_FUTURE_CYCLES) continue;
    if (c.periodStart) {
      const end = c.periodEnd ?? c.periodStart;
      for (const iso of range(c.periodStart, end)) {
        set(iso, { marker: 'period', periodKind: 'predicted', late: false }, true);
      }
      const start = fromApiDate(c.periodStart);
      for (let i = 1; i <= PMS_WINDOW_DAYS; i += 1) {
        set(toApiDate(addDays(start, -i)), { marker: 'pms', periodKind: null, late: false });
      }
    }
    if (!suppressed && c.fertileStart && c.fertileEnd) {
      for (const iso of range(c.fertileStart, c.fertileEnd)) {
        set(iso, { marker: 'fertile', periodKind: null, late: false }, true);
      }
    }
    if (!suppressed && c.ovulation) {
      set(c.ovulation, { marker: 'ovulation', periodKind: null, late: false }, true);
    }
  }
  return out;
}

/**
 * How a v6 month day is painted. The server's per-day fields decide:
 * `period_kind` (logged / assumed / predicted), a late `period_expected` day,
 * the `fertile` main phase (ovulation on the day the cycle day meets the
 * estimated ovulation day) and PMS in the {@link PMS_WINDOW_DAYS} days before a
 * period. A day the payload does not carry falls back to the `future_cycles`
 * day map. Returns null for a neutral day.
 */
export function monthDayPaint(
  day: CycleMonthDay | undefined,
  iso: string,
  futureMap: ReadonlyMap<string, MonthDayPaint>,
  suppressed = false,
): MonthDayPaint | null {
  if (!day) return futureMap.get(iso) ?? null;

  if (day.periodKind) return { marker: 'period', periodKind: day.periodKind, late: false };
  if (isLateDay(day)) return { marker: null, periodKind: null, late: true };
  // The predicted start day itself: expected today, not late yet.
  if (day.mainPhase === 'period_expected') {
    return { marker: 'period', periodKind: 'predicted', late: false };
  }
  if (day.mainPhase === 'fertile' && !suppressed) {
    const ovulation = day.cycleDay != null && day.cycleDay === day.estimatedOvulationDay;
    return { marker: ovulation ? 'ovulation' : 'fertile', periodKind: null, late: false };
  }
  const toPeriod = day.daysToPeriod;
  if (toPeriod != null && toPeriod >= 1 && toPeriod <= PMS_WINDOW_DAYS && day.mainPhase !== 'unknown') {
    return { marker: 'pms', periodKind: null, late: false };
  }
  return null;
}

/**
 * The open period's end for the calendar (E-04): the engine's
 * `anchors.current_period_end`, not the profile length. `confirmed` says whether
 * the user logged that end or the engine assumed it. Null when the anchors are
 * for another period (`startIso`), or carry no end.
 */
export function openPeriodEnd(
  anchors: CycleAnchors | null | undefined,
  startIso: string,
): { end: string; confirmed: boolean } | null {
  if (!anchors?.currentPeriodStart || anchors.currentPeriodStart !== startIso) return null;
  if (!anchors.currentPeriodEnd) return null;
  return { end: anchors.currentPeriodEnd, confirmed: anchors.currentPeriodEndIsConfirmed };
}
