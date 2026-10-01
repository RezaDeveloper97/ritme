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
  | 'not_enough_data'
  | 'no_data';

export interface TopFinding {
  /** Unknown kinds from a newer backend parse to `null`. */
  kind: TopFindingKind | null;
  parts: AnalysisPhrase[];
  text: string;
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

export interface MoodByPhase {
  phases: Array<{ phase: CyclePhase; goodPct: number | null; days: number }>;
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
  ready: boolean;
  unit: string;
  current: number | null;
  asOf: string | null;
  delta7d: number | null;
  delta30d: number | null;
  deltaRange: number | null;
  bmi: number | null;
  points: WeightPoint[];
}

export interface HubVitals {
  bloodPressure: { systolic: number; diastolic: number; readings: number } | null;
  bloodSugar: { avg: number; readings: number } | null;
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
