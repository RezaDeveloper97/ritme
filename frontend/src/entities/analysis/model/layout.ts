import type { HubRecentCycle } from './types';

/** What one day of a recent-cycle strip shows (An_Hub «پریود و تاریخچه سیکل‌ها»). */
export type StripDay = 'period' | 'fertile' | 'ovulation' | 'other';

export interface StripCell {
  day: number;
  kind: StripDay;
  /** A day of the current cycle that hasn't happened yet (drawn faint). */
  future: boolean;
}

/** The day dots of one cycle, day 1 first. */
export function stripCells(cycle: HubRecentCycle): StripCell[] {
  const cells: StripCell[] = [];
  for (let day = 1; day <= cycle.length; day++) {
    let kind: StripDay = 'other';
    if (day <= cycle.periodDays) kind = 'period';
    else if (day === cycle.ovulationDay) kind = 'ovulation';
    else if (day >= cycle.fertileStartDay && day <= cycle.fertileEndDay) kind = 'fertile';
    const future = cycle.isCurrent && cycle.daysSoFar !== null && day > cycle.daysSoFar;
    cells.push({ day, kind, future });
  }
  return cells;
}

/** Bar heights of the cycle card as a share (0–1) of the tallest, with a floor so a short cycle stays visible. */
export function barShares(lengths: readonly number[], floor = 0.35): number[] {
  const max = Math.max(0, ...lengths);
  const min = Math.min(...lengths);
  if (!lengths.length || max <= 0) return lengths.map(() => 1);
  if (max === min) return lengths.map(() => 1);
  // Scale the spread, not the absolute value: 28 vs 31 days must read as different bars.
  return lengths.map((l) => floor + (1 - floor) * ((l - min) / (max - min)));
}

/** Share (0–100) of each count against the largest, for the horizontal symptom bars. */
export function shareOfMax(values: readonly number[]): number[] {
  const max = Math.max(0, ...values);
  return values.map((v) => (max > 0 ? Math.round((v / max) * 100) : 0));
}

/** Index of the lowest non-null value (the phase the mood card marks), or -1. */
export function lowestIndex(values: ReadonlyArray<number | null>): number {
  let at = -1;
  values.forEach((v, i) => {
    if (v !== null && (at === -1 || v < (values[at] as number))) at = i;
  });
  return at;
}
