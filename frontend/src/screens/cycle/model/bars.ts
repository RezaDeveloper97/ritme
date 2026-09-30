import type { CycleHistoryRow } from './schema';

/** The shortest scale the bars are drawn against, so one short history doesn't fill the track. */
export const MIN_BAR_SCALE = 35;

/**
 * Days every bar is scaled to: the longest cycle on screen (or the predicted
 * length of the current one), never below {@link MIN_BAR_SCALE}.
 */
export function barScale(rows: readonly CycleHistoryRow[], predicted: number): number {
  let longest = predicted;
  for (const r of rows) longest = Math.max(longest, r.length);
  return Math.max(MIN_BAR_SCALE, longest);
}

export interface BarGeometry {
  /** Share of the track the bleed days take, 0–100. */
  period: number;
  /** Share the whole cycle takes (days so far for the current one), 0–100. */
  cycle: number;
}

const pct = (days: number, scale: number) => Math.min(100, Math.max(0, (days / scale) * 100));

/** Widths of one history bar, as percentages of the track. */
export function barGeometry(row: CycleHistoryRow, scale: number): BarGeometry {
  const cycle = pct(row.length, scale);
  return { period: Math.min(cycle, pct(row.periodDays, scale)), cycle };
}

/** Heat level of one strip value: 0 (nothing) … 3 (most cycles, strongest). */
export function heatLevel(value: number): 0 | 1 | 2 | 3 {
  if (value <= 0.05) return 0;
  if (value < 0.34) return 1;
  if (value < 0.67) return 2;
  return 3;
}

export interface HeatRun {
  level: 1 | 2 | 3;
  /** 0-based first day of the run. */
  start: number;
  /** Number of days. */
  length: number;
}

/**
 * Consecutive days of the same non-zero level, so the strip paints a few
 * rounded pills instead of 28 hairline cells.
 */
export function heatRuns(strip: readonly number[]): HeatRun[] {
  const runs: HeatRun[] = [];
  strip.forEach((v, i) => {
    const level = heatLevel(v);
    if (level === 0) return;
    const last = runs[runs.length - 1];
    if (last && last.level === level && last.start + last.length === i) last.length += 1;
    else runs.push({ level, start: i, length: 1 });
  });
  return runs;
}

/** The typical-cycle bands drawn above the strips (1-based, inclusive days). */
export interface TypicalBands {
  period: [number, number];
  fertile: [number, number];
  ovulation: number;
  pms: [number, number];
}

/** Fertile window = the 5 days before ovulation through the day after; PMS = the last 5 days. */
export function typicalBands(cycleLength: number, periodLength: number, ovulationDay: number): TypicalBands {
  const clamp = (d: number) => Math.min(cycleLength, Math.max(1, d));
  return {
    period: [1, clamp(periodLength)],
    fertile: [clamp(ovulationDay - 5), clamp(ovulationDay + 1)],
    ovulation: clamp(ovulationDay),
    pms: [clamp(cycleLength - 4), cycleLength],
  };
}
