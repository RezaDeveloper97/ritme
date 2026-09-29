import { addDays, diffInDays, fromApiDate } from '@/shared/lib/date';

import {
  calcMainPhase,
  cycleDayMarker,
  FERTILE_WINDOW_LEAD_DAYS,
  PMS_WINDOW_DAYS,
} from './predictions';
import type { CycleCalculation, CycleDayMarker, CycleView } from './types';

/** Fixed ovulation → next-period offset the engine uses (`CyclePredictionService`). */
const LUTEAL_LENGTH = 14;

const DEFAULT_CYCLE_LENGTH = 28;

/** Bleeding length when the engine gave none (the backend's default). */
const DEFAULT_PERIOD_LENGTH = 5;

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
  /**
   * Effective bleeding length — what a *predicted* cycle's period lasts, so a
   * rolled-forward cycle opens its window after its own period (§19), not
   * after the current one's logged end.
   */
  periodLength: number;
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

  const periodLength = Math.max(
    1,
    view?.metrics?.effectivePeriodLength ??
      view?.effectiveValues.periodDuration ??
      DEFAULT_PERIOD_LENGTH,
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
  const periodEnd = anchors?.currentPeriodStart ? parse(anchors.currentPeriodEnd) : null;
  const fertileStart = displayFertileStart(ovulation, periodEnd);

  return {
    cycleStart,
    cycleLength,
    periodLength,
    nextPeriodStart,
    ovulation,
    fertileStart,
    fertileEnd: ovulation,
    pmsEnd: addDays(nextPeriodStart, -1),
    pmsStart: addDays(nextPeriodStart, -PMS_WINDOW_DAYS),
  };
}

/**
 * task.md §19 display window start: `max(ovulation − 5, period end + 1)`. With
 * no known period end it is the biological start (ovulation − 5).
 */
function displayFertileStart(ovulation: Date, periodEnd: Date | null): Date {
  const biologicalStart = addDays(ovulation, -FERTILE_WINDOW_LEAD_DAYS);
  if (!periodEnd) return biologicalStart;
  const dayAfterPeriod = addDays(periodEnd, 1);
  return diffInDays(dayAfterPeriod, biologicalStart) > 0 ? dayAfterPeriod : biologicalStart;
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
  const cycleStart = addDays(schedule.cycleStart, shift);
  const ovulation = addDays(schedule.ovulation, shift);
  return {
    ...schedule,
    cycleStart,
    nextPeriodStart: addDays(schedule.nextPeriodStart, shift),
    ovulation,
    // Another cycle's period is a predicted one of the effective length (the
    // engine's roll-forward), so its window opens after *that* period.
    fertileStart: displayFertileStart(ovulation, addDays(cycleStart, schedule.periodLength - 1)),
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

/**
 * The §19 window of the schedule's own cycle as 1-based cycle days (for linear
 * day-1 → day-N charts), or `null` when a long period swallowed it.
 */
export function fertileWindowDays(
  schedule: CycleSchedule,
): { startDay: number; endDay: number; ovulationDay: number } | null {
  if (!hasFertileWindow(schedule)) return null;
  const day = (d: Date) => diffInDays(d, schedule.cycleStart) + 1;
  return {
    startDay: day(schedule.fertileStart),
    endDay: day(schedule.fertileEnd),
    ovulationDay: day(schedule.ovulation),
  };
}

/**
 * THE calendar marker for a day — every surface that paints cycle days (the
 * calendar grid and day sheet, the home week strip and ring) asks this, so the
 * fertile window is the same days everywhere (task.md §19, T-M5-13):
 *
 * - From the current cycle on (the schedule's cycle and the predicted ones
 *   after it) the **window and ovulation come from the anchored schedule** —
 *   the very dates the home timeline, `/fertility/bbt` and `/fertility/insights`
 *   show. Period and PMS still come from the engine's per-day calculation
 *   (logged periods), and a period day always wins.
 * - Before the current cycle (history) the per-day calculation is read by the
 *   same §19 rule ({@link cycleDayMarker}).
 * - A day with no calculation falls back to the schedule alone.
 */
export function cycleDayMarkerAt(
  date: Date,
  calc: CycleCalculation | null | undefined,
  schedule: CycleSchedule | null,
  periodLength: number,
): CycleDayMarker | null {
  if (!calc) return schedule ? scheduleDayMarker(schedule, date, periodLength) : null;
  if (!schedule || diffInDays(date, schedule.cycleStart) < 0) return cycleDayMarker(calc);

  if (calcMainPhase(calc) === 'period') return 'period';
  const s = cycleScheduleFor(schedule, date);
  if (diffInDays(date, s.ovulation) === 0) return 'ovulation';
  if (diffInDays(date, s.fertileStart) >= 0 && diffInDays(s.fertileEnd, date) >= 0) return 'fertile';
  return calc.isPmsWindow ? 'pms' : null;
}
