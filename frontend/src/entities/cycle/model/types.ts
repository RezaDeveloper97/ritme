/** Phases of the menstrual cycle, in the order they occur. */
export type CyclePhase =
  | 'period' // menstruation (bleeding) days
  | 'follicular' // post-period, pre-fertile
  | 'fertile' // fertile window leading up to ovulation
  | 'ovulation' // the estimated ovulation day
  | 'luteal'; // post-ovulation, pre-next-period

/** Relative likelihood of conception on a given day. Informational only (§11). */
export type PregnancyChance = 'low' | 'medium' | 'high';

/** The parameters that define a user's cycle. Gregorian at this boundary (§7). */
export interface CycleConfig {
  /** First day of the current/reference period (day 1 of the cycle). */
  startDate: Date;
  /** Average full cycle length in days (period start → next period start). */
  cycleLength: number;
  /** Average number of bleeding days. */
  periodLength: number;
}

/** What a single calendar day means within the cycle. */
export interface CycleDayInfo {
  /** 1-based day number within the cycle (1 … cycleLength). */
  cycleDay: number;
  phase: CyclePhase;
  pregnancyChance: PregnancyChance;
}

/**
 * The backend's per-day cycle calculation (a subset of the API's
 * `CycleCalculation` schema — CLAUDE.md §8.1). camelCase domain shape produced
 * by the zod boundary parser; probabilities are informational only (§11).
 */
export interface CycleCalculation {
  /** 1-based day within the current cycle. */
  cycleDay: number;
  /** Raw phase string from the API (normalized to `CyclePhase` in selectors). */
  phase: string;
  /** Finer-grained phase label, when the backend provides one. */
  subphase: string | null;
  /** Estimated ovulation day, as a 1-based cycle day. */
  estimatedOvulationDay: number;
  /** Cycle length the calculation was based on (days). */
  cycleLength: number;
  isFertileWindow: boolean;
  isPmsWindow: boolean;
  isPeriodTomorrow: boolean;
  /** Conception probability for the day, as a percentage (0–100). Informational. */
  fertilityPercent: number;
  /** e.g. "regular" / "irregular"; null when unknown. */
  cycleVariability: string | null;
}

/** Summary counts for a month's worth of calculations. */
export interface MonthSummary {
  fertileDays: number;
  periodDays: number;
  pmsDays: number;
}

/**
 * Values the home screen derives from today's `CycleCalculation`. Offsets are
 * whole days from *today* — the UI turns them into Jalali dates through
 * `shared/lib/date` (§7), so this stays pure and locale-free.
 */
export interface CyclePredictions {
  cycleDay: number;
  /** Normalized phase (falls back to `luteal` for unknown backend strings). */
  phase: CyclePhase;
  cycleLength: number;
  /** Conception probability today, 0–100 (rounded). Informational only (§11). */
  fertilityPercent: number;
  /** Days from today until the next period starts (never negative). */
  daysUntilNextPeriod: number;
  /** Days from today until estimated ovulation (negative once it has passed). */
  daysUntilOvulation: number;
  /** Days from today until the fertile window opens (may be negative/zero). */
  daysUntilFertileWindow: number;
  isPeriodTomorrow: boolean;
  isFertileWindow: boolean;
}
