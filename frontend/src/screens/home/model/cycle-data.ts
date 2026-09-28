import { type CycleCalculation, type CycleView, deriveCycleSchedule } from '@/entities/cycle';

/**
 * Has the engine nothing to place today in a cycle? True when no period start
 * is on record (a new cycle user, or one who just left pregnancy mode without
 * cycle history): the anchors are empty and there is no cycle day to count
 * back from. The home then asks for the last period instead of showing a ring
 * and a phase card full of «—». `/messages/daily` answers 400 in the same
 * state (see `isNoDailyMessage` in `entities/message`).
 */
export function needsPeriodData(view: CycleView | null, calc: CycleCalculation | null): boolean {
  return deriveCycleSchedule(view, calc) === null;
}
