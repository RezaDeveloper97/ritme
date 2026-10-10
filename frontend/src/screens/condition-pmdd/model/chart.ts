import { PMDD_MAX, PMDD_MIN, type PmddAlert, type PmddCycle, type PmddWindow } from '@/entities/condition';
import { diffInDays, fromApiDate } from '@/shared/lib/date';

/** Whole calendar days b − a between two API `YYYY-MM-DD` dates. */
export function daysBetween(a: string, b: string): number {
  return diffInDays(fromApiDate(b), fromApiDate(a));
}

/** Her cycle day today in the current cycle (1-based), or null without a cycle / before its start. */
export function cycleDayOn(cycle: PmddCycle | undefined, date: string): number | null {
  if (!cycle) return null;
  const day = daysBetween(cycle.start, date) + 1;
  return day >= 1 ? day : null;
}

export interface ChartBand {
  /** 0 … 1 along the cycle (day 1 at 0). */
  x0: number;
  x1: number;
}

export interface ChartGeometry {
  length: number;
  /** x 0 … 1 (cycle day), y 0 … 1 (mean 1 → 0, mean 6 → 1). */
  points: { x: number; y: number }[];
  period: ChartBand | null;
  lateLuteal: ChartBand | null;
}

/**
 * The chart of one cycle (nbl_Cond_PMDD «این سیکل»): the daily mean of the six items over the cycle days, with the
 * period and the last-10-days bands. A band covers whole days, so day d spans [(d − 1) / L, d / L].
 */
export function chartGeometry(cycle: PmddCycle): ChartGeometry {
  const length = Math.max(1, daysBetween(cycle.start, cycle.end) + 1);
  const band = (w: PmddWindow | null): ChartBand | null => {
    if (!w) return null;
    const from = Math.max(0, daysBetween(cycle.start, w.from));
    const to = Math.min(length, daysBetween(cycle.start, w.to) + 1);
    return to > from ? { x0: from / length, x1: to / length } : null;
  };
  const points = [...cycle.days]
    .filter((d) => d.cycleDay <= length)
    .sort((a, b) => a.cycleDay - b.cycleDay)
    .map((d) => ({
      x: (d.cycleDay - 0.5) / length,
      y: (Math.min(PMDD_MAX, Math.max(PMDD_MIN, d.mean)) - PMDD_MIN) / (PMDD_MAX - PMDD_MIN),
    }));
  return { length, points, period: band(cycle.period), lateLuteal: band(cycle.lateLuteal) };
}

export interface PmddNotes {
  /** The pattern sentence (`pmdd_pattern_*` / `pmdd_not_enough_data`). */
  pattern: PmddAlert | null;
  /** «برای جمع‌بندی، ۲ سیکل ثبت کامل لازم است» until ready. */
  needsTwo: PmddAlert | null;
  /** Crisis hotlines (always sent). */
  crisis: PmddAlert | null;
}

/** Sorts the chart's `condition_alerts` into the slots of the screen. */
export function pmddNotes(alerts: PmddAlert[]): PmddNotes {
  return {
    pattern: alerts.find((a) => a.code.startsWith('pmdd_pattern_') || a.code === 'pmdd_not_enough_data') ?? null,
    needsTwo: alerts.find((a) => a.code === 'pmdd_needs_two_cycles') ?? null,
    crisis: alerts.find((a) => a.code === 'pmdd_crisis') ?? null,
  };
}
