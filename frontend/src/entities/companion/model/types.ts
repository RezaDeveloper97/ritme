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
}
