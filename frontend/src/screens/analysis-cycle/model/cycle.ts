import type { CycleReport, ReportCycle } from '@/entities/analysis';

/** Day layout of one cycle (backend-go/internal/analysis NewLayout: luteal 14 days, fertile = ovulation − 5 … ovulation). */
export interface DayLayout {
  periodDays: number;
  ovulationDay: number;
  fertileStart: number;
  fertileEnd: number;
}

const LUTEAL_DAYS = 14;
const FERTILE_BEFORE_OVULATION = 5;

export function dayLayout(length: number, periodDays: number): DayLayout {
  const period = Math.min(Math.max(1, periodDays), length);
  const ovulation = Math.min(length, Math.max(period + 1, length - LUTEAL_DAYS + 1));
  const fertileStart = Math.min(ovulation, Math.max(period + 1, ovulation - FERTILE_BEFORE_OVULATION));
  return { periodDays: period, ovulationDay: ovulation, fertileStart, fertileEnd: ovulation };
}

export type Verdict = 'good' | 'warn' | 'none';

/** Tone of the cycle-length tile: FIGO normal, out of range, or no verdict (no median / FIGO not applied). */
export function cycleVerdict(r: CycleReport): Verdict {
  if (r.cycleLength.median == null || !r.figo.applies || !r.cycleLength.status) return 'none';
  return r.cycleLength.status === 'normal' ? 'good' : 'warn';
}

export function periodVerdict(r: CycleReport): Verdict {
  if (r.periodLength.median == null || !r.periodLength.status) return 'none';
  if (!r.figo.applies) return 'none';
  return r.periodLength.status === 'normal' ? 'good' : 'warn';
}

export function variationVerdict(r: CycleReport): Verdict {
  if (r.variation.status === 'not_enough_data') return 'none';
  return r.variation.status === 'regular' ? 'good' : 'warn';
}

/** Cycles oldest → newest for the bar chart (the API lists newest first). */
export function chronological(cycles: readonly ReportCycle[]): ReportCycle[] {
  return [...cycles].reverse();
}

/** The excluded cycles that need a note under the bars (newest first, as the API lists them). */
export function excludedCycles(cycles: readonly ReportCycle[]): ReportCycle[] {
  return cycles.filter((c) => !c.counted && c.excluded !== null);
}

/** Cycles still needed before the analysis is ready (the engine wants 2 for medians). */
export const MIN_CYCLES = 2;
