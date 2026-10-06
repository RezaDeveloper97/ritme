/**
 * Health record «پرونده سلامت من» (bloom B-N6-03, `GET /api/v1/health-record`). Owner-only, health data of the most
 * sensitive kind (CLAUDE.md §11): never logged, never in a URL. Dates are Gregorian `YYYY-MM-DD` from the API.
 */

export const BLOOD_TYPES = ['A+', 'A-', 'B+', 'B-', 'AB+', 'AB-', 'O+', 'O-'] as const;
export type BloodType = (typeof BLOOD_TYPES)[number];

/** Codes of the onboarding Conditions step (PUT /onboarding/steps/conditions), screen order. */
export const CHRONIC_ILLNESSES = ['diabetes', 'hypertension', 'thyroid', 'asthma', 'anemia', 'migraine', 'other'] as const;
export const GYN_CONDITIONS = ['pcos', 'endometriosis', 'fibroids', 'recurrent_infections', 'other'] as const;
export type ChronicIllness = (typeof CHRONIC_ILLNESSES)[number];
export type GynCondition = (typeof GYN_CONDITIONS)[number];

/** Manual entry outcomes the user can choose; the record also emits `ongoing` and `birth` for tracked rows. */
export const MANUAL_OUTCOMES = ['vaginal', 'cesarean', 'ended'] as const;
export type ManualOutcome = (typeof MANUAL_OUTCOMES)[number];
export type PregnancyOutcome = ManualOutcome | 'ongoing' | 'birth';

export const MAX_ALLERGIES = 20;
export const MAX_ALLERGY_LENGTH = 60;
export const MAX_BABY_COUNT = 4;

export interface RecordPerson {
  name: string | null;
  age: number | null;
  gender: string | null;
  lifeMode: string;
}

export interface Basics {
  heightCm: number | null;
  weightKg: number | null;
  bmi: { value: number; category: string } | null;
  bloodType: string | null;
  bloodTypeSource: 'record' | 'pregnancy' | null;
}

export interface Conditions {
  chronicIllnesses: string[] | null;
  gynConditions: string[] | null;
  answered: boolean;
}

export interface RecordMedication {
  id: number;
  title: string;
  dose: string | null;
  recurrence: string;
  weekdays: number[];
  times: string[];
  notes: string | null;
}

export interface Medications {
  items: RecordMedication[];
  profileMedications: string[] | null;
}

export interface Allergies {
  items: string[] | null;
  answered: boolean;
}

export interface CycleSummary {
  basedOn: number;
  medianCycle: number | null;
  variation: number | null;
  medianPeriod: number | null;
  regularity: string;
  lastPeriodStart: string | null;
  topSymptoms: { key: string; label: string; days: number }[];
}

export interface ValueSummary {
  avg: number;
  min: number;
  max: number;
  readings: number;
  inTargetPercent: number | null;
}

export interface VitalsSummary {
  days: number;
  bloodPressure: { systolic: number; diastolic: number; readings: number; tone: string } | null;
  heartRate: ValueSummary | null;
  glucoseFasting: ValueSummary | null;
  glucoseAfterMeal: ValueSummary | null;
  glucoseOther: ValueSummary | null;
}

export interface PregnancyEntry {
  id: number | null;
  source: 'tracked' | 'manual';
  outcome: PregnancyOutcome;
  date: string | null;
  babyCount: number | null;
  editable: boolean;
}

export interface Pregnancies {
  pregnanciesCount: number;
  birthsCount: number;
  items: PregnancyEntry[];
}

export interface RecordCheckup {
  id: number;
  title: string;
  doneOn: string;
  result: string;
}

export interface RecordLab {
  id: number;
  title: string;
  date: string;
  markerCount: number;
  attentionCount: number;
  allNormal: boolean;
  attention: { name: string; stateLabel: string }[];
}

/** A section as the screen reads it: `editable` / `empty` from the API plus its parsed body. */
export interface Section<T> {
  editable: boolean;
  empty: boolean;
  data: T;
}

export interface HealthRecord {
  date: string;
  updatedAt: string | null;
  person: RecordPerson;
  basics: Section<Basics>;
  conditions: Section<Conditions>;
  medications: Section<Medications>;
  allergies: Section<Allergies>;
  cycle: Section<CycleSummary>;
  vitals: Section<VitalsSummary>;
  pregnancies: Section<Pregnancies>;
  checkups: Section<{ items: RecordCheckup[] }>;
  labs: Section<{ items: RecordLab[] }>;
}

export interface BasicsInput {
  /** `undefined` keeps the stored value, `null` clears it. */
  bloodType?: BloodType | null;
  /** `[]` = «ندارم», `null` = clear. */
  allergies?: string[] | null;
}

export interface PregnancyInput {
  outcome: ManualOutcome;
  /** Gregorian `YYYY-MM-DD` (first day of the chosen year), or null when unknown. */
  endedOn: string | null;
  babyCount: number | null;
}

export interface ConditionsInput {
  chronicIllnesses: string[];
  gynConditions: string[];
}
