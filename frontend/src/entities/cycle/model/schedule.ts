import { addDays, diffInDays, fromApiDate } from '@/shared/lib/date';

import {
  FERTILE_WINDOW_LEAD_DAYS,
  FERTILE_WINDOW_TRAIL_DAYS,
  PMS_WINDOW_DAYS,
} from './predictions';
import type { CycleCalculation, CycleView } from './types';

/** Fixed ovulation → next-period offset the engine uses (`CyclePredictionService`). */
const LUTEAL_LENGTH = 14;

const DEFAULT_CYCLE_LENGTH = 28;

/**
 * The calendar of one cycle, as **absolute dates**.
 *
 * Every event here is anchored to the period *start* the engine resolved — the
 * same rule the backend predicts by (`CyclePredictionService`: next start =
 * current start + effective length, ovulation = next start − luteal length).
 * Deriving screens from these dates instead of from "today + N days" offsets is
 * what keeps the home screen's countdown, the row dates and the calendar from
 * drifting apart when the server's calculation is a day behind the device's day
 * (timezones) or when the user scrubs to another day.
 */
export interface CycleSchedule {
  /** Day 1 of the cycle these dates belong to. */
  cycleStart: Date;
  /** Effective cycle length in days (start → next start). */
  cycleLength: number;
  nextPeriodStart: Date;
  ovulation: Date;
  fertileStart: Date;
  fertileEnd: Date;
  /** First PMS day (the run of days ending the day before the next period). */
  pmsStart: Date;
  pmsEnd: Date;
}

const parse = (value: string | null | undefined): Date | null =>
  value ? fromApiDate(value) : null;

/**
 * Build the cycle calendar from the engine's own anchors/forecast, falling back
 * to the legacy per-day `calculation` (cycle day + length) when a backend
 * without `cycle_view` answers. Returns `null` when neither can place the cycle.
 */
export function deriveCycleSchedule(
  view: CycleView | null,
  calc: CycleCalculation | null,
): CycleSchedule | null {
  const anchors = view?.anchors ?? null;
  const forecast = view?.forecast ?? null;

  // Day 1 of the cycle: the resolved period start, else back-count from the
  // calculation's own date (never from the device's today — that's the drift).
  const cycleStart =
    parse(anchors?.currentPeriodStart) ??
    (calc ? addDays(fromApiDate(calc.calculationDate), -(calc.cycleDay - 1)) : null);
  if (!cycleStart) return null;

  const cycleLength = Math.max(
    1,
    view?.metrics?.effectiveCycleLength ??
      view?.effectiveValues.cycleLength ??
      calc?.cycleLength ??
      DEFAULT_CYCLE_LENGTH,
  );

  const nextPeriodStart =
    parse(forecast?.nextPeriodStart) ??
    parse(anchors?.predictedNextPeriodStart) ??
    addDays(cycleStart, cycleLength);

  const ovulation =
    parse(forecast?.estimatedOvulationDate) ??
    parse(anchors?.estimatedOvulationDate) ??
    (calc ? addDays(cycleStart, calc.estimatedOvulationDay - 1) : null) ??
    addDays(nextPeriodStart, -LUTEAL_LENGTH);

  return {
    cycleStart,
    cycleLength,
    nextPeriodStart,
    ovulation,
    fertileStart:
      parse(forecast?.fertileWindowStart) ?? addDays(ovulation, -FERTILE_WINDOW_LEAD_DAYS),
    fertileEnd:
      parse(forecast?.fertileWindowEnd) ?? addDays(ovulation, FERTILE_WINDOW_TRAIL_DAYS),
    pmsEnd: addDays(nextPeriodStart, -1),
    pmsStart: addDays(nextPeriodStart, -PMS_WINDOW_DAYS),
  };
}

/**
 * The same schedule projected onto the cycle that contains `date`, by shifting
 * it whole cycles forward or back. Lets the UI scrub to any day without
 * refetching and still show that day's own cycle — the countdown then falls
 * smoothly to a fixed date instead of jumping when a cycle boundary is crossed.
 */
export function cycleScheduleFor(schedule: CycleSchedule, date: Date): CycleSchedule {
  const cycles = Math.floor(diffInDays(date, schedule.cycleStart) / schedule.cycleLength);
  if (cycles === 0) return schedule;

  const shift = cycles * schedule.cycleLength;
  return {
    ...schedule,
    cycleStart: addDays(schedule.cycleStart, shift),
    nextPeriodStart: addDays(schedule.nextPeriodStart, shift),
    ovulation: addDays(schedule.ovulation, shift),
    fertileStart: addDays(schedule.fertileStart, shift),
    fertileEnd: addDays(schedule.fertileEnd, shift),
    pmsStart: addDays(schedule.pmsStart, shift),
    pmsEnd: addDays(schedule.pmsEnd, shift),
  };
}

/** Whole days from `date` to that cycle's next period start (never negative). */
export function daysUntilNextPeriod(schedule: CycleSchedule, date: Date): number {
  return Math.max(0, diffInDays(cycleScheduleFor(schedule, date).nextPeriodStart, date));
}

/** How far through its cycle `date` sits, 0–100 — for the hero progress bar. */
export function cycleProgressPercent(schedule: CycleSchedule, date: Date): number {
  const rolled = cycleScheduleFor(schedule, date);
  const elapsed = diffInDays(date, rolled.cycleStart);
  return Math.min(100, Math.max(0, Math.round((elapsed / rolled.cycleLength) * 100)));
}
