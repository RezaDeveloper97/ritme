/*
 * Teen mode (CB-TEEN-01 API, CB-TEEN-02 screens). Shapes of `/api/v1/teen/*`
 * in camelCase. The audience is a minor: nothing here is ever logged or sent
 * anywhere but the API (CLAUDE.md §11).
 */

export const TEEN_AGE_BANDS = ['10_12', '13_15', '16_17'] as const;
export type TeenAgeBand = (typeof TEEN_AGE_BANDS)[number];

export const TEEN_MENARCHE = ['not_yet', 'under_1y', 'over_1y'] as const;
export type TeenMenarche = (typeof TEEN_MENARCHE)[number];

export const TEEN_PERIOD_WEEKS = ['this_week', 'next_week', 'later', 'unknown'] as const;
export type TeenPeriodWeek = (typeof TEEN_PERIOD_WEEKS)[number];

export type TeenGrantLevel = 'none' | 'view';

export interface TeenGrants {
  teenPeriodWeek: TeenGrantLevel;
  teenKit: TeenGrantLevel;
  teenNotes: TeenGrantLevel;
}

export interface TeenProfile {
  ageBand: TeenAgeBand;
  menarche: TeenMenarche;
  parentNote: string | null;
}

/** Commercial surfaces the account may see — all false in teen mode. */
export interface TeenAllows {
  shop: boolean;
  banners: boolean;
  ads: boolean;
  plusUpsell: boolean;
}

export interface TeenProfileState {
  profile: TeenProfile | null;
  needsOnboarding: boolean;
  isTeenMode: boolean;
  allows: TeenAllows;
}

/** A catalog row (teen_signs / teen_faq): admin-editable copy, flagged for clinical review. */
export interface TeenCatalogItem {
  code: string;
  title: string | null;
  body: string | null;
  /** `caution` on the «talk to someone» estimate; `null` otherwise. */
  severity: string | null;
}

export interface TeenKitItem {
  code: string;
  title: string | null;
  checked: boolean;
}

export interface TeenKit {
  items: TeenKitItem[];
  checkedCount: number;
  total: number;
  ready: boolean;
}

export interface TeenParentLink {
  id: number;
  status: 'invited' | 'active';
  displayName: string | null;
  grants: TeenGrants;
}

/** The teen sections a parent link can see, in the board's order (Teen_Parent). */
export const TEEN_SHARE_KEYS = ['teenPeriodWeek', 'teenKit', 'teenNotes'] as const;
export type TeenShareKey = (typeof TEEN_SHARE_KEYS)[number];

/** Nothing shared — the default of a new parent invite (most private). */
export const NO_TEEN_GRANTS: TeenGrants = { teenPeriodWeek: 'none', teenKit: 'none', teenNotes: 'none' };

export interface TeenParentView {
  nextPeriodWeek: TeenPeriodWeek | null;
  kitReady: boolean | null;
  note: string | null;
}

export interface TeenToday extends TeenProfileState {
  readiness: TeenCatalogItem | null;
  signs: TeenCatalogItem[];
  talkNote: TeenCatalogItem | null;
  kit: TeenKit;
  faq: TeenCatalogItem[];
  parentPreview: TeenParentView;
  parentLinks: TeenParentLink[];
}

export interface TeenProfileInput {
  ageBand: TeenAgeBand;
  menarche: TeenMenarche;
}

/** One read-only card of GET /teen/linked (the parent's side). Ungranted parts are null. */
export interface TeenParentCard extends TeenParentView {
  linkId: number;
  teenName: string | null;
  grants: TeenGrants;
}

/** The one-time invite of POST /companions {type: parent} / renew. Keep in component state only. */
export interface TeenParentInvite {
  code: string;
  expiresAt: string;
  /** Masked bound number, e.g. 0912****567. */
  phone: string | null;
  smsSent: boolean;
}

export interface TeenParentInviteInput {
  phone: string;
  displayName?: string;
  grants: TeenGrants;
}
