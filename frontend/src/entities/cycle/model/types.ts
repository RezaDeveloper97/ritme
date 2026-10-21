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

/**
 * Visual weight of a day's marker tint. Graded by the day's conception
 * probability relative to the other days carrying the same marker, so e.g.
 * fertile-window cells deepen toward ovulation. Informational only (§11).
 */
export type MarkerIntensity = 'faint' | 'medium' | 'strong';

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
 * A phase/symptom-driven tip for the day, already localized by the backend.
 * `type` is a stable category code (`nutrition`, `sleep`, …) the UI maps to an
 * icon. Display-only health copy — never log it (§11).
 *
 * `title` is the localized heading the backend resolved — an admin's override
 * from the recommendations panel, else the category's own label. It is optional
 * so an older backend (which sent only `type` + `text`) still parses; the UI
 * falls back to its own translated category label then.
 */
export interface CycleDailyTip {
  type: string;
  title?: string;
  text: string;
}

/**
 * The backend's per-day cycle calculation (a subset of the API's
 * `CycleCalculation` schema — CLAUDE.md §8.1). camelCase domain shape produced
 * by the zod boundary parser; probabilities are informational only (§11).
 */
export interface CycleCalculation {
  /** Gregorian `YYYY-MM-DD` this calculation is for (§7 — display localized). */
  calculationDate: string;
  /** 1-based day within the current cycle. */
  cycleDay: number;
  /** Raw phase string from the API (normalized to `CyclePhase` in selectors). */
  phase: string;
  /** Finer-grained phase label, when the backend provides one. */
  subphase: string | null;
  /** Estimated ovulation day, as a 1-based cycle day; null when none is estimated (v6, e.g. hormonal contraception). */
  estimatedOvulationDay: number | null;
  /** Cycle length the calculation was based on (days). */
  cycleLength: number;
  isFertileWindow: boolean;
  isPmsWindow: boolean;
  isPeriodTomorrow: boolean;
  /** Conception probability for the day, as a percentage (0–100). Informational. */
  fertilityPercent: number;
  /** e.g. "regular" / "irregular"; null when unknown. */
  cycleVariability: string | null;
  /** Today's recommendations for this day; empty when the engine has none. */
  dailyTips: CycleDailyTip[];
}

/** Summary counts for a month's worth of calculations. */
export interface MonthSummary {
  fertileDays: number;
  periodDays: number;
  pmsDays: number;
}

/**
 * Values the home screen derives from today's `CycleCalculation`. Offsets are
 * whole days from *today* — the UI turns them into localized dates through
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
export type FertilityLevel =
  'very_low' | 'low' | 'medium' | 'high' | 'very_high' | 'peak' | 'unknown';

/** How much to trust a prediction. `unknown` = no resolvable anchor (v1.1 §29). */
export type ConfidenceLevel = 'high' | 'medium' | 'low' | 'unknown';

/** v1.1 engine main phase (task.md §13) — the authoritative classification. */
export type MainPhase =
  'menstrual' | 'follicular' | 'fertile' | 'luteal' | 'period_expected' | 'unknown';

/** v1.1 fertility level for `cycle_view.fertility_level` (task.md §26). Informational (§11). */
export type CycleFertilityLevel = 'none' | 'low' | 'medium' | 'high' | 'peak' | 'unknown';

/** Deterministic data-quality grade of the engine output (task.md §28). */
export type DataQualityLevel = 'good' | 'partial' | 'poor' | 'insufficient';

/** The primary source the engine resolved the day from (task.md §30). */
export type ResolutionSource =
  | 'user_logged'
  | 'user_logged_with_assumed_end'
  | 'prediction'
  | 'onboarding_based'
  | 'default_based'
  | 'unknown';

/**
 * The v1.1 anchors block (task.md §10, §35): the resolved current period with
 * explicit sources, plus the predicted next start and ovulation estimate.
 * Gregorian dates at this boundary (§7).
 */
export interface CycleAnchors {
  currentPeriodStart: string | null;
  currentPeriodStartSource: string;
  currentPeriodEnd: string | null;
  currentPeriodEndSource: string;
  currentPeriodEndIsConfirmed: boolean;
  predictedNextPeriodStart: string | null;
  estimatedOvulationDate: string | null;
  /** v6: how the ovulation date was established. Optional so Laravel payloads type-check. */
  ovulationSource?: OvulationSource;
}

