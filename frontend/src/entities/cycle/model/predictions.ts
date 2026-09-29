import type { CycleCalculation, CycleDayMarker, CyclePhase, CyclePredictions } from './types';

/**
 * Days before ovulation the fertile window opens (task.md §18/§19: the window
 * runs from `max(O − 5, period end + 1)` to O).
 */
export const FERTILE_WINDOW_LEAD_DAYS = 5;

/**
 * Length of the PMS window: the run of days ending the day before the next
 * period. task.md §25.2 — `pms_possible` is `days_until_period ∈ {1, 2, 3}`.
 * The one PMS length every surface draws (calendar, home timeline, insights
 * strips); the legacy per-day `is_pms_window` flag (7 days) is only read
 * through {@link calcInPmsWindow}.
 */
export const PMS_WINDOW_DAYS = 3;

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
  // v1.1 engine main-phase values (task.md §13).
  menstrual: 'period',
  period_expected: 'luteal',
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
 * The legacy per-day calculation (`cycle/month`, `cycle/today`) still draws the
 * *biological* window: `is_fertile_window` on O−5 … O+1 and phase `ovulation`
 * on both O and O+1. The display window every screen shows is task.md §19 —
 * `max(O−5, period end + 1)` … O — so the day after ovulation is luteal. Since
 * the legacy engine already clears the flags on bleeding days, dropping O+1 is
 * all it takes to read a calculation by the §19 rule.
 */
function isAfterOvulation(calc: CycleCalculation): boolean {
  return calc.cycleDay > calc.estimatedOvulationDay;
}

/**
 * {@link normalizePhase} for a calculation, with the legacy O+1 `ovulation`
 * day read as luteal (§19: the display window ends on ovulation). Never
 * `fertile` — see {@link calcToPhase} for the window-aware colour phase.
 */
export function calcMainPhase(calc: CycleCalculation): CyclePhase {
  const phase = normalizePhase(calc.phase);
  return phase === 'ovulation' && isAfterOvulation(calc) ? 'luteal' : phase;
}

/**
 * Whether a calculation's day is inside the §19 display fertile window
 * (ovulation day included). The one reading of the legacy flag every screen
 * uses — never `calc.isFertileWindow` directly.
 */
export function calcInFertileWindow(calc: CycleCalculation): boolean {
  const phase = calcMainPhase(calc);
  if (phase === 'period' || isAfterOvulation(calc)) return false;
  return phase === 'ovulation' || calc.isFertileWindow;
}

/**
 * Whether a calculation's day is a §25.2 PMS day: 1…{@link PMS_WINDOW_DAYS}
 * days before the next period. The legacy `is_pms_window` flag runs 7 days (and
 * on past the expected start), so it is narrowed here — never read
 * `calc.isPmsWindow` directly.
 */
export function calcInPmsWindow(calc: CycleCalculation): boolean {
  if (!calc.isPmsWindow) return false;
  const daysUntilNextPeriod = calc.cycleLength - calc.cycleDay + 1;
  return daysUntilNextPeriod >= 1 && daysUntilNextPeriod <= PMS_WINDOW_DAYS;
}

/**
 * Phase for calendar/day coloring. Like {@link normalizePhase}, but surfaces the
 * fertile window as its own `fertile` color — the backend flags it separately
 * (`is_fertile_window`) during the follicular phase rather than as a phase of
 * its own. Period and ovulation always win over the fertile tint; the window is
 * the §19 one ({@link calcInFertileWindow}).
 */
export function calcToPhase(calc: CycleCalculation): CyclePhase {
  const phase = calcMainPhase(calc);
  if (phase === 'period' || phase === 'ovulation') return phase;
  return calcInFertileWindow(calc) ? 'fertile' : phase;
}

/**
 * The colored marker for a calendar day, or `null` for a neutral day. Priority:
 * period → ovulation → fertile window → PMS window. Ovulation and the fertile
 * window overlap (ovulation sits inside it), so ovulation wins; period always
 * wins. The window is the §19 one, so the day after ovulation carries no
 * fertile/ovulation marker.
 */
export function cycleDayMarker(calc: CycleCalculation): CycleDayMarker | null {
  const phase = calcMainPhase(calc);
  if (phase === 'period') return 'period';
  if (phase === 'ovulation') return 'ovulation';
  if (calcInFertileWindow(calc)) return 'fertile';
  if (calcInPmsWindow(calc)) return 'pms';
  return null;
}

/**
 * The phase a day reads as once its marker is known — so the label never
 * contradicts the colour. Window markers are phases themselves; a day without
 * one (e.g. the anchored window says "not fertile" where the legacy flag
 * disagreed) reads as follicular before ovulation and luteal after it.
 */
export function markerPhase(calc: CycleCalculation, marker: CycleDayMarker | null): CyclePhase {
  if (marker === 'period' || marker === 'ovulation' || marker === 'fertile') return marker;
  const phase = calcMainPhase(calc);
  if (phase !== 'ovulation' && phase !== 'fertile') return phase;
  return calc.cycleDay < calc.estimatedOvulationDay ? 'follicular' : 'luteal';
}

/**
 * Turn today's raw calculation into the day-offsets and labels the home screen
 * renders. Pure and locale-free (CLAUDE.md §7) — the UI formats the offsets as
 * localized dates. Offsets are clamped to sane bounds so a slightly stale
 * calculation can't produce nonsense like a negative "days until next period".
 */
export function deriveCyclePredictions(calc: CycleCalculation): CyclePredictions {
  // +1: the next period starts on cycle day cycleLength+1, not cycleLength, so the
  // offset reaches the next period start (matches the backend forecast date).
  const daysUntilNextPeriod = Math.max(0, calc.cycleLength - calc.cycleDay + 1);
  const daysUntilOvulation = calc.estimatedOvulationDay - calc.cycleDay;
  // PMS is the short run of days ending the day before the next period. Clamp to
  // today so an imminent/overdue period never yields a negative window.
  const daysUntilPmsEnd = Math.max(0, daysUntilNextPeriod - 1);
  const daysUntilPmsStart = Math.max(0, daysUntilNextPeriod - PMS_WINDOW_DAYS);

  return {
    cycleDay: calc.cycleDay,
    phase: calcMainPhase(calc),
    cycleLength: calc.cycleLength,
    fertilityPercent: Math.round(Math.min(100, Math.max(0, calc.fertilityPercent))),
    daysUntilNextPeriod,
    daysUntilOvulation,
    daysUntilFertileWindow: daysUntilOvulation - FERTILE_WINDOW_LEAD_DAYS,
    daysUntilPmsStart,
    daysUntilPmsEnd,
    isPeriodTomorrow: calc.isPeriodTomorrow,
    isFertileWindow: calcInFertileWindow(calc),
  };
}
