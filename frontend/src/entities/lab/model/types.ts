/*
 * Lab analysis (B-N6-06 API, B-N6-07 screens): `/api/v1/labs/*`, Go only.
 * Health data (CLAUDE.md §11): never log a lab, a marker or a value.
 */

/** Upload form categories, in the order the server lists them (`limits.categories`). */
export const LAB_CATEGORIES = ['blood', 'hormone', 'thyroid', 'urine', 'other'] as const;
export type LabCategory = (typeof LAB_CATEGORIES)[number];

export type LabStatus = 'queued' | 'extracting' | 'needs_review' | 'interpreting' | 'ready' | 'failed';
/** Processing screen stage (queued → reading → extracting → review → explaining → done | failed). */
export type LabStage = 'queued' | 'reading' | 'extracting' | 'review' | 'explaining' | 'done' | 'failed';
export type MarkerState = 'low' | 'borderline_low' | 'normal' | 'borderline_high' | 'high' | 'unknown';

/** The consent the AI lab analysis needs (`internal/ai/access`, versioned catalog). */
export const LAB_CONSENT_CODE = 'ai_lab_analysis';
/** Plus entitlement key of the upload. */
export const LAB_PLUS_FEATURE = 'plus.lab_ai';

export interface LabLimits {
  maxFiles: number;
  maxImageKb: number;
  maxPdfKb: number;
  categories: string[];
}

export interface LabListItem {
  id: number;
  source: 'upload' | 'manual';
  category: string;
  title: string;
  date: string;
  status: LabStatus;
  stage: LabStage;
  progress: number;
  markerCount: number;
  attentionCount: number;
  allNormal: boolean;
}

export interface LabList {
  labs: LabListItem[];
  limits: LabLimits;
}

export interface MarkerReference {
  low: number | null;
  high: number | null;
  text: string | null;
  /** `sheet` (printed on the sheet) or `catalog` (typical range); null when there is none. */
  source: string | null;
}

export interface LabMarker {
  id: number;
  code: string | null;
  name: string;
  printedName: string;
  subtitle: string | null;
  value: number | null;
  valueText: string | null;
  unit: string | null;
  reference: MarkerReference;
  state: MarkerState;
  stateLabel: string;
  attention: boolean;
  confidence: number | null;
  lowConfidence: boolean;
  source: string;
}

export interface LabFile {
  id: number;
  page: number;
  mime: string;
}

export interface RedFlag {
  markerId: number;
  name: string;
  severity: 'urgent' | 'soon';
  message: string;
}

export interface LabInterpretation {
  /** Plain text — rendered as text, never as HTML. */
  summary: string;
  source: 'ai' | 'rules';
  stale: boolean;
  redFlags: RedFlag[];
  doctorQuestions: string[];
  disclaimer: string;
}

export interface LabCounts {
  total: number;
  normal: number;
  attention: number;
  low: number;
  borderlineLow: number;
  high: number;
  borderlineHigh: number;
  unknown: number;
}

export interface Lab {
  id: number;
  source: 'upload' | 'manual';
  category: string;
  title: string;
  takenOn: string | null;
  date: string;
  fasting: boolean | null;
  labName: string | null;
  status: LabStatus;
  stage: LabStage;
  progress: number;
  errorCode: string | null;
  errorMessage: string | null;
  editable: boolean;
  counts: LabCounts;
  lowConfidenceCount: number;
  markers: LabMarker[];
  files: LabFile[];
  interpretation: LabInterpretation | null;
  feedback: { helpful: boolean } | null;
}

export interface LabStatusPoll {
  id: number;
  status: LabStatus;
  stage: LabStage;
  progress: number;
  markerCount: number;
  errorMessage: string | null;
}

export interface TrendPoint {
  labId: number;
  markerId: number;
  date: string;
  value: number;
  state: MarkerState;
}

export interface MarkerTrend {
  count: number;
  direction: string | null;
  sentence: string;
  points: TrendPoint[];
}

export interface MarkerDetail {
  lab: { id: number; title: string; date: string; status: LabStatus };
  marker: LabMarker;
  about: { name: string; subtitle: string | null; body: string | null; typicalRange: string | null } | null;
  factors: string[];
  seeDoctor: string | null;
  contextNotes: string[];
  redFlag: RedFlag | null;
  trend: MarkerTrend;
  disclaimer: string;
}

export interface TrendSeries {
  key: string;
  name: string;
  unit: string | null;
  referenceText: string | null;
  latest: { labId: number; markerId: number; value: number | null; valueText: string | null; state: MarkerState; stateLabel: string; attention: boolean };
  trend: MarkerTrend;
}

export interface LabTrends {
  labsCount: number;
  minPoints: number;
  markers: TrendSeries[];
}

export interface CatalogMarker {
  code: string;
  name: string;
  unit: string | null;
  typicalRange: string | null;
}

export interface LabConsent {
  code: string;
  version: number;
  title: string;
  body: string;
  points: string[];
  granted: boolean;
  needsConsent: boolean;
}

/** Body of POST / PUT `/labs/{id}/markers[/{mid}]`. */
export interface MarkerInput {
  name: string;
  value: number | null;
  valueText: string | null;
  unit: string | null;
  refLow: number | null;
  refHigh: number | null;
  refText: string | null;
}
