/*
 * PMDD daily questionnaire + cycle chart (CB-COND-01 API, screen CB-COND-04): every `pmdd_items` item rated
 * 1 (not at all) … 6 (extreme) per day. Mood ratings are health data (§11): never log any of it.
 */

export const PMDD_MIN = 1;
export const PMDD_MAX = 6;

/** A `pmdd_items` catalog row (admin copy in the request locale, catalog order). */
export interface PmddItem {
  code: string;
  title: string;
}

/** `GET|PUT /conditions/pmdd/{date}`. */
export interface PmddDay {
  date: string;
  /** item code → 1–6 (only the rated items). */
  scores: Record<string, number>;
  mean: number | null;
}

export interface PmddWindow {
  from: string;
  to: string;
}

export interface PmddChartDay {
  date: string;
  cycleDay: number;
  inPeriod: boolean;
  lateLuteal: boolean;
  mean: number;
}

export interface PmddCycle {
  start: string;
  end: string;
  closed: boolean;
  complete: boolean;
  period: PmddWindow | null;
  lateLuteal: PmddWindow | null;
  follicular: PmddWindow | null;
  lutealMean: number | null;
  follicularMean: number | null;
  days: PmddChartDay[];
}

/** A `condition_alerts` row (pattern sentence, «needs two cycles», crisis hotlines). */
export interface PmddAlert {
  code: string;
  title: string | null;
  body: string | null;
  severity: 'info' | 'caution' | 'urgent';
  hotlines: { number: string; label: string | null }[];
}

export const PMDD_PATTERNS = ['luteal', 'unclear', 'not_enough_data'] as const;
export type PmddPattern = (typeof PMDD_PATTERNS)[number];

/** `GET /conditions/pmdd/chart`: the current cycle first, then up to two closed ones. */
export interface PmddChart {
  requiredCycles: number;
  completeCycles: number;
  ready: boolean;
  pattern: PmddPattern;
  cycles: PmddCycle[];
  alerts: PmddAlert[];
}
