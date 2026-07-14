import type { CycleCalculation, CycleDayMarker, CyclePhase, CyclePredictions } from './types';

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
 * Backend phase values that don't match our UI union verbatim. The engine's
 * `CyclePhase` enum calls the bleeding phase `menstruation`; the UI calls it
 * `period`. Map those synonyms so the phase renders correctly everywhere.
 */
const PHASE_ALIASES: Record<string, CyclePhase> = {
  menstruation: 'period',
};

/**
 * Map the API's free-form phase string onto our `CyclePhase` union. Unknown
 * values fall back to `luteal` (the safe "between events" phase) rather than
 * throwing — a new backend label should never blank the home screen.
 */
export function normalizePhase(phase: string): CyclePhase {
  if ((KNOWN_PHASES as readonly string[]).includes(phase)) {
    return phase as CyclePhase;
  }
  return PHASE_ALIASES[phase] ?? 'luteal';
}

/**
 * Phase for calendar/day coloring. Like {@link normalizePhase}, but surfaces the
 * fertile window as its own `fertile` color — the backend flags it separately
 * (`is_fertile_window`) during the follicular phase rather than as a phase of
 * its own. Period and ovulation always win over the fertile tint.
 */
export function calcToPhase(calc: CycleCalculation): CyclePhase {
  const phase = normalizePhase(calc.phase);
  if (phase === 'period' || phase === 'ovulation') return phase;
  return calc.isFertileWindow ? 'fertile' : phase;
}

/**
 * The colored marker for a calendar day, or `null` for a neutral day. Priority:
 * period → ovulation → fertile window → PMS window. Ovulation and the fertile
 * window overlap (ovulation sits inside it), so ovulation wins; period always
 * wins. This is what paints the calendar cells and drives the legend.
 */
export function cycleDayMarker(calc: CycleCalculation): CycleDayMarker | null {
  const phase = normalizePhase(calc.phase);
  if (phase === 'period') return 'period';
  if (phase === 'ovulation') return 'ovulation';
  if (calc.isFertileWindow) return 'fertile';
  if (calc.isPmsWindow) return 'pms';
  return null;
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
