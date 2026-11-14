import { z } from 'zod';

import {
  ACCESS_LEVELS,
  COMPANION_ACTIONS,
  COMPANION_GOALS,
  COMPANION_SECTIONS,
  COMPANION_SIGNALS,
  type CompanionAction,
  type CompanionAuditEntry,
  type CompanionPreferences,
  type CompanionSignalScope,
  type CompanionGrants,
  type CompanionInvite,
  type CreatedCompanion,
  type OwnerCompanion,
} from '../model/types';

/*
 * `/api/v1/companions` payloads (backend-go/internal/companion/handlers.go,
 * openapi `CompanionOwnerLink` / `CompanionInvite` / `CompanionAuditEntry`),
 * validated at the boundary (CLAUDE.md §10). An unknown section or level from a
 * newer server is dropped / read as `none` — the most private reading.
 */

const level = z.enum(ACCESS_LEVELS).catch('none');

const grantsSchema = z
  .record(z.string(), z.unknown())
  .transform((raw): CompanionGrants => {
    const out = {} as CompanionGrants;
    for (const s of COMPANION_SECTIONS) out[s] = level.parse(raw[s]);
    return out;
  });

/*
 * v2 preferences (goal / signals / action_prefs). Lenient on purpose: a Laravel
 * production server or an older Go build sends none of them, and the most
 * private reading (nothing shared, no prefs) is the default.
 */
const goalSchema = z.enum(COMPANION_GOALS).catch('unknown');
export const signalScopeSchema = z
  .record(z.string(), z.unknown())
  .nullable()
  .catch(null)
  .optional()
  .transform((raw): CompanionSignalScope => {
    const out = {} as CompanionSignalScope;
    for (const s of COMPANION_SIGNALS) out[s] = raw?.[s] === true;
    return out;
  });
export const actionPrefsSchema = z
  .array(z.unknown())
  .nullable()
  .catch(null)
  .optional()
  .transform((raw): CompanionAction[] =>
    (raw ?? []).filter((a): a is CompanionAction => (COMPANION_ACTIONS as readonly unknown[]).includes(a)),
  );
export const preferencesSchema = z
  .object({ goal: z.unknown().optional(), signals: signalScopeSchema, action_prefs: actionPrefsSchema })
  .transform(
    (p): CompanionPreferences => ({ goal: goalSchema.parse(p.goal), signals: p.signals, actionPrefs: p.action_prefs }),
  );

const ownerLinkShape = z.object({
  goal: z.unknown().optional(),
  signals: signalScopeSchema,
  action_prefs: actionPrefsSchema,
  id: z.number(),
  type: z.enum(['partner', 'spouse']),
  status: z.enum(['invited', 'active']),
  display_name: z.string().nullable().optional(),
  name: z.string().nullable().optional(),
  invited_at: z.string().nullable().optional(),
  accepted_at: z.string().nullable().optional(),
  grants: grantsSchema,
  invite: z
    .object({ expires_at: z.string(), phone: z.string().nullable().optional() })
    .nullable()
    .optional(),
  family: z
    .object({ id: z.number(), shared_child_ids: z.array(z.number()).nullable().optional() })
    .nullable()
    .optional(),
});

export const ownerCompanionSchema = ownerLinkShape.transform(
  (l): OwnerCompanion => ({
    id: l.id,
    type: l.type,
    status: l.status,
    displayName: l.display_name ?? null,
    name: l.name ?? null,
    invitedAt: l.invited_at ?? null,
    acceptedAt: l.accepted_at ?? null,
    grants: l.grants,
    invite: l.invite ? { expiresAt: l.invite.expires_at, phone: l.invite.phone ?? null } : null,
    family: l.family ? { id: l.family.id, sharedChildIds: l.family.shared_child_ids ?? [] } : null,
    preferences: { goal: goalSchema.parse(l.goal), signals: l.signals, actionPrefs: l.action_prefs },
  }),
);

/**
 * GET /companions — partner / spouse rows. A row this client can't show (a
 * teen's `parent` link, CB-TEEN-01, managed on `/teen/parent`; a newer type) is
 * skipped instead of failing the whole list.
 */
export const ownerCompanionListSchema = z.array(z.unknown()).transform((rows) =>
  rows.flatMap((row) => {
    const parsed = ownerCompanionSchema.safeParse(row);
    return parsed.success ? [parsed.data] : [];
  }),
);

export const inviteSchema = z
  .object({
    code: z.string(),
    expires_at: z.string(),
    phone: z.string().nullable().optional(),
    sms_sent: z.boolean(),
  })
  .transform(
    (i): CompanionInvite => ({ code: i.code, expiresAt: i.expires_at, phone: i.phone ?? null, smsSent: i.sms_sent }),
  );

export const createdCompanionSchema = z
  .object({ companion: ownerCompanionSchema, invite: inviteSchema })
  .transform((v): CreatedCompanion => v);

const auditEntrySchema = z
  .object({
    id: z.number(),
    action: z.enum(['read', 'write', 'invited', 'accepted', 'revoked', 'grants_changed']),
    section: z.enum(COMPANION_SECTIONS).nullable().catch(null),
    companion_id: z.number().nullable(),
    actor: z.object({ id: z.number(), name: z.string().nullable(), is_me: z.boolean() }).nullable(),
    at: z.string(),
  })
  .transform(
    (e): CompanionAuditEntry => ({
      id: e.id,
      action: e.action,
      section: e.section,
      companionId: e.companion_id,
      actor: e.actor ? { id: e.actor.id, name: e.actor.name, isMe: e.actor.is_me } : null,
      at: e.at,
    }),
  );

/** GET /companions/audit — an entry a newer server shapes differently is skipped, not fatal. */
export const auditSchema = z.array(z.unknown()).transform((rows) =>
  rows.flatMap((row) => {
    const parsed = auditEntrySchema.safeParse(row);
    return parsed.success ? [parsed.data] : [];
  }),
);
