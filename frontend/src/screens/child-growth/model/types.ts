import type { ChildMeasurement, GrowthVerdict } from '@/entities/child';

export const GROWTH_INDICATORS = ['weight', 'length', 'head'] as const;
export type GrowthIndicator = (typeof GROWTH_INDICATORS)[number];

/** GET /children/{id}/measurements — newest first (the birth values are the last row). */
export interface MeasurementList {
  measurements: ChildMeasurement[];
  verdict: GrowthVerdict;
  disclaimer: string | null;
}

/** One WHO reference month (values in the indicator's unit). */
export interface GrowthReferenceRow {
  month: number;
  p3: number;
  p15: number;
  p50: number;
  p85: number;
  p97: number;
}

/** One of the child's values on the chart (oldest first; `id` null = the birth values). */
export interface GrowthPoint {
  id: number | null;
  measuredOn: string;
  ageMonths: number;
  value: number;
  percentile: number | null;
  inBand: boolean | null;
}

/** GET /children/{id}/growth?indicator= */
export interface GrowthSeries {
  indicator: GrowthIndicator;
  label: string;
  unitLabel: string;
  /** False when there are no reference curves (sex unknown → `reason: sex_unknown`). */
  available: boolean;
  reason: string | null;
  fromMonth: number;
  toMonth: number;
  bandLabel: string;
  medianLabel: string | null;
  reference: GrowthReferenceRow[];
  points: GrowthPoint[];
  latest: GrowthPoint | null;
  verdict: GrowthVerdict;
  disclaimer: string | null;
}

/** The add/edit form; values are canonical decimal text ('' = not given). */
export interface MeasurementForm {
  measuredOn: string;
  weightKg: string;
  lengthCm: string;
  headCm: string;
}
