import { addDays, diffInDays, fromApiDate } from '@/shared/lib/date';

import { FERTILE_WINDOW_LEAD_DAYS, PMS_WINDOW_DAYS } from './predictions';
import type { CycleCalculation, CycleDayMarker, CycleView } from './types';

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
  /**
   * The display fertile window (task.md §19): from `max(ovulation − 5, period
   * end + 1)` to the ovulation day itself — the days the engine resolves as
   * `main_phase = fertile`, and the window `/fertility/bbt` and
   * `/fertility/insights` draw. Empty (`fertileEnd` before `fertileStart`) when
   * a long period swallows it; see {@link hasFertileWindow}.
   */
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

  // §19 display window from the same anchors as the engine's phases. Not the
  // legacy `predictions.fertile_window_*` (the biological O−5 … O+1): that one
  // ran a day past ovulation and disagreed with the fertility screens.
  const biologicalStart = addDays(ovulation, -FERTILE_WINDOW_LEAD_DAYS);
  const periodEnd = anchors?.currentPeriodStart ? parse(anchors.currentPeriodEnd) : null;
  const dayAfterPeriod = periodEnd ? addDays(periodEnd, 1) : null;
  const fertileStart =
    dayAfterPeriod && diffInDays(dayAfterPeriod, biologicalStart) > 0 ? dayAfterPeriod : biologicalStart;

  return {
    cycleStart,
    cycleLength,
    nextPeriodStart,
    ovulation,
    fertileStart,
    fertileEnd: ovulation,
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

/** Whether the schedule has a non-empty display fertile window (task.md §19). */
export function hasFertileWindow(schedule: CycleSchedule): boolean {
  return diffInDays(schedule.fertileEnd, schedule.fertileStart) >= 0;
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

/**
 * The marker a day carries according to the schedule alone — period → ovulation
 * → fertile window → PMS, the same priority as {@link cycleDayMarker}. For
 * surfaces that paint a day before (or without) that day's own calculation;
 * wherever the engine's per-day calculation is at hand, it wins.
 */
export function scheduleDayMarker(
  schedule: CycleSchedule,
  date: Date,
  periodLength: number,
): CycleDayMarker | null {
  const s = cycleScheduleFor(schedule, date);
  const within = (from: Date, to: Date) =>
    diffInDays(date, from) >= 0 && diffInDays(to, date) >= 0;

  if (diffInDays(date, s.cycleStart) < Math.max(1, periodLength)) return 'period';
  if (diffInDays(date, s.ovulation) === 0) return 'ovulation';
  if (within(s.fertileStart, s.fertileEnd)) return 'fertile';
  if (within(s.pmsStart, s.pmsEnd)) return 'pms';
  return null;
}
