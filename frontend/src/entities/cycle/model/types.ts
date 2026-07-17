/** Phases of the menstrual cycle, in the order they occur. */
export type CyclePhase =
  | 'period' // menstruation (bleeding) days
  | 'follicular' // post-period, pre-fertile
  | 'fertile' // fertile window leading up to ovulation
  | 'ovulation' // the estimated ovulation day
  | 'luteal'; // post-ovulation, pre-next-period

/** Relative likelihood of conception on a given day. Informational only (§11). */
export type PregnancyChance = 'low' | 'medium' | 'high';

/**
 * The colored marker a calendar day gets. Unlike {@link CyclePhase} this is a
 * display concern: the fertile window and PMS window are flags the backend sets
 * *within* the follicular/luteal phases, but the calendar surfaces them as their
 * own colors. A day with no marker reads as neutral.
 */
export type CycleDayMarker = 'period' | 'fertile' | 'ovulation' | 'pms';

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
  /** Gregorian `YYYY-MM-DD` this calculation is for (§7 — display in Jalali). */
  calculationDate: string;
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
  /** Days from today to the first PMS day (the run of days before next period). */
  daysUntilPmsStart: number;
  /** Days from today to the last PMS day (the day before next period). */
  daysUntilPmsEnd: number;
  isPeriodTomorrow: boolean;
  isFertileWindow: boolean;
}

/** How a day's information was derived — drives the daily card's framing (spec §18). */
export type DataStatus = 'actual' | 'predicted' | 'needs_confirmation' | 'incomplete';

/** Calendar-based fertility level on the daily card (spec §17). Informational only (§11). */
export type FertilityLevel = 'very_low' | 'low' | 'medium' | 'high' | 'very_high';

/** How much to trust a prediction (spec §18). */
export type ConfidenceLevel = 'high' | 'medium' | 'low';

/** A daily-card call to action — `type` is a stable code, `label` is localized. */
export interface CardAction {
  type: string;
  label: string;
}

/**
 * The backend-rendered day-status card (spec §19). The client only renders it and
 * never re-implements the Cycle Engine's title rules.
 */
export interface DailyCard {
  title: string;
  subtitle: string;
  dataStatus: DataStatus;
  fertilityLevel: FertilityLevel;
  fertilityLabel: string;
  badges: string[];
  primaryAction: CardAction | null;
  secondaryActions: CardAction[];
}

/** Upcoming period / ovulation / fertile-window forecast (spec §13). Gregorian dates (§7). */
export interface CycleForecast {
  nextPeriodStart: string;
  nextPeriodEnd: string;
  estimatedOvulationDate: string;
  fertileWindowStart: string;
  fertileWindowEnd: string;
  /** Which layer the forecast is based on: recent_valid_cycles | profile | default. */
  source: string;
  confidence: ConfidenceLevel;
  confidenceReasons: string[];
}

/** One layer of the profile / calculated / effective values (spec §12). */
export interface CycleValueLayer {
  cycleLength: number | null;
  periodDuration: number | null;
}

export interface CalculatedValues extends CycleValueLayer {
  /** Number of valid cycles behind the calculated values (null until ≥3 exist). */
  basedOnCycles: number | null;
}

export interface EffectiveValues extends CycleValueLayer {
  source: string;
  periodDurationSource: string;
}

/** Confidence + regularity read-out for the day (spec §18). */
export interface CycleDataQuality {
  confidence: ConfidenceLevel;
  confidenceReasons: string[];
  regularityStatus: string;
  isIrregularPossible: boolean;
  missingPeriodEnd: boolean;
}

/**
 * The full render-ready daily payload (spec §19), served under `data.cycle_view`.
 * Core fields are null when the profile is incomplete (no anchor yet).
 */
export interface CycleView {
  date: string;
  cycleDay: number | null;
  phase: string | null;
  subphase: string | null;
  fertilityLevel: FertilityLevel | null;
  dataStatus: DataStatus | null;
  forecast: CycleForecast | null;
  profileValues: CycleValueLayer;
  calculatedValues: CalculatedValues;
  effectiveValues: EffectiveValues;
  dataQuality: CycleDataQuality;
  dailyCard: DailyCard | null;
}
