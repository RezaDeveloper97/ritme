/*
 * Menopause mode (CB-MENO-02 API, CB-MENO-05 screens). Shapes of
 * `/api/v1/menopause/*` and `/api/v1/messages/menopause` after parsing
 * (camelCase). Health data (CLAUDE.md §11): never log any of these.
 */

/** The answer of the stage screen (nbl_Meno_Stage); `unsure` lets the API derive it. */
export const MENOPAUSE_STAGE_ANSWERS = ['peri', 'meno', 'post', 'unsure'] as const;
export type MenopauseStageAnswer = (typeof MENOPAUSE_STAGE_ANSWERS)[number];

/** The effective stage the API computes (docs/canvas-build/menopause.md §2). */
export type MenopauseStage = Exclude<MenopauseStageAnswer, 'unsure'>;

/** A `meno_*` catalog item (tips, alerts) in the request language — needs clinical review. */
export interface MenopauseCatalogItem {
  code: string;
  title: string | null;
  body: string | null;
  needsReview: boolean;
}

export interface MenopauseProfile {
  /** Effective stage; `null` = ask (stage screen). */
  stage: MenopauseStage | null;
  storedStage: MenopauseStageAnswer | null;
  /** Approximate last period, first of the month (`YYYY-MM-DD`). */
  lastPeriod: string | null;
  surgical: boolean | null;
  hrt: boolean | null;
  monthsWithoutPeriod: number | null;
  /** `meno` when she answered peri but her last period is 12+ months ago. */
  suggestedStage: 'meno' | null;
  needsStage: boolean;
  postMenopausal: boolean;
  tip: MenopauseCatalogItem | null;
}

/** Partial `PUT /menopause/profile` body; `null` clears a field. */
export interface MenopauseProfileUpdate {
  stage: MenopauseStageAnswer;
  lastPeriod: string | null;
  surgical: boolean | null;
  hrt: boolean | null;
}

export interface MenopauseFlash {
  id: number;
  /** ISO date-time with offset. */
  startedAt: string;
  running: boolean;
  durationS: number | null;
  /** The timer's length when the response was made (capped at an hour), or the duration once stopped. */
  elapsedS: number;
}

export interface MenopauseTrendPoint {
  /** Gregorian date of the Jalali month's first day. */
  month: string;
  total: number | null;
  band: string | null;
}

export interface MenopauseScoreSummary {
  total: number;
  max: number;
  band: { code: string; title: string | null } | null;
  /** total − the previous questionnaire's total (negative = fewer symptoms). */
  delta: number | null;
}

export interface MenopauseTreatment {
  id: number;
  kind: 'hrt' | 'supplement' | 'lifestyle';
  name: string;
  takenToday: boolean;
  daysTaken: number;
  days: number;
  reviewOn: string | null;
}

/** A checkup of her M4 plan; the raw item is kept for the checkup entity's parser. */
export type MenopauseCheckupRaw = Record<string, unknown>;

export interface MenopauseToday {
  date: string;
  profile: MenopauseProfile;
  hotFlashes: { count: number; nightCount: number; running: MenopauseFlash | null };
  nightSweats: { count: number };
  sleep: { hours: number } | null;
  score: { max: number; latest: MenopauseScoreSummary | null; trend: MenopauseTrendPoint[] };
  checkups: MenopauseCheckupRaw[];
  treatment: MenopauseTreatment[];
  bleeding: { alert: boolean; lastOn: string | null; alertItem: MenopauseCatalogItem | null };
}

export type MenopauseMessageKind = 'alert' | 'reminder' | 'tip';

export interface MenopauseMessage {
  key: string;
  kind: MenopauseMessageKind;
  priority: 'high' | 'medium' | 'low';
  title: string | null;
  body: string | null;
  action: string | null;
  /** In-app path; the screen only follows the ones whose route exists. */
  link: string | null;
}
