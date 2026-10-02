import { z } from 'zod';

import {
  TEEN_AGE_BANDS,
  TEEN_MENARCHE,
  TEEN_PERIOD_WEEKS,
  type TeenAllows,
  type TeenCatalogItem,
  type TeenGrantLevel,
  type TeenKit,
  type TeenParentLink,
  type TeenProfile,
  type TeenProfileState,
  type TeenToday,
} from '../model/types';

/*
 * Parsers of `/api/v1/teen/*` (CB-TEEN-01). Lenient on display fields so one
 * odd catalog row never blanks the home; a row that can't be shown is dropped.
 */

const text = z.string().nullable().catch(null);
const flag = z.boolean().catch(false);
const level = z.enum(['none', 'view']).catch('none');

const profileSchema = z
  .object({
    age_band: z.enum(TEEN_AGE_BANDS),
    menarche: z.enum(TEEN_MENARCHE),
    parent_note: text,
  })
  .transform((p): TeenProfile => ({ ageBand: p.age_band, menarche: p.menarche, parentNote: p.parent_note }));

// Unknown → the most restrictive reading: a commercial surface stays off.
const NO_COMMERCE = { shop: false, banners: false, ads: false, plus_upsell: false };
const allowsSchema = z
  .object({ shop: flag, banners: flag, ads: flag, plus_upsell: flag })
  .catch(NO_COMMERCE)
  .transform((a): TeenAllows => ({ shop: a.shop, banners: a.banners, ads: a.ads, plusUpsell: a.plus_upsell }));

const profileStateShape = {
  profile: profileSchema.nullable().catch(null),
  needs_onboarding: flag,
  is_teen_mode: flag,
  allows: allowsSchema,
};

interface RawProfileState {
  profile: TeenProfile | null;
  needs_onboarding: boolean;
  is_teen_mode: boolean;
  allows: TeenAllows;
}

function toProfileState(d: RawProfileState): TeenProfileState {
  return { profile: d.profile, needsOnboarding: d.needs_onboarding, isTeenMode: d.is_teen_mode, allows: d.allows };
}

/** `data` of GET|PUT /teen/profile and PUT /teen/parent-note. */
export const teenProfileStateSchema = z.object(profileStateShape).transform(toProfileState);

const catalogItemSchema = z
  .object({
    code: z.string(),
    title: text,
    body: text,
    meta: z
      .object({ severity: z.string().nullable().optional().catch(null) })
      .passthrough()
      .nullable()
      .catch(null),
  })
  .transform(
    (i): TeenCatalogItem => ({ code: i.code, title: i.title, body: i.body, severity: i.meta?.severity ?? null }),
  );

/** A list where an unparseable row is dropped, not the whole list. */
function rows<T>(schema: z.ZodType<T, z.ZodTypeDef, unknown>) {
  return z
    .array(z.unknown())
    .catch([])
    .transform((list) =>
      list.flatMap((raw) => {
        const parsed = schema.safeParse(raw);
        return parsed.success ? [parsed.data] : [];
      }),
    );
}

const kitItemSchema = z
  .object({ code: z.string(), title: text, checked: flag })
  .transform((i) => ({ code: i.code, title: i.title, checked: i.checked }));

/** `data` of PUT /teen/kit/{code} and `kit` of GET /teen/today. */
export const teenKitSchema = z
  .object({
    items: rows(kitItemSchema),
    checked_count: z.number().int().catch(0),
    total: z.number().int().catch(0),
    ready: flag,
  })
  .transform((k): TeenKit => ({ items: k.items, checkedCount: k.checked_count, total: k.total, ready: k.ready }));

const grantsSchema = z
  .object({ teen_period_week: level, teen_kit: level, teen_notes: level })
  .catch({ teen_period_week: 'none', teen_kit: 'none', teen_notes: 'none' })
  .transform((g) => ({
    teenPeriodWeek: g.teen_period_week as TeenGrantLevel,
    teenKit: g.teen_kit as TeenGrantLevel,
    teenNotes: g.teen_notes as TeenGrantLevel,
  }));

const parentLinkSchema = z
  .object({
    id: z.number().int(),
    status: z.enum(['invited', 'active']),
    display_name: text,
    grants: grantsSchema,
  })
  .transform((l): TeenParentLink => ({ id: l.id, status: l.status, displayName: l.display_name, grants: l.grants }));

const parentViewSchema = z
  .object({
    next_period_week: z.enum(TEEN_PERIOD_WEEKS).nullable().catch(null),
    kit_ready: z.boolean().nullable().catch(null),
    note: text,
  })
  .catch({ next_period_week: null, kit_ready: null, note: null })
  .transform((v) => ({ nextPeriodWeek: v.next_period_week, kitReady: v.kit_ready, note: v.note }));

/** `data` of GET /teen/today. */
export const teenTodaySchema = z
  .object({
    ...profileStateShape,
    readiness: catalogItemSchema.nullable().catch(null),
    signs: rows(catalogItemSchema),
    talk_note: catalogItemSchema.nullable().catch(null),
    kit: teenKitSchema,
    faq: rows(catalogItemSchema),
    parent_preview: parentViewSchema,
    parent_links: rows(parentLinkSchema),
  })
  .transform(
    (d): TeenToday => ({
      ...toProfileState(d),
      readiness: d.readiness,
      signs: d.signs,
      talkNote: d.talk_note,
      kit: d.kit,
      faq: d.faq,
      parentPreview: d.parent_preview,
      parentLinks: d.parent_links,
    }),
  );
