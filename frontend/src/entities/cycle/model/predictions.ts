import type { CycleCalculation, CyclePhase, CyclePredictions } from './types';

/** Days before ovulation the fertile window is considered to open. */
const FERTILE_WINDOW_LEAD_DAYS = 4;

/** Length of the PMS window (the run of days ending the day before next period). */
const PMS_WINDOW_DAYS = 4;

const KNOWN_PHASES: readonly CyclePhase[] = [
  'period',
  'follicular',
  'fertile',
  'ovulation',
  'luteal',
];

/**
 * Map the API's free-form phase string onto our `CyclePhase` union. Unknown
 * values fall back to `luteal` (the safe "between events" phase) rather than
 * throwing — a new backend label should never blank the home screen.
 */
export function normalizePhase(phase: string): CyclePhase {
  return (KNOWN_PHASES as readonly string[]).includes(phase)
    ? (phase as CyclePhase)
    : 'luteal';
}

/**
 * Turn today's raw calculation into the day-offsets and labels the home screen
 * renders. Pure and locale-free (CLAUDE.md §7) — the UI formats the offsets as
 * Jalali dates. Offsets are clamped to sane bounds so a slightly stale
 * calculation can't produce nonsense like a negative "days until next period".
 */
export function deriveCyclePredictions(calc: CycleCalculation): CyclePredictions {
  const daysUntilNextPeriod = Math.max(0, calc.cycleLength - calc.cycleDay);
  const daysUntilOvulation = calc.estimatedOvulationDay - calc.cycleDay;
  // PMS is the short run of days ending the day before the next period. Clamp to
  // today so an imminent/overdue period never yields a negative window.
  const daysUntilPmsEnd = Math.max(0, daysUntilNextPeriod - 1);
  const daysUntilPmsStart = Math.max(0, daysUntilNextPeriod - PMS_WINDOW_DAYS);

  return {
    cycleDay: calc.cycleDay,
    phase: normalizePhase(calc.phase),
    cycleLength: calc.cycleLength,
    fertilityPercent: Math.round(Math.min(100, Math.max(0, calc.fertilityPercent))),
    daysUntilNextPeriod,
    daysUntilOvulation,
    daysUntilFertileWindow: daysUntilOvulation - FERTILE_WINDOW_LEAD_DAYS,
    daysUntilPmsStart,
    daysUntilPmsEnd,
    isPeriodTomorrow: calc.isPeriodTomorrow,
    isFertileWindow: calc.isFertileWindow,
  };
}
