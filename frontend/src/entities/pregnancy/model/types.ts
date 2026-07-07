/**
 * Domain types for **pregnancy mode** (CLAUDE.md §1 — `mode` is first-class).
 * These map the API's `pregnancy` endpoint group (§8.1).
 *
 * Two shapes coexist on purpose, mirroring the rest of the codebase:
 * - Read models that the backend *computes* (status, gestational age, due date)
 *   are camelCase — they are display data, not request bodies.
 * - Log **inputs** keep the API's snake_case field names so a daily/weekly log
 *   is one object across the boundary (same approach as `health-log`).
 *
 * Privacy (§11): pregnancy status and symptoms are sensitive health data — none
 * of it is ever put in URLs, query strings, or logs.
 */

// ── Localization ───────────────────────────────────────────────
/** A bilingual string the backend already localized both ways; the UI picks. */
export interface Bilingual {
  en: string;
  fa: string;
}

// ── Status / read models ───────────────────────────────────────
export interface GestationalAge {
  weeks: number | null;
  days: number | null;
  totalDays: number | null;
  trimester: number | null;
  confidenceLevel: string | null;
  confidenceLabel: string | null;
  uncertaintyDays: number | null;
  formatted: Bilingual | null;
}

export interface EstimatedDueDate {
  /** Gregorian `YYYY-MM-DD` — format for display via `shared/lib/date` (§7). */
  date: string;
  daysRemaining: number;
  weeksRemaining: number;
  formatted: Bilingual;
}

/** Contextual info/warning banners the backend derives from the profile. */
export type PregnancyFlags = Record<string, Bilingual | string>;

/** GET /pregnancy/status — the single source of truth for the tracker screen. */
export interface PregnancyStatus {
  isActive: boolean;
  gestationalAge: GestationalAge | null;
  estimatedDueDate: EstimatedDueDate | null;
  currentWeek: number | null;
  trimester: number | null;
  ageSource: string | null;
  confidenceLevel: string | null;
  isHighRisk: boolean;
  fetalMovementTrackingActive: boolean;
  fetalMovementRequired: boolean;
  flags: PregnancyFlags;
}

/** The stored dating/history profile (useful subset of the API model). */
export interface PregnancyProfile {
  pregnancyMode: boolean;
  onboardingCompleted: boolean;
  isLocked: boolean;
  ageSource: string | null;
  confidenceLevel: string | null;
  lmpDate: string | null;
  ultrasoundDate: string | null;
  ultrasoundWeeks: number | null;
  ultrasoundDays: number | null;
  manualWeeks: number | null;
  manualDays: number | null;
  estimatedDueDate: string | null;
  uncertaintyDays: number | null;
  hasMiscarriageHistory: boolean | null;
  hasHighRiskHistory: boolean | null;
  preExistingConditions: string[];
  bloodType: string | null;
  rhFactor: string | null;
  rhNegativeCareFlag: boolean;
  fetalMovementFelt: boolean;
  firstFetalMovementDate: string | null;
}

/** GET /pregnancy/activate — the mode-switch result. */
export interface PregnancyActivation {
  pregnancyMode: boolean;
  cycleMode: boolean;
  onboardingRequired: boolean;
}

// ── Enums (option lists that drive the forms; §8.1 — derive, never hardcode) ──
export type AgeSource = 'lmp' | 'ultrasound' | 'manual';

export interface PregnancyEnums {
  ageSources: string[];
  confidenceLevels: string[];
  bloodTypes: string[];
  rhFactors: string[];
  preExistingConditions: string[];
  alertLevels: string[];
}

export interface SymptomEnums {
  severity: string[];
}

export interface WeeklyEnums {
  swellingLocations: string[];
  overallMood: string[];
  severity: string[];
  fetalMovementStatus: string[];
}

// ── Onboarding input ───────────────────────────────────────────
/** POST /pregnancy/onboarding body (§8.1). `age_source` decides which dating
 *  fields are required — validated again on the server. */
export interface OnboardingInput {
  age_source: AgeSource;
  lmp_date?: string;
  ultrasound_date?: string;
  ultrasound_weeks?: number;
  ultrasound_days?: number;
  manual_weeks?: number;
  manual_days?: number;
  has_miscarriage_history?: boolean;
  has_high_risk_history?: boolean;
  pre_existing_conditions?: string[];
  blood_type?: string;
  rh_factor?: string;
}

