import type { AnalysisPhrase } from './types';

/**
 * `GET /api/v1/analysis/labs` (T-M10-26, SPEC v6 §4.1 «روند آزمایش», Plus): one
 * marker across the user's consecutive confirmed lab reports, a trend from 2
 * reports. The sentence describes the numbers only (never a diagnosis).
 */
export type LabTrendDirection = 'rising' | 'falling' | 'stable';

export interface LabTrendMarker {
  /** Catalog code, else `name:<printed name>` (the same key as `/labs/trends`). */
  marker: string;
  name: string;
  unit: string | null;
  reports: number;
  ready: boolean;
  reportsNeeded: number;
  direction: LabTrendDirection | null;
  finding: AnalysisPhrase | null;
}

export interface LabTrendList {
  locked: boolean;
  ready: boolean;
  minReports: number;
  markers: LabTrendMarker[];
  /** The non-diagnostic note («این فقط مسیر عددهاست، نه تشخیص…»). */
  note: AnalysisPhrase | null;
}

/** A marker's trend row by key (the `/labs/trends` series key). */
export function trendByMarker(list: LabTrendList | undefined, key: string): LabTrendMarker | null {
  return list?.markers.find((m) => m.marker === key) ?? null;
}
