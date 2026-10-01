/**
 * Global search (CB-NAV-01 API, CB-NAV-02 screen). Shapes mirror
 * `GET /api/v1/search` — backend-go/api/openapi.yaml tag Search and the
 * goldens in backend-go/contract/golden/search/.
 */

/** What the query runs over; `all` = every group, 3 items each. The shop is never searched here. */
export const SEARCH_SCOPES = ['all', 'mine', 'education', 'services', 'programs'] as const;
export type SearchScope = (typeof SEARCH_SCOPES)[number];

/** Result groups, in the board's order. */
export const SEARCH_GROUP_KEYS = ['mine', 'programs', 'education', 'services'] as const;
export type SearchGroupKey = (typeof SEARCH_GROUP_KEYS)[number];

export const SEARCH_HIT_TYPES = [
  'log_insight',
  'log_analysis',
  'reminder',
  'program',
  'article',
  'service',
  'checkup',
] as const;
export type SearchHitType = (typeof SEARCH_HIT_TYPES)[number];

/** Shortest query the API accepts (it answers 422 below). */
export const SEARCH_MIN_LENGTH = 2;
/** Longest query the API accepts. */
export const SEARCH_MAX_LENGTH = 100;

/** Peak 1–10 score of a scored log item inside the insight window. */
export interface SearchPeak {
  score: number;
  cycleDay: number | null;
}

/** The fields of `meta` the screen reads; everything else is ignored. */
export interface SearchHitMeta {
  /** log_insight / log_analysis: taxonomy category (`bleeding`, `pain`, …); article / checkup: their category. */
  category: string | null;
  /** log_insight: days with the item in the window. */
  days: number | null;
  /** log_insight: `cycle` = the current cycle, `days` = the last 30 days. */
  windowKind: 'cycle' | 'days' | null;
  /** log_insight: length of a `days` window, in days. */
  windowDays: number | null;
  peak: SearchPeak | null;
  /** reminder: medication | appointment; article: article. */
  kind: string | null;
  readTimeMinutes: number | null;
}

export interface SearchHit {
  type: SearchHitType;
  id: string;
  title: string;
  subtitle: string | null;
  /** In-app route without the locale prefix — may name a screen that isn't built yet (see `searchTarget`). */
  route: string;
  meta: SearchHitMeta;
}

export interface SearchGroup {
  key: SearchGroupKey;
  /** Full match count; `items` holds at most `limit`. */
  total: number;
  items: SearchHit[];
}

export interface SearchResults {
  query: string;
  scope: SearchScope;
  groups: SearchGroup[];
}
