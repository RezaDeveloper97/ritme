import { z } from 'zod';

import {
  CHECKUP_CATEGORIES,
  CHECKUP_PERFORMERS,
  CHECKUP_RESULTS,
  CHECKUP_SECTIONS,
  CHECKUP_STATUSES,
  CHECKUP_TONES,
  type CheckupDetail,
  type CheckupFindingOption,
  type CheckupGuideStep,
  type CheckupHome,
  type CheckupItem,
  type CheckupList,
  type CheckupNextPreview,
  type CheckupRecord,
  type CheckupRecordPage,
  type CheckupRecordResult,
  type CheckupSummary,
} from '../model/types';

/**
 * Boundary parsers for `/api/v1/checkups/*` (CLAUDE.md §10 — zod at the
 * boundary). Written against docs/checkups/README.md before the Go endpoints
 * existed, so they are deliberately tolerant — a newer (or older) backend must
 * never crash the bundle:
 *
 * - an unknown enum value falls back to a calm default: status → `up_to_date`
 *   (no alarming styling for a state this bundle can't explain), tone →
 *   `neutral`, category → `custom`, result → `pending`, performer → `doctor`;
 *   an unknown section falls back to the item's category;
 * - one malformed row is dropped, not the whole list;
 * - ids may be numbers or numeric strings; dates may carry a time part;
 * - bilingual fields normally arrive localized (Accept-Language) — if a raw
 *   `{"fa":…,"en":…}` object slips through, its first non-empty text is shown
 *   rather than nothing.
 */

const idSchema = z.union([z.number(), z.string()]).pipe(z.coerce.number().int());

const nullableId = z
  .union([z.number(), z.string()])
  .nullish()
  .transform((v) => {
    const n = v == null || v === '' ? NaN : Number(v);
    return Number.isInteger(n) ? n : null;
  });

/** A localized string, or a bilingual object → its first non-empty text. */
function pickText(value: unknown): string | null {
  if (typeof value === 'string') return value.trim() ? value : null;
  if (typeof value === 'number') return String(value);
  if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
    for (const v of Object.values(value)) {
      if (typeof v === 'string' && v.trim()) return v;
    }
  }
  return null;
}

const text = z.unknown().transform(pickText);

/** `Y-m-d`, `Y-m-d H:i:s` or ISO → `Y-m-d`; anything else → null. */
const dateText = z.unknown().transform((v) => {
  if (typeof v !== 'string') return null;
  const m = /^(\d{4}-\d{2}-\d{2})/.exec(v.trim());
  return m ? m[1] : null;
});

const count = z.coerce.number().int().nonnegative().catch(0).default(0);

const nullableCount = z.unknown().transform((v) => {
  const n = typeof v === 'string' && v.trim() !== '' ? Number(v) : v;
  return typeof n === 'number' && Number.isFinite(n) ? Math.trunc(n) : null;
});

const bool = (fallback: boolean) =>
  z
    .unknown()
    .transform((v) => (v === undefined || v === null ? fallback : v === true || v === 1 || v === '1' || v === 'true'));

const categorySchema = z.enum(CHECKUP_CATEGORIES).catch('custom');
const statusSchema = z.enum(CHECKUP_STATUSES).catch('up_to_date');
const toneSchema = z.enum(CHECKUP_TONES).catch('neutral');
const resultSchema = z.enum(CHECKUP_RESULTS).catch('pending');
const performerSchema = z.enum(CHECKUP_PERFORMERS).catch('doctor');

/** Array whose malformed items are dropped one by one; a non-array → []. */
function tolerantArray<T>(item: z.ZodType<T, z.ZodTypeDef, unknown>) {
  return z.unknown().transform((raw): T[] => {
    let list: unknown = raw;
    if (typeof list === 'string') {
      try {
        list = JSON.parse(list) as unknown;
      } catch {
        return [];
      }
    }
    if (!Array.isArray(list)) return [];
    const out: T[] = [];
    for (const v of list) {
      const parsed = item.safeParse(v);
      if (parsed.success) out.push(parsed.data);
    }
    return out;
  });
}

// ── Summary ──────────────────────────────────────────────────────

const summaryRowSchema = z.object({
  total: count,
  up_to_date: count,
  due: count,
  overdue: count,
});

function toSummary(s: z.infer<typeof summaryRowSchema>): CheckupSummary {
  return { total: s.total, upToDate: s.up_to_date, due: s.due, overdue: s.overdue };
}

/** Summary from the items, for a response that omits it. */
function summarize(items: CheckupItem[]): CheckupSummary {
  const applicable = items.filter((i) => i.status !== 'not_yet');
  return {
    total: applicable.length,
    upToDate: applicable.filter((i) => i.status === 'up_to_date' || i.status === 'soon').length,
    due: applicable.filter((i) => i.status === 'due').length,
    overdue: applicable.filter((i) => i.status === 'overdue').length,
  };
}

