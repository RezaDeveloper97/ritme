import { z } from 'zod';

import { env } from '@/shared/config';

import {
  CATEGORY_KEYS,
  type CategoryKey,
  DOCUMENT_KINDS,
  type DatingOffer,
  type Extraction,
  type ReadValue,
  type RecordCategories,
  type RecordDocument,
  type RecordExtras,
  RELATIVES,
  type StoredFile,
  TIMELINE_KINDS,
  type TimelinePage,
} from '../model/documents';
import { safeFileUrl } from '../model/documents-lib';

/* Boundary schemas of the record documents API (CB-REC-01/02, CB-CORE-05). Unknown enum values degrade, never throw. */

const nstr = z.string().nullable();
const kind = z.enum(DOCUMENT_KINDS);
const review = z.enum(['manual', 'pending', 'needs_review', 'confirmed', 'failed']);
const linkType = z.enum(['claim', 'pregnancy']);
const linkState = z.enum(['attached', 'waiting', 'applied']);

export const recordCategoriesSchema = z
  .object({
    categories: z.array(z.object({ key: z.string(), count: z.number() })),
    documents_count: z.number(),
    all_count: z.number(),
    needs_review_count: z.number(),
  })
  .transform((d): RecordCategories => {
    const counts = Object.fromEntries(CATEGORY_KEYS.map((k) => [k, 0])) as Record<CategoryKey, number>;
    for (const c of d.categories) if ((CATEGORY_KEYS as readonly string[]).includes(c.key)) counts[c.key as CategoryKey] = c.count;
    return { counts, documentsCount: d.documents_count, allCount: d.all_count, needsReviewCount: d.needs_review_count };
  });

export const recordExtrasSchema = z
  .object({
    allergies: z.array(z.string()).nullable(),
    allergies_on_emergency_card: z.boolean(),
    surgeries: z.array(z.object({ title: z.string(), date: nstr })).nullable(),
    family_history: z
      .array(z.object({ condition: z.string(), relative: z.enum(RELATIVES).nullable().catch(null) }))
      .nullable(),
  })
  .transform(
    (d): RecordExtras => ({
      allergies: d.allergies,
      allergiesOnEmergencyCard: d.allergies_on_emergency_card,
      surgeries: d.surgeries,
      familyHistory: d.family_history,
    }),
  );

const timelineItem = z.object({
  type: z.enum(['document', 'lab']),
  id: z.number(),
  kind: z.enum(['lab', ...DOCUMENT_KINDS] as const).catch('other'),
  title: nstr,
  date: z.string(),
  date_known: z.boolean(),
  ended_on: nstr,
  centre: nstr,
  doctor: nstr,
  file_count: z.number().nullable(),
  review_state: review.nullable().catch(null),
  lab: z
    .object({ category: z.string(), marker_count: z.number(), attention_count: z.number(), all_normal: z.boolean() })
    .nullable(),
  links: z.array(z.object({ type: linkType, state: linkState })).catch([]),
});

export const timelinePageSchema = z
  .object({
    kind: z.enum(TIMELINE_KINDS).catch('all'),
    months: z.array(
      z.object({
        key: z.string(),
        jalali_year: z.number(),
        jalali_month: z.number(),
        start: z.string(),
        end: z.string(),
        items: z.array(timelineItem),
      }),
    ),
    next_before: nstr,
  })
  .transform(
    (d): TimelinePage => ({
      kind: d.kind,
      nextBefore: d.next_before,
      months: d.months.map((m) => ({
        key: m.key,
        jalaliYear: m.jalali_year,
        jalaliMonth: m.jalali_month,
        start: m.start,
        end: m.end,
        items: m.items.map((i) => ({
          type: i.type,
          id: i.id,
          kind: i.kind,
          title: i.title,
          date: i.date,
          dateKnown: i.date_known,
          endedOn: i.ended_on,
          centre: i.centre,
          doctor: i.doctor,
          fileCount: i.file_count,
          reviewState: i.review_state,
          lab: i.lab
            ? {
                category: i.lab.category,
                markerCount: i.lab.marker_count,
                attentionCount: i.lab.attention_count,
                allNormal: i.lab.all_normal,
              }
            : null,
          links: i.links,
        })),
      })),
    }),
  );

const readValue = z.object({ value: z.union([z.string(), z.number(), z.boolean()]), confidence: z.number() });
/** `{key: {value, confidence}}`, dropping anything malformed. */
const readMap = z.record(z.string(), z.unknown()).transform((m) => {
  const out: Record<string, ReadValue> = {};
  for (const [k, v] of Object.entries(m)) {
    const p = readValue.safeParse(v);
    if (p.success) out[k] = p.data;
  }
  return out;
});
const plain = z.union([z.string(), z.number(), z.null()]);

