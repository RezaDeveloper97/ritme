/*
 * Children (bloom B-N5-02 API, B-N5-05 screens). Mirrors `ChildProfile`,
 * `ChildCard`, `ChildHome`, `ChildrenList` of backend-go/api/openapi.yaml.
 * Child data is health data (CLAUDE.md §11): never logged or sent to analytics.
 */

export const CHILD_SEXES = ['girl', 'boy'] as const;
export type ChildSex = (typeof CHILD_SEXES)[number];

export const CHILD_DELIVERY_TYPES = ['vaginal', 'cesarean'] as const;
export type ChildDeliveryType = (typeof CHILD_DELIVERY_TYPES)[number];

/** Validation mirrors of `internal/children/service.go` (the API decides). */
export const MAX_CHILDREN = 10;
export const MAX_CHILD_NAME = 64;
export const MAX_CHILD_AGE_YEARS = 18;
export const BIRTH_RANGES = {
  weightKg: { min: 0.3, max: 7 },
  lengthCm: { min: 20, max: 70 },
  headCm: { min: 15, max: 50 },
} as const;
export type BirthField = keyof typeof BIRTH_RANGES;

export type ChildRole = 'owner' | 'shared';

export interface ChildAge {
  days: number;
  weeks: number;
  months: number;
  years: number;
  daysInMonth: number;
  /** Server copy in the request locale («۳ ماه و ۱۲ روز»). */
  label: string;
}

export interface ChildBirth {
  weightKg: number | null;
  lengthCm: number | null;
  headCm: number | null;
}

export type VaccineVisitStatus = 'done' | 'due' | 'overdue' | 'soon' | 'upcoming';

export interface VaccineVisit {
  code: string;
  ageMonths: number;
  /** «۴ ماهگی». */
  label: string;
  dueDate: string;
  daysLeft: number;
  status: VaccineVisitStatus | string;
  statusLabel: string | null;
  given: number;
  total: number;
  doseNames: string[];
}

export interface VaccineSummary {
  given: number;
  total: number;
  completedVisits: number;
  upToDate: boolean;
  complete: boolean;
  next: VaccineVisit | null;
}

export type GrowthStatus = 'normal' | 'check' | 'unknown';

export interface GrowthVerdict {
  status: GrowthStatus;
  label: string;
}

/** A child as every list/card payload starts (`ChildCard`). */
export interface Child {
  id: number;
  name: string;
  initial: string;
  birthDate: string;
  sex: ChildSex | null;
  sexLabel: string | null;
  deliveryType: ChildDeliveryType | null;
  deliveryTypeLabel: string | null;
  birth: ChildBirth;
  age: ChildAge;
  role: ChildRole;
  canEdit: boolean;
  /** The owner's name, only on a child shared with the caller. */
  ownerName: string | null;
  vaccines: VaccineSummary;
  growth: GrowthVerdict;
}

export interface ChildrenList {
  children: Child[];
  count: number;
  ownedCount: number;
  maxChildren: number;
  canAdd: boolean;
  sharingNote: string | null;
}

export interface IndicatorValue {
  value: number;
  percentile: number | null;
  inBand: boolean | null;
}

export interface ChildMeasurement {
  id: number | null;
  source: 'measurement' | 'birth' | string;
  measuredOn: string;
  ageLabel: string | null;
  weight: IndicatorValue | null;
  length: IndicatorValue | null;
  head: IndicatorValue | null;
}

export interface MilestoneSummary {
  bandMonths: number;
  label: string;
  checked: number;
  total: number;
}

export interface ThisWeek {
  weeks: number;
  months: number;
  body: string | null;
}

export interface LearnTipPreview {
  code: string;
  topic: string;
  topicLabel: string | null;
  title: string | null;
  minutes: number | null;
}

/** GET /children/{id} — the child home (`ChildHome`). `today` stays null until B-N5-03. */
export interface ChildHome extends Child {
  latest: ChildMeasurement | null;
  milestones: MilestoneSummary | null;
  thisWeek: ThisWeek | null;
  learn: { count: number; featured: LearnTipPreview | null };
  today: unknown;
}

/** POST / PUT body (camelCase); `null` clears an optional field. */
export interface ChildInput {
  name: string;
  birthDate: string;
  sex: ChildSex | null;
  deliveryType: ChildDeliveryType | null;
  birthWeightKg: number | null;
  birthLengthCm: number | null;
  birthHeadCm: number | null;
}

/** One child of the companion home `child` card (`/companion/home`, B-N5-02). */
export interface CompanionChild {
  id: number;
  name: string;
  initial: string;
  sex: ChildSex | null;
  ageLabel: string;
  nextVaccine: { label: string; dueDate: string; daysLeft: number; status: string } | null;
}