export const checkupSummarySchema = summaryRowSchema.transform(toSummary);

// ── Item ─────────────────────────────────────────────────────────

const itemRowSchema = z.object({
  id: idSchema,
  key: text,
  title: z.unknown().transform(pickText).pipe(z.string()),
  subtitle: text,
  category: categorySchema.default('custom'),
  section: z.unknown(),
  status: statusSchema.default('up_to_date'),
  icon: text,
  tone: toneSchema.default('neutral'),
  interval_label: text,
  timing_label: text,
  last_done_on: dateText,
  next_due_on: dateText,
  next_due_label: text,
  is_custom: z.unknown(),
});

function toItem(r: z.infer<typeof itemRowSchema>): CheckupItem {
  const section = z.enum(CHECKUP_SECTIONS).safeParse(r.section);
  const isCustom =
    r.is_custom === undefined || r.is_custom === null
      ? r.category === 'custom'
      : r.is_custom === true || r.is_custom === 1 || r.is_custom === '1';
  return {
    id: r.id,
    key: r.key,
    title: r.title,
    subtitle: r.subtitle,
    category: r.category,
    section: section.success ? section.data : r.category,
    status: r.status,
    icon: r.icon,
    tone: r.tone,
    intervalLabel: r.interval_label,
    timingLabel: r.timing_label,
    lastDoneOn: r.last_done_on,
    nextDueOn: r.next_due_on,
    nextDueLabel: r.next_due_label,
    isCustom,
  };
}

export const checkupItemSchema = itemRowSchema.transform(toItem);

/** GET /checkups. */
export const checkupListSchema = z
  .object({
    age: nullableCount,
    summary: summaryRowSchema.nullish().catch(null),
    items: tolerantArray(checkupItemSchema),
  })
  .transform(
    (d): CheckupList => ({
      age: d.age,
      summary: d.summary ? toSummary(d.summary) : summarize(d.items),
      items: d.items,
    }),
  );

/** GET /checkups/home — `null`/empty data or `total = 0` → null (the card hides). */
export const checkupHomeSchema = z
  .object({
    summary: summaryRowSchema.nullish().catch(null),
    highlights: tolerantArray(checkupItemSchema),
  })
  .nullish()
  .transform((d): CheckupHome | null => {
    if (!d) return null;
    const summary = d.summary ? toSummary(d.summary) : summarize(d.highlights);
    if (summary.total === 0) return null;
    return { summary, highlights: d.highlights.slice(0, 2) };
  });

// ── Records ──────────────────────────────────────────────────────

const findingsSchema = tolerantArray(z.union([z.string(), z.number()]).transform(String));

const recordRowSchema = z.object({
  id: idSchema,
  checkup_type_id: nullableId,
  checkup_id: nullableId,
  checkup_title: text,
  title: text,
  done_on: dateText.pipe(z.string()),
  result: resultSchema.default('pending'),
  findings: findingsSchema,
  note: text,
  has_attachment: bool(false),
  next_due_on: dateText,
});

export const checkupRecordSchema = recordRowSchema.transform(
  (r): CheckupRecord => ({
    id: r.id,
    checkupTypeId: r.checkup_type_id ?? r.checkup_id ?? 0,
    checkupTitle: r.checkup_title ?? r.title,
    doneOn: r.done_on,
    result: r.result,
    findings: r.findings,
    note: r.note,
    hasAttachment: r.has_attachment,
    nextDueOn: r.next_due_on,
  }),
);

const pageMetaSchema = z
  .object({
    current_page: z.coerce.number().int().catch(1).default(1),
    last_page: z.coerce.number().int().catch(1).default(1),
    total: z.coerce.number().int().nonnegative().optional().catch(undefined),
  })
  .nullish()
  .catch(null);

/**
 * GET /checkups/records — a bare array, or `{items|records, pagination|meta}`
 * (the two pagination shapes the API already uses elsewhere).
 */
export const checkupRecordPageSchema = z
  .unknown()
  .transform((raw): CheckupRecordPage => {
    const obj = typeof raw === 'object' && raw !== null && !Array.isArray(raw) ? (raw as Record<string, unknown>) : null;
    const records = tolerantArray(checkupRecordSchema).parse(obj ? (obj.items ?? obj.records ?? obj.data) : raw);
    const meta = pageMetaSchema.parse(obj ? (obj.pagination ?? obj.meta) : null);
    return {
      records,
      page: meta?.current_page ?? 1,
      lastPage: meta?.last_page ?? 1,
      total: meta?.total ?? records.length,
    };
  });

// ── Detail ───────────────────────────────────────────────────────

/*
 * Step / option items: a plain string, a structured object (`{text}`,
 * `{title, body}`, `{key, label}`), or — if the server forgot to localize — a
 * bilingual object. Structured keys are read first so a `{title, body}` step is
 * never mistaken for a bilingual pair.
 */
