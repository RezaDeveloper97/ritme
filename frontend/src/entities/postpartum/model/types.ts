/*
 * Postpartum mode (B-N5-01 API, B-N5-04 screens): `/api/v1/postpartum*`, Go only.
 * Health data (CLAUDE.md §11): never log a payload, a response or an EPDS answer.
 */

export const DELIVERY_TYPES = ['vaginal', 'cesarean'] as const;
export type DeliveryType = (typeof DELIVERY_TYPES)[number];

/** Upper bounds of the activation / recovery validators (backend `service.go`, `recovery.go`). */
export const MAX_BABY_COUNT = 4;
export const MAX_BIRTH_AGE_DAYS = 365;
export const MAX_FEEDS = 30;
export const MAX_SLEEP_HOURS = 24;

export const LOCHIA_AMOUNTS = ['none', 'spotting', 'light', 'medium', 'heavy'] as const;
export type LochiaAmount = (typeof LOCHIA_AMOUNTS)[number];
export const LOCHIA_COLORS = ['red', 'pink_brown', 'yellow_white'] as const;
export type LochiaColor = (typeof LOCHIA_COLORS)[number];
export const PAIN_LEVELS = ['none', 'mild', 'moderate', 'severe'] as const;
export type PainLevel = (typeof PAIN_LEVELS)[number];
export const PAIN_LOCATIONS = ['stitches', 'abdomen', 'breast', 'back', 'head'] as const;
export type PainLocation = (typeof PAIN_LOCATIONS)[number];
export const BREAST_SYMPTOMS = ['engorgement', 'nipple_pain', 'redness'] as const;
export type BreastSymptom = (typeof BREAST_SYMPTOMS)[number];

export const EPDS_KINDS = ['short', 'full'] as const;
export type EpdsKind = (typeof EPDS_KINDS)[number];

export interface PostpartumProfile {
  birthDate: string;
  deliveryType: DeliveryType | null;
  deliveryTypeLabel: string | null;
  babyCount: number;
  source: string;
}

export interface PostpartumStatus {
  daysSinceBirth: number;
  /** Completed weeks + the remaining days («۲ هفته و ۳ روز»). */
  weeks: number;
  days: number;
  /** 1-based week since the birth (week 3 while `weeks` = 2) — the headline uses `weeks` (QUESTIONS #97). */
  week: number;
  phase: string;
  phaseLabel: string | null;
  puerperiumDays: number;
  puerperiumDaysLeft: number;
  /** 0–1 of the puerperium. */
  progress: number;
}

export type AlertAction =
  | { type: 'call'; number: string; label: string | null }
  | { type: 'open_check'; kind: EpdsKind; label: string | null };

export interface PostpartumAlert {
  key: string;
  level: string;
  title: string | null;
  body: string | null;
  actionLabel: string | null;
  action: AlertAction | null;
}

export interface EpdsCheck {
  id: number;
  kind: EpdsKind;
  takenOn: string;
  week: number | null;
  total: number;
  max: number;
  band: string;
  bandLabel: string | null;
  urgent: boolean;
}

export interface PostpartumCheckin {
  due: EpdsKind | null;
  nextDueOn: string | null;
  last: EpdsCheck | null;
}

export interface PostpartumRecovery {
  date: string;
  lochiaAmount: LochiaAmount | null;
  lochiaColor: LochiaColor | null;
  painLevel: PainLevel | null;
  painLocations: PainLocation[];
  /** `null` = not logged, `[]` = normal. */
  breasts: BreastSymptom[] | null;
  feedsCount: number | null;
  sleepHours: number | null;
  alerts: PostpartumAlert[];
}

export interface PostpartumText {
  title: string | null;
  body: string | null;
}

export interface PostpartumWeekTip extends PostpartumText {
  week: number;
}

export interface PostpartumOverview {
  active: boolean;
  mode: string;
  /** Postpartum mode is on but the birth was never entered (mode switched from Me) → the activation form. */
  setupRequired: boolean;
  profile: PostpartumProfile | null;
  status: PostpartumStatus | null;
  checkin: PostpartumCheckin | null;
  today: PostpartumRecovery | null;
  alerts: PostpartumAlert[];
  weekTip: PostpartumWeekTip | null;
  callWhen: PostpartumText | null;
}

export interface ActivatePostpartumInput {
  birthDate: string;
  deliveryType: DeliveryType | null;
  babyCount: number;
}

/** Partial PUT body: keys left out stay, `null` clears (B-N5-01). */
export interface RecoveryUpdate {
  lochiaAmount?: LochiaAmount | null;
  lochiaColor?: LochiaColor | null;
  painLevel?: PainLevel | null;
  painLocations?: PainLocation[] | null;
  breasts?: BreastSymptom[] | null;
  feedsCount?: number | null;
  sleepHours?: number | null;
}

export interface EpdsOption {
  score: number;
  label: string;
}

export interface EpdsItem {
  code: string;
  number: number;
  text: string;
  options: EpdsOption[];
}

export interface EpdsQuestionnaire {
  kind: EpdsKind;
  intro: string | null;
  disclaimer: string | null;
  max: number;
  items: EpdsItem[];
}

export interface SafetyMessage {
  level: 'urgent' | 'advice' | 'follow_up' | string;
  title: string | null;
  body: string | null;
  actions: AlertAction[];
}

export interface EpdsResult {
  check: EpdsCheck;
  safety: SafetyMessage | null;
  followUp: EpdsKind | null;
  checkin: PostpartumCheckin | null;
}
