import type { FlowLevel, PeriodReport } from '@/entities/analysis';

export type Level = 1 | 2 | 3 | 4 | null;

const LEVEL: Record<FlowLevel, Exclude<Level, null>> = { light: 1, medium: 2, heavy: 3, very_heavy: 4 };

export function levelOf(flow: FlowLevel | null): Level {
  return flow ? LEVEL[flow] : null;
}

/** Number of day columns of the flow grid: the longest period or the average, whichever is longer. */
export function flowColumns(r: PeriodReport): number {
  return Math.max(r.average.length, ...r.periods.map((p) => p.days.length), 0);
}

/** Whether any flow at all was logged (otherwise the grid would be empty). */
export function hasFlow(r: PeriodReport): boolean {
  return r.average.some((a) => a.level !== null) || r.periods.some((p) => p.days.some((d) => d.flow !== null));
}

export const LEVELS: readonly FlowLevel[] = ['light', 'medium', 'heavy', 'very_heavy'];