const asObject = (v: unknown): Record<string, unknown> | null =>
  typeof v === 'object' && v !== null && !Array.isArray(v) ? (v as Record<string, unknown>) : null;

const prepStepSchema = z.unknown().transform((v, ctx) => {
  const o = asObject(v);
  const t = o && ('text' in o || 'title' in o) ? pickText(o.text ?? o.title) : pickText(v);
  if (!t) {
    ctx.addIssue({ code: z.ZodIssueCode.custom });
    return z.NEVER;
  }
  return t;
});

const guideStepSchema = z.unknown().transform((v, ctx): CheckupGuideStep => {
  const o = asObject(v);
  if (o && ('title' in o || 'body' in o || 'text' in o)) {
    const title = pickText(o.title);
    if (title) return { title, body: pickText(o.body ?? o.text) };
  } else {
    const plain = pickText(v);
    if (plain) return { title: plain, body: null };
  }
  ctx.addIssue({ code: z.ZodIssueCode.custom });
  return z.NEVER;
});

/** Keys that mean «چیزی متفاوت نبود» when the server doesn't flag `exclusive`. */
const NOTHING_KEYS = new Set(['none', 'nothing', 'no_change', 'normal']);

const findingOptionSchema = z.unknown().transform((v, ctx): CheckupFindingOption => {
  if (typeof v === 'string' && v.trim()) {
    return { key: v, label: v, exclusive: NOTHING_KEYS.has(v) };
  }
  const o = asObject(v);
  if (o) {
    const label = pickText(o.label ?? o.title ?? o.text);
    const key = typeof o.key === 'string' && o.key ? o.key : label;
    if (key && label) {
      return {
        key,
        label,
        exclusive: typeof o.exclusive === 'boolean' ? o.exclusive : NOTHING_KEYS.has(key),
      };
    }
  }
  ctx.addIssue({ code: z.ZodIssueCode.custom });
  return z.NEVER;
});

const settingsSchema = z
  .object({ enabled: bool(true), remind: bool(true) })
  .nullish()
  .catch(null)
  .transform((s) => s ?? { enabled: true, remind: true });

/** GET /checkups/{id}. */
export const checkupDetailSchema = itemRowSchema
  .extend({
    why: text,
    performed_by: performerSchema.default('doctor'),
    interval_months: nullableCount,
    cycle_day_from: nullableCount,
    cycle_day_to: nullableCount,
    prep_steps: tolerantArray(prepStepSchema),
    guide_steps: tolerantArray(guideStepSchema),
    finding_options: tolerantArray(findingOptionSchema),
    records: tolerantArray(checkupRecordSchema),
    settings: settingsSchema,
  })
  .transform(
    (r): CheckupDetail => ({
      ...toItem(r),
      why: r.why,
      performedBy: r.performed_by,
      intervalMonths: r.interval_months,
      cycleDayFrom: r.cycle_day_from,
      cycleDayTo: r.cycle_day_to,
      prepSteps: r.prep_steps,
      guideSteps: r.guide_steps,
      findingOptions: r.finding_options,
      records: [...r.records].sort((a, b) => b.doneOn.localeCompare(a.doneOn)).slice(0, 2),
      settings: r.settings,
    }),
  );

// ── Mutations & preview ──────────────────────────────────────────

/**
 * POST /checkups/{id}/records and PUT /checkups/records/{id}.
 *
 * The README says "201 + recomputed item". The on-device attachment needs the
 * new record's id, so this reads, in order: `{item, record}` (preferred —
 * T-M4-02 note), a bare item, or a bare record. Anything unreadable → nulls,
 * and the caller refetches.
 */
export const checkupRecordResultSchema = z.unknown().transform((raw): CheckupRecordResult => {
  if (typeof raw !== 'object' || raw === null) return { item: null, record: null };
  const o = raw as Record<string, unknown>;
  if ('item' in o || 'record' in o) {
    const item = checkupItemSchema.safeParse(o.item);
    const record = checkupRecordSchema.safeParse(o.record);
    return { item: item.success ? item.data : null, record: record.success ? record.data : null };
  }
  if ('done_on' in o) {
    const record = checkupRecordSchema.safeParse(o);
    return { item: null, record: record.success ? record.data : null };
  }
  const item = checkupItemSchema.safeParse(o);
  return { item: item.success ? item.data : null, record: null };
});

/** GET /checkups/preview-next. */
export const checkupNextPreviewSchema = z
  .object({
    next_due_on: dateText,
    next_due_label: text,
    interval_label: text,
    reminder_label: text,
  })
  .transform(
    (p): CheckupNextPreview => ({
      nextDueOn: p.next_due_on,
      nextDueLabel: p.next_due_label,
      intervalLabel: p.interval_label,
      reminderLabel: p.reminder_label,
    }),
  );