/** Where the ovulation estimate came from (v6 §2.20). Unknown values degrade to `unknown`. */
export type OvulationSource = 'bbt_confirmed' | 'lh_positive' | 'calendar' | 'unknown';

/** Cycle regularity label (v6 §2.27). */
export type CycleRegularity = 'regular' | 'irregular' | 'insufficient_data';

/** The v1.1 effective metrics block (task.md §9, §27, §35). */
export interface CycleEngineMetrics {
  /** Null when the engine could not resolve a length (v6). */
  effectiveCycleLength: number | null;
  effectivePeriodLength: number | null;
  /** max − min of the last ≤3 valid cycle lengths; null with fewer than 2. */
  cycleVariability: number | null;
  /** v6: regular / irregular / insufficient_data; null when absent. */
  cycleRegularity?: CycleRegularity | null;
}

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
  nextPeriodStart: string | null;
  nextPeriodEnd: string | null;
  estimatedOvulationDate: string | null;
  fertileWindowStart: string | null;
  fertileWindowEnd: string | null;
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
 * The full render-ready daily payload, served under `data.cycle_view`: the v1.1
 * engine status (task.md §35 — main phase, day counters, anchors, warnings,
 * quality/confidence) merged with the legacy render keys (daily card, forecast,
 * three-layer values). Core fields are null when no anchor exists yet.
 */
export interface CycleView {
  date: string;
  cycleDay: number | null;
  /** Legacy 4-phase value derived from `mainPhase`; null when unknown. */
  phase: string | null;
  subphase: string | null;
  mainPhase: MainPhase | null;
  fertilityLevel: CycleFertilityLevel | null;
  dataStatus: DataStatus | null;
  /** Days until the estimated ovulation (negative once passed); null when unresolved (§34). */
  daysToOvulation: number | null;
  /** Days until the predicted next period, floored at 0 (§22.1). */
  daysToPeriod: number | null;
  /** Days past the predicted period start, floored at 0 (§22.1). */
  daysLate: number | null;
  anchors: CycleAnchors | null;
  metrics: CycleEngineMetrics | null;
  confidence: ConfidenceLevel | null;
  confidenceReasons: string[];
  /** v1.1 deterministic grade (§28); the old quality object is `dataQualityDetails`. */
  dataQuality: DataQualityLevel | null;
  dataQualityDetails: CycleDataQuality | null;
  resolutionSource: ResolutionSource | null;
  isPredicted: boolean;
  requiresUserInput: boolean;
  /** Stable warning codes (§32) — soft signals, never blocking. */
  warnings: string[];
  forecast: CycleForecast | null;
  profileValues: CycleValueLayer;
  calculatedValues: CalculatedValues;
  effectiveValues: EffectiveValues;
  dailyCard: DailyCard | null;
  // --- v6 additions: optional so Laravel payloads / old fixtures type-check ---
  /** Predicted cycles after the current one (k = 2 is the cycle after next). */
  futureCycles?: CycleFutureCycle[];
  clinicalFlags?: CycleClinicalFlags | null;
  suggestPregnancyTest?: boolean;
  todayUi?: CycleTodayUi | null;
  limits?: CycleLimits | null;
}

export interface CycleFutureCycle {
  k: number;
  periodStart: string | null;
  ovulation: string | null;
}

/** FIGO-style flags (v6 §2.27); labels are open strings so a new one never throws. */
export interface CycleClinicalFlags {
  frequency: string | null;
  duration: string | null;
  absent: boolean;
}

export interface CycleDateRange {
  start: string | null;
  end: string | null;
}

export interface CycleTodayUi {
  /** `start | end | edit` (E-17); open string so new actions degrade. */
  ringAction: string | null;
  /** `end_missing | overdue | ask_last_period` or null. */
  banner: string | null;
  showPregnancyTestButton: boolean;
  predictionCard: {
    period: CycleDateRange | null;
    pms: CycleDateRange | null;
    fertile: CycleDateRange | null;
    ovulation: { date: string | null; source: OvulationSource } | null;
    cycleLength: number | null;
    cycleRegularity: CycleRegularity | null;
  } | null;
}

/** Output limits (E-24). `fertilitySuppressed` is clinical suppression only. */
export interface CycleLimits {
  fertilitySuppressed: boolean;
  suppressionReason: string | null;
  engineSuspended: boolean;
  promptPregnancyMode: boolean;
}
