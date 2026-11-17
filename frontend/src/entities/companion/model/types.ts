/*
 * Companion «همدم» & family (B-N4-01/02, backend-go/internal/companion) — the
 * owner's side. Camel-cased from the `/api/v1/companions` payloads.
 */

/** partner = sees what she allows; spouse = the same + shared children / family. */
export const COMPANION_TYPES = ['partner', 'spouse'] as const;
export type CompanionType = (typeof COMPANION_TYPES)[number];

export type CompanionStatus = 'invited' | 'active';

/** The shareable sections, in the backend's (and the artboard's) order. */
export const COMPANION_SECTIONS = ['cycle', 'symptoms', 'meds', 'appointments', 'pregnancy'] as const;
export type CompanionSection = (typeof COMPANION_SECTIONS)[number];

/** Access per section, most private first. */
export const ACCESS_LEVELS = ['none', 'view', 'edit'] as const;
export type AccessLevel = (typeof ACCESS_LEVELS)[number];

export type CompanionGrants = Record<CompanionSection, AccessLevel>;

/*
 * «ریتمی همراه» v2 (companion-v2 design): what the owner wants from the
 * companionship, which coarse daily signals she shares (each with its own
 * consent, all off by default) and which kinds of support she prefers.
 */

/** «دوست داری همراهت بیشتر در چه چیزی کنارت باشد؟» — shapes copy and cards only, never a diagnosis. */
export const COMPANION_GOALS = ['cycle', 'support', 'low_energy', 'quality_time', 'info', 'unknown'] as const;
export type CompanionGoal = (typeof COMPANION_GOALS)[number];

/**
 * She may pick several goals, but only the ones that ask the companion for something combine. «info» and
 * «unknown» are the two "nothing in particular" answers: each stands alone, and `unknown` is the empty list on
 * the wire (the server applies the same rule — companion.NormalizeGoals).
 */
export const EXCLUSIVE_COMPANION_GOALS: readonly CompanionGoal[] = ['info', 'unknown'];

/** Coarse today-only flags («وقتی درد دارم» …); a bare yes/no, never the logged detail. */
export const COMPANION_SIGNALS = ['pain', 'mood', 'energy', 'sleep'] as const;
export type CompanionSignal = (typeof COMPANION_SIGNALS)[number];
export type CompanionSignalScope = Record<CompanionSignal, boolean>;

/** Support categories the owner likes («اقدام‌های ترجیحی»); the companion sees these first. */
export const COMPANION_ACTIONS = ['food', 'gift', 'time', 'calm', 'help'] as const;
export type CompanionAction = (typeof COMPANION_ACTIONS)[number];

export interface CompanionPreferences {
  /** In the order she picked them; never `unknown` (empty = not decided). */
  goals: CompanionGoal[];
  signals: CompanionSignalScope;
  actionPrefs: CompanionAction[];
}

export interface CompanionFamily {
  id: number;
  sharedChildIds: number[];
}

/** One row of GET /companions (Hamdam_List). */
export interface OwnerCompanion {
  id: number;
  type: CompanionType;
  status: CompanionStatus;
  /** What the owner typed when inviting («اسمش»). */
  displayName: string | null;
  /** displayName, else the companion account's name; null for an unnamed pending invite. */
  name: string | null;
  invitedAt: string | null;
  acceptedAt: string | null;
  grants: CompanionGrants;
  /** The open invite of a pending link (masked phone). */
  invite: { expiresAt: string; phone: string | null } | null;
  family: CompanionFamily | null;
  /** v2 preferences; a server without them answers the defaults (goal `unknown`, nothing shared, no prefs). */
  preferences: CompanionPreferences;
}

/**
 * The one-time invite returned by create / renew. `code` exists only in this
 * response — keep it in component state, never persist or log it.
 */
export interface CompanionInvite {
  code: string;
  expiresAt: string;
  /** Masked bound number, e.g. 0912****567. */
  phone: string | null;
  smsSent: boolean;
}

export interface CreatedCompanion {
  companion: OwnerCompanion;
  invite: CompanionInvite;
}

export type CompanionAuditAction = 'read' | 'write' | 'invited' | 'accepted' | 'revoked' | 'grants_changed';

export interface CompanionAuditEntry {
  id: number;
  action: CompanionAuditAction;
  section: CompanionSection | null;
  companionId: number | null;
  actor: { id: number; name: string | null; isMe: boolean } | null;
  at: string;
}

/** POST /companions body. */
export interface CreateCompanionInput {
  type: CompanionType;
  displayName?: string;
  phone?: string;
  grants: CompanionGrants;
  childIds?: number[];
  /** v2: goal, signal scope and preferred actions (optional — the server defaults them). */
  preferences?: Partial<CompanionPreferences>;
}

/** PUT /companions/{id}/preferences body — every field optional, only the sent ones change. */
export type UpdateCompanionPreferencesInput = Partial<CompanionPreferences>;
