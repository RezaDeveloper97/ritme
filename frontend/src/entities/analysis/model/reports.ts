/**
 * Detail reports of the analysis tab (B-N3-07 engine, B-N3-09 screens):
 * `GET /api/v1/analysis/{cycle,period,symptoms,correlations,body}?range=`,
 * after the boundary parsers in `api/reports.ts` (snake_case → camelCase).
 * Personal health data (CLAUDE.md §11): never log these.
 */
import type {
  AnalysisRange,
  Correlation,
  CycleStatus,
  HubWeight,
  PatternRelation,
  Regularity,
  SymptomCount,
  SymptomHighlight,
} from './types';

/** The typical cycle as phase lengths (1-based days). */
export interface CycleLayout {
  cycleLength: number;
  ovulationDay: number;
  fertileStartDay: number;
  fertileEndDay: number;
  days: { period: number; follicular: number; fertile: number; luteal: number };
}

/** Why a cycle was left out of the median: outside 21–45 days, or a single outlier. */
export type CycleExclusion = 'implausible' | 'outlier';

export interface ReportCycle {
  start: string;
  end: string;
  length: number;
  periodDays: number;
  inFigoRange: boolean;
  counted: boolean;
  excluded: CycleExclusion | null;
}

/** `GET /analysis/cycle` (An_Cycle). Cycles are newest first. */
export interface CycleReport {
  range: AnalysisRange;
  ready: boolean;
  basedOnCycles: number;
  figo: { applies: boolean; cycleMin: number; cycleMax: number; periodMax: number; variationMax: number };
  cycleLength: { median: number | null; status: CycleStatus | null };
  periodLength: { median: number | null; status: PeriodStatus | null };
  variation: { days: number | null; max: number; status: Regularity; cyclesNeeded: number };
  typical: CycleLayout;
  current: { start: string; day: number } | null;
  cycles: ReportCycle[];
}

export type PeriodStatus = 'normal' | 'prolonged';
export type FlowLevel = 'light' | 'medium' | 'heavy' | 'very_heavy';

export interface ReportPeriod {
  start: string;
  length: number;
  closed: boolean;
  isCurrent: boolean;
  days: Array<{ day: number; date: string; flow: FlowLevel | null }>;
}

/** `GET /analysis/period` (An_Period). Periods are newest first. */
export interface PeriodReport {
  range: AnalysisRange;
  ready: boolean;
  basedOnPeriods: number;
  periodMax: number;
  periodLength: { median: number | null; status: PeriodStatus | null };
  peak: { day: number; level: FlowLevel | null } | null;
  spotting: { days: number; cycles: number; ofCycles: number };
  /** Mean flow per period day (score 1–4, null on a day without flow logged). */
  average: Array<{ day: number; score: number | null; level: FlowLevel | null }>;
  periods: ReportPeriod[];
  coSymptoms: { ready: boolean; periodsNeeded: number; items: Array<{ key: string; label: string; periods: number; pct: number }> };
}

export interface SymptomPatternItem {
  key: string;
  label: string;
  cycles: number;
  /** One share (0–1) per day of the typical cycle, day 1 first. */
  strip: number[];
  window: { startDay: number; endDay: number; relation: PatternRelation | null; days: number } | null;
}

/** `GET /analysis/symptoms` (An_Symptoms). */
export interface SymptomsReport {
  range: AnalysisRange;
  daysLogged: number;
  symptomDays: number;
  top: SymptomCount[];
  trend: {
    ready: boolean;
    bucket: 'day' | 'week';
    keys: string[];
    /** Oldest first; `values[i]` belongs to `keys[i]`. */
    points: Array<{ start: string; values: number[] }>;
  };
  pattern: {
    ready: boolean;
    cyclesCounted: number;
    cyclesNeeded: number;
    typical: CycleLayout;
    items: SymptomPatternItem[];
  };
  highlight: SymptomHighlight | null;
}

/** The pairs An_Correlations explains (backend-go/internal/analysis/correlations.go). */
export const CORRELATION_KEYS = ['sleep_mood', 'phase_energy', 'exercise_cramps', 'phase_mood'] as const;
export type CorrelationKey = (typeof CORRELATION_KEYS)[number];

/** `GET /analysis/correlations` (Plus). A free user gets `locked: true` and no data. */
export interface CorrelationsReport {
  range: AnalysisRange;
  notCausal: boolean;
  plus: boolean;
  locked: boolean;
  ready: boolean;
  data: { daysLogged: number; items: Array<Correlation & { key: CorrelationKey }> } | null;
}

/** `GET /analysis/body` (An_Body). */
export interface BodyReport {
  range: AnalysisRange;
  weight: HubWeight & { movingAverageDays: number };
  sleep: {
    ready: boolean;
    nights: number;
    avgHours: number | null;
    lutealAvgHours: number | null;
    /** Seven averages, Saturday first (null = no night logged on that weekday). */
    byWeekday: Array<number | null>;
  };
  activity: { activeDays: number; days: number; weeks: Array<{ start: string; days: number; activeDays: number }> };
}

/** Range tabs of each detail screen (artboards: An_Symptoms 2w–6m, An_Body 7d–1y; the rest follow the hub). */
export const REPORT_RANGES = {
  cycle: ['3m', '6m', '1y'],
  period: ['3m', '6m', '1y'],
  symptoms: ['2w', '1m', '3m', '6m'],
  correlations: ['3m', '6m', '1y'],
  body: ['7d', '1m', '3m', '1y'],
} as const;
export type ReportName = keyof typeof REPORT_RANGES;

/** Default window per report (An_Symptoms opens on «۲ هفته», An_Body on «۳۰ روز»). */
export const REPORT_DEFAULT_RANGE: { [K in ReportName]: (typeof REPORT_RANGES)[K][number] } = {
  cycle: '6m',
  period: '6m',
  symptoms: '2w',
  correlations: '6m',
  body: '1m',
};