const extraction = z
  .object({
    schema: kind.catch('other'),
    status: z.enum(['pending', 'done', 'failed']).catch('failed'),
    error_code: nstr.catch(null),
    fields: readMap.catch({}),
    items: z.array(readMap).catch([]),
    reviewed: z.record(z.string(), plain.catch(null)).nullable().catch(null),
    reviewed_items: z.array(z.record(z.string(), z.string().nullable().catch(null))).nullable().catch(null),
    dating: z.enum(['applied', 'dismissed']).nullable().catch(null),
  })
  .transform(
    (e): Extraction => ({
      schema: e.schema,
      status: e.status,
      errorCode: e.error_code,
      fields: e.fields,
      items: e.items,
      reviewed: e.reviewed,
      reviewedItems: e.reviewed_items,
      dating: e.dating,
    }),
  );

export const recordDocumentSchema = z
  .object({
    id: z.number(),
    kind: kind.catch('other'),
    title: nstr,
    date: nstr,
    ended_on: nstr,
    centre: nstr,
    doctor: nstr,
    note: nstr,
    review_state: review.catch('manual'),
    extracted: extraction.nullable().catch(null),
    files: z.array(
      z.object({ id: z.number(), mime: z.string(), size_bytes: z.number(), url: nstr, url_expires_at: nstr }),
    ),
    where_used: z
      .array(z.object({ type: linkType, target_id: z.number(), state: linkState, updated_at: nstr }))
      .catch([]),
  })
  .transform(
    (d): RecordDocument => ({
      id: d.id,
      kind: d.kind,
      title: d.title,
      date: d.date,
      endedOn: d.ended_on,
      centre: d.centre,
      doctor: d.doctor,
      note: d.note,
      reviewState: d.review_state,
      extracted: d.extracted,
      files: d.files.map((f) => ({
        id: f.id,
        mime: f.mime,
        sizeBytes: f.size_bytes,
        url: safeFileUrl(f.url, env.apiBaseUrl),
        urlExpiresAt: f.url_expires_at,
      })),
      whereUsed: d.where_used.map((w) => ({ type: w.type, targetId: w.target_id, state: w.state, updatedAt: w.updated_at })),
    }),
  );

export const datingOfferSchema = z
  .object({
    state: z.enum(['offered', 'applied', 'dismissed', 'unavailable']).catch('unavailable'),
    reason: nstr.catch(null),
    message: z.string().catch(''),
    scan_date: nstr,
    proposed: z
      .object({ ga_weeks: z.number(), ga_days: z.number(), due_date: z.string(), weeks_today: z.number(), days_today: z.number() })
      .nullable(),
    current: z.object({ source: z.string(), due_date: z.string() }).nullable(),
    difference_days: z.number().nullable(),
  })
  .transform(
    (d): DatingOffer => ({
      state: d.state,
      reason: d.reason,
      message: d.message,
      scanDate: d.scan_date,
      proposed: d.proposed
        ? {
            gaWeeks: d.proposed.ga_weeks,
            gaDays: d.proposed.ga_days,
            dueDate: d.proposed.due_date,
            weeksToday: d.proposed.weeks_today,
            daysToday: d.proposed.days_today,
          }
        : null,
      current: d.current ? { source: d.current.source, dueDate: d.current.due_date } : null,
      differenceDays: d.difference_days,
    }),
  );

export const reviewResultSchema = z.object({ document: recordDocumentSchema, dating: datingOfferSchema });

export const storedFileSchema = z
  .object({ id: z.number(), mime: z.string(), size_bytes: z.number() })
  .transform((f): StoredFile => ({ id: f.id, mime: f.mime, sizeBytes: f.size_bytes }));

/** `GET /consents/ai_documents` (B-N6-05 platform; same envelope as the lab consent). */
export const docConsentSchema = z
  .object({
    consent: z.object({
      code: z.string(),
      version: z.number().int(),
      title: z.string().nullable().catch(null),
      body: z.string().nullable().catch(null),
      points: z.array(z.string()).catch([]),
      granted: z.boolean().catch(false),
      needs_consent: z.boolean().catch(true),
    }),
  })
  .transform(({ consent: c }) => ({
    code: c.code,
    version: c.version,
    title: c.title,
    body: c.body,
    points: c.points,
    granted: c.granted,
    needsConsent: c.needs_consent,
  }));
export type DocConsent = z.output<typeof docConsentSchema>;
