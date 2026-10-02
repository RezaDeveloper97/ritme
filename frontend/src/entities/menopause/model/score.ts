/*
 * Monthly score + patterns (CB-MENO-02 API, CB-MENO-08 screen nbl_Meno_Score).
 * Shapes of `GET|POST /menopause/scores`, `GET /menopause/patterns` and the
 * catalog groups `meno_score_items` / `meno_score_bands` after parsing.
 * Health data (CLAUDE.md §11): never log any of these.
 */

export const MENOPAUSE_SCORE_DOMAINS = ['somatic', 'psychological', 'urogenital'] as const;
export type MenopauseScoreDomain = (typeof MENOPAUSE_SCORE_DOMAINS)[number];

/** One `meno_score_bands` band (inclusive totals). */
export interface MenopauseScoreBand {
  code: string;
  title: string | null;
  min: number;
  max: number;
}

export interface MenopauseDomainScore {
  code: string;
  score: number;
  max: number;
}

/** One month's questionnaire with its band, domains and delta. */
export interface MenopauseScoreEntry {
  /** Gregorian date of the Jalali month's first day. */
  month: string;
  total: number;
  max: number;
  band: MenopauseScoreBand | null;
  domains: MenopauseDomainScore[];
  answers: Record<string, number>;
  /** total − the previous questionnaire's total (negative = fewer symptoms). */
  delta: number | null;
}

/** «از وقتی هورمون‌درمانی را شروع کردی (مرداد)، امتیازت ۸ واحد کمتر شده». */
export interface MenopauseHrtEffect {
  startedOn: string;
  baselineTotal: number | null;
  latestTotal: number | null;
  /** latest − baseline; null when either is missing. */
  change: number | null;
}

export interface MenopauseScoreHistory {
  months: number;
  max: number;
  bands: MenopauseScoreBand[];
  latest: MenopauseScoreEntry | null;
  /** One point per Jalali month, oldest first; the last one is the current month. */
  trend: Array<{ month: string; total: number | null; band: string | null }>;
  items: MenopauseScoreEntry[];
  hrt: MenopauseHrtEffect | null;
}

/** One question of the monthly questionnaire (`meno_score_items`). */
export interface MenopauseScoreQuestion {
  code: string;
  title: string;
  body: string | null;
  domain: string;
  max: number;
}

/** POST /menopause/scores — `month` omitted = this month. */
export interface MenopauseScoreInput {
  answers: Record<string, number>;
}

export interface MenopausePattern {
  key: string;
  trigger: string | null;
  found: boolean;
  /** Localized sentence of a found pattern (needs clinical review). */
  text: string | null;
}

export interface MenopausePatterns {
  daysLogged: number;
  minDays: number;
  found: number;
  disclaimer: { title: string | null; body: string | null } | null;
  items: MenopausePattern[];
}