// ── Daily symptom log ──────────────────────────────────────────
/** The 14 tracked symptoms. Four are "critical" (spotting, bleeding,
 *  fluid_leakage, severe_sudden_pain) and can raise alerts on save. */
export type SymptomKey =
  | 'nausea'
  | 'vomiting'
  | 'fatigue'
  | 'headache'
  | 'dizziness'
  | 'breast_pain'
  | 'lower_abdominal_pain'
  | 'cramping'
  | 'back_pain'
  | 'pelvic_pressure'
  | 'spotting'
  | 'bleeding'
  | 'fluid_leakage'
  | 'severe_sudden_pain';

/** POST /pregnancy/symptoms body — `log_date` plus any `has_<x>`/`<x>_severity`
 *  pairs the user recorded. Upserts by date. */
export interface SymptomLogInput {
  log_date: string;
  notes?: string;
  // Indexable so `has_${SymptomKey}` / `${SymptomKey}_severity` pairs can be
  // set generically from the field config without 28 explicit keys.
  [key: string]: string | boolean | undefined;
}

// ── Weekly checkup log ─────────────────────────────────────────
/** POST /pregnancy/weekly body — one row per `pregnancy_week` (upserts). */
export interface WeeklyLogInput {
  log_date: string;
  pregnancy_week: number;
  weight?: number;
  has_swelling?: boolean;
  swelling_locations?: string[];
  has_shortness_of_breath?: boolean;
  has_blood_pressure_device?: boolean;
  systolic_pressure?: number;
  diastolic_pressure?: number;
  fasting_blood_sugar?: number;
  post_meal_blood_sugar?: number;
  overall_mood?: string;
  has_anxiety?: boolean;
  anxiety_severity?: string;
  has_mood_swings?: boolean;
  mood_swings_severity?: string;
  has_depression_feelings?: boolean;
  depression_severity?: string;
  notes?: string;
}

// ── Fetal movement ─────────────────────────────────────────────
export type MovementStatus =
  | 'not_felt_yet'
  | 'felt'
  | 'normal'
  | 'reduced'
  | 'increased'
  | 'none';

/** POST /pregnancy/fetal-movement body (upserts by date). */
export interface FetalMovementInput {
  log_date: string;
  pregnancy_week: number;
  movement_status: string;
  movement_count?: number;
  first_movement_time?: string;
  last_movement_time?: string;
  notes?: string;
}

// ── Weekly educational content ─────────────────────────────────
export interface WeeklyFaqItem {
  question: string;
  answer: string;
}

/** GET /pregnancy/content/{week}?locale=… — already resolved to one locale. */
export interface WeeklyContent {
  weekNumber: number;
  fetalDevelopment: string | null;
  motherBodyChanges: string | null;
  dosAndDonts: string | null;
  carePlan: string | null;
  bodyAdaptation: string | null;
  emotionalStatus: string | null;
  keyNutrition: string | null;
  physicalActivity: string | null;
  testsAndCheckups: string | null;
  faq: WeeklyFaqItem[];
}

/** The nine long-form content modules, in display order (faq is rendered
 *  separately). Keys map to `pregnancyContent.modules.<key>` i18n titles. */
export type ContentModuleKey =
  | 'fetalDevelopment'
  | 'motherBodyChanges'
  | 'bodyAdaptation'
  | 'emotionalStatus'
  | 'keyNutrition'
  | 'physicalActivity'
  | 'dosAndDonts'
  | 'carePlan'
  | 'testsAndCheckups';

// ── Alerts ─────────────────────────────────────────────────────
export type AlertLevel = 'info' | 'warning' | 'emergency';

export interface PregnancyAlert {
  id: number;
  alertLevel: AlertLevel;
  alertType: string;
  title: string;
  message: string;
  pregnancyWeek: number | null;
  recommendedActions: string[];
  isRead: boolean;
  isDismissed: boolean;
  createdAt: string | null;
}

export interface AlertCounts {
  total: number;
  unread: number;
  emergency: number;
  warning: number;
  info: number;
}

export interface AlertSummary {
  hasEmergency: boolean;
  hasUnread: boolean;
  counts: AlertCounts;
  latestEmergency: PregnancyAlert | null;
}
