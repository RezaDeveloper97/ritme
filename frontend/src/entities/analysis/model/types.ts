/**
 * Analysis tab (B-N3-07 engine, B-N3-08 hub). Shapes of `GET /api/v1/analysis/*`
 * after the boundary parsers in `api/schema.ts` (snake_case → camelCase).
 * Personal health data (CLAUDE.md §11): never log these.
 */

/** `range` query values (backend-go/internal/analysis/range.go). */
export const ANALYSIS_RANGES = ['7d', '2w', '1m', '3m', '6m', '1y', 'all'] as const;
export type AnalysisRangeKey = (typeof ANALYSIS_RANGES)[number];

/** The hub's selector («۳ ماه · ۶ ماه · ۱ سال», An_Hub) and its default. */
export const HUB_RANGES = ['3m', '6m', '1y'] as const satisfies readonly AnalysisRangeKey[];
export type HubRangeKey = (typeof HUB_RANGES)[number];
export const DEFAULT_RANGE: HubRangeKey = '6m';

export function isAnalysisRange(value: unknown): value is AnalysisRangeKey {
  return typeof value === 'string' && (ANALYSIS_RANGES as readonly string[]).includes(value);
}

/** The window every report carries. */
export interface AnalysisRange {
  key: AnalysisRangeKey;
  from: string;
  to: string;
  days: number;
}

/** One localized sentence: key under the `analysis` namespace, its params and the server-rendered text. */
export interface AnalysisPhrase {
  key: string;
  params: Record<string, string | number>;
  text: string;
}

/**
 * One card of a report. `locked` = a Plus section for a free user (data is
 * null, nothing was computed); `ready` = enough data for the main content.
 */
export interface AnalysisSection<T> {
  plus: boolean;
  locked: boolean;
  ready: boolean;
  data: T | null;
}

export type TopFindingKind =
  | 'cycle_frequent'
  | 'cycle_infrequent'
  | 'period_prolonged'
  | 'cycle_irregular'
  | 'cycle_regular'
  | 'pattern'
  | 'correlation'
  | 'mood_by_phase'
  | 'not_enough_data'
  | 'keep_logging'
  | 'no_data';

/**
 * «مهم‌ترین یافته» (SPEC v6 §4.1, D-175): the best finding by strength ×
 * recency, or the «log N more days» nudge (`daysNeeded`).
 */
export interface TopFinding {
  /** Unknown kinds from a newer backend parse to `null`. */
  kind: TopFindingKind | null;
  parts: AnalysisPhrase[];
  text: string;
  /** 0–1; null for a nudge. */
  strength: number | null;
  score: number | null;
  evidenceDate: string | null;
  /** The nudge's N (days); null for a finding. */
  daysNeeded: number | null;
}

export type Regularity = 'regular' | 'irregular' | 'not_enough_data';
export type CycleStatus = 'normal' | 'frequent' | 'infrequent';

export interface HubCycleBar {
  start: string;
  length: number;
  inFigoRange: boolean;
}

export interface HubCycle {
  medianCycle: number | null;
  medianPeriod: number | null;
  cycleStatus: CycleStatus | null;
  regularity: Regularity;
  variationDays: number | null;
  variationMax: number;
  basedOnCycles: number;
  /** Oldest → newest (the order the bars read). */
  bars: HubCycleBar[];
}

/** A recent cycle as a day layout (1-based days). Newest first. */
export interface HubRecentCycle {
  start: string;
  length: number;
  isCurrent: boolean;
  daysSoFar: number | null;
  periodDays: number;
  fertileStartDay: number;
  fertileEndDay: number;
  ovulationDay: number;
}

export interface SymptomCount {
  key: string;
  label: string;
  days: number;
  cycles: number;
}

export type PatternRelation = 'before_period' | 'early' | 'mid';

export interface SymptomHighlight {
  key: string;
  label: string;
  cycles: number;
  ofCycles: number;
  relation: PatternRelation | null;
  days: number | null;
  startDay: number;
  endDay: number;
}

export interface HubSymptoms {
  cyclesCounted: number;
  cyclesNeeded: number;
  top: SymptomCount[];
  highlight: SymptomHighlight | null;
}

export type CyclePhase = 'period' | 'follicular' | 'fertile' | 'luteal';

/** One mood's share of a phase's mood-logged days. */
export interface MoodShare {
  mood: string;
  label: string;
  days: number;
  pct: number;
}

/** A phase of «حال در فازها»: shares only from 10 mood-logged days (§4.1). */
export interface PhaseMoods {
  phase: CyclePhase;
  ready: boolean;
  /** null below the phase minimum. */
  goodPct: number | null;
  days: number;
  moods: MoodShare[];
}

export interface MoodByPhase {
  ready: boolean;
  minDays: number;
  daysNeeded: number;
  phases: PhaseMoods[];
  finding: AnalysisPhrase | null;
}

export interface CorrelationGroup {
  key: string;
  days: number;
  hits: number;
  pct: number;
}

export interface Correlation {
  key: string;
  status: string;
  strength: string | null;
  n: number;
  minDays: number;
  /** Paired days still missing for the §4.1 minimum (0 once met). */
  daysNeeded: number;
  /** «قوی» = at least `days` paired days and a `ratio`× difference. */
  strongRule: { days: number; ratio: number } | null;
  groups: CorrelationGroup[];
  ratio: number | null;
  finding: AnalysisPhrase | null;
}

export interface WeightPoint {
  date: string;
  /** null on a day without a weigh-in. */
  value: number | null;
  avg7: number | null;
}

export interface HubWeight {
  /** §4.1: 3 weigh-ins in the last 14 days; below it the figures are null. */
  ready: boolean;
  logsInWindow: number;
  minLogs: number;
  windowDays: number;
  daysNeeded: number;
  unit: string;
  current: number | null;
  asOf: string | null;
  delta7d: number | null;
  delta30d: number | null;
  deltaRange: number | null;
  bmi: number | null;
  points: WeightPoint[];
}

/** Readiness of one vitals type (§4.1: 2 readings on 2 different days). */
export interface VitalsReadiness {
  ready: boolean;
  readings: number;
  days: number;
  readingsNeeded: number;
  /** Class → readings (the vitals classifier); null below the minimum. */
  classes: Record<string, number> | null;
}

export interface BloodPressureStats extends VitalsReadiness {
  systolic: number | null;
  diastolic: number | null;
  min: { systolic: number; diastolic: number } | null;
  max: { systolic: number; diastolic: number } | null;
}

export interface BloodSugarStats extends VitalsReadiness {
  avg: number | null;
  min: number | null;
  max: number | null;
}

export interface HubVitals {
  minReadings: number;
  minDays: number;
  bloodPressure: BloodPressureStats | null;
  bloodSugar: BloodSugarStats | null;
  /** Heart-rate readings (bpm); null with none in the range. */
  heartRate: BloodSugarStats | null;
}

/** `GET /analysis/summary` — the hub (An_Hub). */
export interface AnalysisSummary {
  range: AnalysisRange;
  topFinding: TopFinding;
  sections: {
    cycle: AnalysisSection<HubCycle>;
    recentCycles: AnalysisSection<HubRecentCycle[]>;
    symptoms: AnalysisSection<HubSymptoms>;
    moodByPhase: AnalysisSection<MoodByPhase>;
    sleepMood: AnalysisSection<Correlation>;
    weight: AnalysisSection<HubWeight>;
    vitals: AnalysisSection<HubVitals>;
    /** Lab trends arrive with B-N6-06; the body is opaque until then. */
    labs: AnalysisSection<unknown>;
  };
}
