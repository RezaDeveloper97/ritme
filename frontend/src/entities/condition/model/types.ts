/*
 * Condition programs (CB-COND-01 API, hub CB-COND-02): endometriosis pain diary, PMDD questionnaire, heavy-bleeding
 * (PBAC) chart and PCOS (enrolment only — it reads the existing logs). Health data (§11): never log any of it.
 */

/** Hub order of the API (`GET /conditions`). */
export const CONDITION_PROGRAMS = ['endo', 'pmdd', 'heavy_bleeding', 'pcos'] as const;
export type ConditionProgram = (typeof CONDITION_PROGRAMS)[number];

export interface ConditionEnrolment {
  code: ConditionProgram;
  enrolled: boolean;
  /** `YYYY-MM-DD`, null while not enrolled. */
  enrolledOn: string | null;
}

export interface ConditionsOverview {
  programs: ConditionEnrolment[];
}

/** A `condition_programs` / `condition_alerts` catalog row (admin copy in the request locale). */
export interface ConditionCatalogItem {
  code: string;
  title: string | null;
  body: string | null;
  /** `condition_programs` meta `logs` — the «ثبت می‌کنی» line. */
  logs: string | null;
}

export const CONDITION_CATALOG_GROUPS = ['condition_programs', 'condition_alerts'] as const;
export type ConditionCatalogGroup = (typeof CONDITION_CATALOG_GROUPS)[number];

export const CONDITION_NUDGE_KEYS = ['heavy_pain', 'heavy_bleeding'] as const;
export type ConditionNudgeKey = (typeof CONDITION_NUDGE_KEYS)[number];

/** One `GET /messages/nudges` suggestion (CB-COND-06b): computed per request, nothing stored. */
export interface ConditionNudge {
  key: ConditionNudgeKey;
  program: ConditionProgram;
  title: string;
  body: string;
  action: string;
  /** Validated in-app path of the suggested program screen, or null. */
  link: string | null;
  /** Null until the doctors directory is wired. */
  doctor: { action: string; link: string } | null;
  days: number;
}
