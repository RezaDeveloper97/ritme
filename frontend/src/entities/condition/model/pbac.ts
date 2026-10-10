/*
 * Heavy-bleeding pictorial chart (PBAC, CB-COND-01 API, screen CB-COND-05): pads per soak level, the clot chip and
 * flooding give the day's score; the period's score sums the logged days of a 10-day window. Health data (§11): never
 * log any of it. The points mirror the API (`GET /conditions/pbac/{date}`) so the tiles move while she edits.
 */

/** Soak levels in board order (کمی · نیمه · کاملاً خیس). */
export const PBAC_LEVELS = ['light', 'medium', 'heavy'] as const;
export type PbacLevel = (typeof PBAC_LEVELS)[number];

export const PBAC_CLOTS = ['none', 'small', 'large'] as const;
export type PbacClots = (typeof PBAC_CLOTS)[number];

export const PBAC_LEVEL_POINTS: Record<PbacLevel, number> = { light: 1, medium: 5, heavy: 20 };
export const PBAC_CLOT_POINTS: Record<PbacClots, number> = { none: 0, small: 1, large: 5 };
export const PBAC_FLOODING_POINTS = 5;
/** Pads per level the API accepts. */
export const PBAC_MAX_PADS = 50;

export interface PbacEntry {
  light: number;
  medium: number;
  heavy: number;
  clots: PbacClots | null;
  flooding: boolean;
}

export interface PbacPeriod {
  start: string;
  end: string;
  /** 1-based day of the period («روز ۲ پریود»). */
  day: number;
  score: number;
  daysLogged: number;
}

/** A `condition_alerts` row picked for the request locale (here `pbac_over_100`). */
export interface PbacAlert {
  code: string;
  title: string | null;
  body: string | null;
}

export interface PbacDay extends PbacEntry {
  date: string;
  score: number;
  period: PbacPeriod | null;
  alertThreshold: number;
  alert: PbacAlert | null;
}

/** The day's PBAC score of an entry (the API's formula). */
export function pbacScore(entry: PbacEntry): number {
  const pads = PBAC_LEVELS.reduce((sum, level) => sum + entry[level] * PBAC_LEVEL_POINTS[level], 0);
  return pads + (entry.clots ? PBAC_CLOT_POINTS[entry.clots] : 0) + (entry.flooding ? PBAC_FLOODING_POINTS : 0);
}
