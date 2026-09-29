import { z } from 'zod';

import {
  ALERT_LEVELS_V2,
  type AlertActionV2,
  type AlertContact,
  type AlertLegendRow,
  type AlertLevelV2,
  CARE_ITEM_KINDS,
  CARE_ITEM_STATES,
  type CalendarDayMarker,
  type CalendarVisit,
  type CareItem,
  type CareItemState,
  CONFIDENCE_LEVELS,
  type Confidence,
  DATING_SOURCES,
  DAY_SYMPTOMS,
  type DateRange,
  type DaySymptoms,
  type DueCard,
  type GestationalAgeV2,
  HIGHLIGHT_ICONS,
  HIGHLIGHT_TONES,
  type LastWeight,
  type Mood,
  MOODS,
  type NextVisit,
  type PregnancyAlertsV2,
  type PregnancyAlertV2,
  type PregnancyCalendar,
  type PregnancyDay,
  type PregnancyProgressV2,
  type PregnancyReport,
  type PregnancyToday,
  type PregnancyWeek,
  type DatingPreview,
  type ReportDay,
  type ReportWeight,
  SEVERITIES,
  type SetupResultCopy,
  type TrimesterSpan,
  V2_TERM_WEEKS,
  VISIT_STAGES,
  WATER_MAX,
  WATER_MIN,
  type WeekDetails,
  type WeekHighlight,
  type WeekRelation,
  type WeekSource,
  type WeekState,
  type WeekSummary,
  type WeekTask,
  type WeekTip,
} from '../model/v2-types';
import { defaultHighlightTone } from '../model/v2';

/**
 * Boundary parsers for `/api/v1/pregnancy/v2/*` (CLAUDE.md §10 — zod at the
 * boundary). The screens (T-M7-09…14) are written before the Go endpoints
 * (T-M7-02…05), so — like the fertility/checkup foundations — these parsers are
 * forgiving about what the README leaves open and strict about what the UI
 * needs:
 *
 * - an enum value this bundle doesn't know parses to `null` (or is dropped from
 *   a list/map) — a newer backend never crashes an older bundle;
 * - numbers may arrive as numbers or numeric strings (`decimal` → `"62.4"`);
 * - dates may carry a time part (`2026-09-22T00:00:00Z`) — only the day is kept;
 * - a list that is missing or not an array is `[]`; a row that fails to parse
 *   is skipped rather than failing the whole screen.
 *
 * Health data (§11): nothing here logs what it parses.
 */

// ── Primitives ─────────────────────────────────────────────────

/** Unknown / missing enum → null. */
const enumOrNull = <T extends readonly [string, ...string[]]>(values: T) =>
  z.unknown().transform((v): T[number] | null => {
    const parsed = z.enum(values).safeParse(v);
    return parsed.success ? parsed.data : null;
  });

/** A number (or numeric string) → number; anything else → null. */
const numberOrNull = z.unknown().transform((v): number | null => {
  if (v === null || v === undefined || v === '' || typeof v === 'boolean') return null;
  const n = typeof v === 'number' ? v : typeof v === 'string' ? Number(v) : Number.NaN;
  return Number.isFinite(n) ? n : null;
});

const intOrNull = numberOrNull.transform((n) => (n === null ? null : Math.round(n)));

/** A non-negative count; missing → 0. */
const count = intOrNull.transform((n) => (n === null || n < 0 ? 0 : n));

const clampPercent = numberOrNull.transform((n) =>
  n === null ? 0 : Math.min(100, Math.max(0, Math.round(n))),
);

/** Trimmed non-empty string → string; anything else → null. */
const textOrNull = z
  .unknown()
  .transform((v): string | null => (typeof v === 'string' && v.trim() !== '' ? v.trim() : null));

const bool = z.unknown().transform((v) => v === true || v === 1 || v === '1' || v === 'true');

/** `YYYY-MM-DD…` → `YYYY-MM-DD`; anything else → null. */
const dateOrNull = z.unknown().transform((v): string | null => {
  if (typeof v !== 'string') return null;
  const day = v.slice(0, 10);
  return /^\d{4}-\d{2}-\d{2}$/.test(day) ? day : null;
});

/** `07:30` / `07:30:00` → `07:30`. */
const timeOrNull = textOrNull.transform((v) => (v && /^\d{1,2}:\d{2}/.test(v) ? v.slice(0, 5) : null));

/** Parse each row with `row`; drop rows that fail. Not an array → `[]`. */
function listOf<T>(row: z.ZodType<T, z.ZodTypeDef, unknown>) {
  return z.unknown().transform((raw): T[] => {
    if (!Array.isArray(raw)) return [];
    const out: T[] = [];
    for (const item of raw) {
      const parsed = row.safeParse(item);
      if (parsed.success) out.push(parsed.data);
    }
    return out;
  });
}

/** Parse with `schema`; on failure (or null/undefined) → null. */
function orNull<T>(schema: z.ZodType<T, z.ZodTypeDef, unknown>) {
  return z.unknown().transform((raw): T | null => {
    if (raw === null || raw === undefined) return null;
    const parsed = schema.safeParse(raw);
    return parsed.success ? parsed.data : null;
  });
}

const obj = z.record(z.unknown());

// ── Shared pieces ──────────────────────────────────────────────

const rangeSchema = obj.transform((r, ctx): DateRange => {
  const from = dateOrNull.parse(r.from);
  const to = dateOrNull.parse(r.to);
  if (!from || !to) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'range needs from/to dates' });
    return z.NEVER;
  }
  return { from, to };
});
const rangeOrNull = orNull(rangeSchema);

/** `{level,label}` or a bare level string. */
const confidenceSchema = z.unknown().transform((raw): Confidence => {
  if (typeof raw === 'string') return { level: enumOrNull(CONFIDENCE_LEVELS).parse(raw), label: null };
  if (typeof raw === 'object' && raw !== null) {
    const r = raw as Record<string, unknown>;
    return { level: enumOrNull(CONFIDENCE_LEVELS).parse(r.level), label: textOrNull.parse(r.label) };
  }
  return { level: null, label: null };
});

/**
 * Gestational age: `{weeks, days}` object, or flat `weeks`/`days` on the parent
 * (both are accepted — {@link ageFrom}).
 */
function ageFrom(r: Record<string, unknown>): GestationalAgeV2 | null {
  const nested = typeof r.gestational_age === 'object' && r.gestational_age !== null
    ? (r.gestational_age as Record<string, unknown>)
    : typeof r.age === 'object' && r.age !== null
      ? (r.age as Record<string, unknown>)
      : r;
  const weeks = intOrNull.parse(nested.weeks);
  if (weeks === null) return null;
  const days = intOrNull.parse(nested.days) ?? 0;
  return { weeks, days: Math.min(6, Math.max(0, days)) };
}

const taskSchema = obj.transform((r, ctx): WeekTask => {
  const key = textOrNull.parse(r.key);
  const label = textOrNull.parse(r.text ?? r.title);
  if (!key || !label) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'task needs key + text' });
    return z.NEVER;
  }
  return { key, text: label, done: bool.parse(r.done) };
});

const tipSchema = obj.transform((r, ctx): WeekTip => {
  const title = textOrNull.parse(r.title);
  const body = textOrNull.parse(r.body ?? r.text);
  if (!title && !body) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'empty tip' });
    return z.NEVER;
  }
  return {
    week: intOrNull.parse(r.week),
    title: title ?? '',
    body: body ?? '',
    readMinutes: intOrNull.parse(r.read_minutes),
    link: textOrNull.parse(r.link ?? r.article_link),
  };
});

const relationSchema = enumOrNull(['past', 'current', 'future'] as const);

// ── POST /pregnancy/v2/dating-preview ──────────────────────────

function setupResultCopy(raw: unknown): SetupResultCopy {
  const c = (typeof raw === 'object' && raw !== null ? raw : {}) as Record<string, unknown>;
  return {
    lead: textOrNull.parse(c.lead),
    suffix: textOrNull.parse(c.suffix),
    dueLabel: textOrNull.parse(c.due_label),
    confidence: textOrNull.parse(c.confidence),
    primary: textOrNull.parse(c.primary),
    secondary: textOrNull.parse(c.secondary),
  };
}

export const datingPreviewSchema = obj.transform((r, ctx): DatingPreview => {
  const age = ageFrom(r);
  const dueDate = dateOrNull.parse(r.due_date ?? (r.due as Record<string, unknown> | undefined)?.date);
  if (!age || !dueDate) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'preview needs weeks + due_date' });
    return z.NEVER;
  }
  return {
    source: enumOrNull(DATING_SOURCES).parse(r.source),
    age,
    trimester: intOrNull.parse(r.trimester),
    dueDate,
    dueDateLabel: textOrNull.parse(r.due_date_label),
    range: rangeOrNull.parse(r.range ?? r.birth_range),
    rangeLabel: textOrNull.parse(r.range_label),
    uncertaintyDays: intOrNull.parse(r.uncertainty_days),
    confidence: confidenceSchema.parse(r.confidence ?? r.confidence_level),
    basis: textOrNull.parse(r.basis ?? r.basis_sentence),
    copy: setupResultCopy(r.copy),
  };
});

// ── GET /pregnancy/v2/today ────────────────────────────────────

const weekSummarySchema = obj.transform((r, ctx): WeekSummary => {
  const week = intOrNull.parse(r.week);
  if (week === null) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'summary needs week' });
    return z.NEVER;
  }
  const relation: WeekRelation =
    relationSchema.parse(r.relation) ?? (bool.parse(r.is_current) ? 'current' : 'future');
  return {
    week,
    trimester: intOrNull.parse(r.trimester),
    title: textOrNull.parse(r.title),
    sizeLine: textOrNull.parse(r.size_line ?? r.size),
    illustrationKey: textOrNull.parse(r.illustration_key),
    relation,
  };
});

const dueCardSchema = obj.transform((r, ctx): DueCard => {
  const date = dateOrNull.parse(r.date);
  if (!date) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'due card needs date' });
    return z.NEVER;
  }
  return {
    date,
    dateLabel: textOrNull.parse(r.date_label ?? r.label),
    daysLeft: intOrNull.parse(r.days_left),
    range: rangeOrNull.parse(r.range),
  };
});

const trimesterSpanSchema = obj.transform((r, ctx): TrimesterSpan => {
  const t = intOrNull.parse(r.trimester);
  if (t !== 1 && t !== 2 && t !== 3) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'trimester must be 1–3' });
    return z.NEVER;
  }
  return {
    trimester: t,
    startDate: dateOrNull.parse(r.start_date),
    startLabel: textOrNull.parse(r.start_label),
    percent: clampPercent.parse(r.percent),
  };
});

/** Progress; a missing `week` / `percent` falls back to the gestational age. */
function parseProgress(raw: unknown, fallbackWeek: number): PregnancyProgressV2 {
  const r = (typeof raw === 'object' && raw !== null ? raw : {}) as Record<string, unknown>;
  const week = intOrNull.parse(r.week) ?? fallbackWeek;
  const percent =
    r.percent === undefined || r.percent === null
      ? Math.min(100, Math.max(0, Math.round((week / V2_TERM_WEEKS) * 100)))
      : clampPercent.parse(r.percent);
  const trimesters = listOf(trimesterSpanSchema)
    .parse(r.trimesters)
    .sort((a, b) => a.trimester - b.trimester);
  return { week, percent, trimesters };
}

const nextVisitSchema = obj.transform((r, ctx): NextVisit => {
  const title = textOrNull.parse(r.title);
  if (!title) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'visit needs title' });
    return z.NEVER;
  }
  return {
    appointmentId: intOrNull.parse(r.appointment_id),
    careItemKey: textOrNull.parse(r.care_item_key),
    title,
    date: dateOrNull.parse(r.date),
    dateLabel: textOrNull.parse(r.date_label),
    time: timeOrNull.parse(r.time),
    week: intOrNull.parse(r.week),
    daysUntil: intOrNull.parse(r.days_until),
    stage: enumOrNull(VISIT_STAGES).parse(r.stage),
  };
});

export const pregnancyTodaySchema = obj.transform((r, ctx): PregnancyToday => {
  const age = ageFrom(r);
  const date = dateOrNull.parse(r.date);
  if (!age || !date) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'today needs date + weeks' });
    return z.NEVER;
  }
  return {
    date,
    age,
    trimester: intOrNull.parse(r.trimester),
    confidence: confidenceSchema.parse(r.confidence),
    uncertaintyDays: intOrNull.parse(r.uncertainty_days),
    carousel: listOf(weekSummarySchema).parse(r.carousel ?? r.weeks),
    due: orNull(dueCardSchema).parse(r.due),
    progress: parseProgress(r.progress, age.weeks),
    nextVisit: orNull(nextVisitSchema).parse(r.next_visit),
    tip: orNull(tipSchema).parse(r.tip),
    tasks: listOf(taskSchema).parse(r.tasks),
    unreadAlerts: count.parse(r.unread_alerts ?? r.alerts_unread),
  };
});

// ── GET /pregnancy/v2/weeks/{n} ────────────────────────────────

const highlightSchema = obj.transform((r, ctx): WeekHighlight => {
  const title = textOrNull.parse(r.title);
  const body = textOrNull.parse(r.body);
  if (!title && !body) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'empty highlight' });
    return z.NEVER;
  }
  const icon = enumOrNull(HIGHLIGHT_ICONS).parse(r.icon);
  return {
    icon,
    tone: enumOrNull(HIGHLIGHT_TONES).parse(r.tone) ?? defaultHighlightTone(icon),
    title: title ?? '',
    body: body ?? '',
  };
});

/** `sources` may be `[{title,url}]`, `["text"]` or one string. */
const sourcesSchema = z.unknown().transform((raw): WeekSource[] => {
  if (typeof raw === 'string') {
    const t = raw.trim();
    return t ? [{ title: t, url: null }] : [];
  }
  if (!Array.isArray(raw)) return [];
  return raw.flatMap((s): WeekSource[] => {
    if (typeof s === 'string') return s.trim() ? [{ title: s.trim(), url: null }] : [];
    if (typeof s === 'object' && s !== null) {
      const o = s as Record<string, unknown>;
      const url = textOrNull.parse(o.url);
      const title = textOrNull.parse(o.title) ?? url;
      return title ? [{ title, url }] : [];
    }
    return [];
  });
});

const stringList = z.unknown().transform((raw): string[] =>
  Array.isArray(raw) ? raw.map((s) => textOrNull.parse(s)).filter((s): s is string => s !== null) : [],
);

const EMPTY_DETAILS: WeekDetails = {
  sizeLabel: null,
  illustrationKey: null,
  length: null,
  weight: null,
  heartRate: null,
  headline: null,
  highlights: [],
  bodySymptoms: [],
  bodyText: null,
  warning: null,
  reviewerName: null,
  reviewedAt: null,
  sources: [],
};

/** A stat may be a number or a text range — keep it as display text. */
const statText = z.unknown().transform((v): string | null =>
  typeof v === 'number' && Number.isFinite(v) ? String(v) : textOrNull.parse(v),
);

const weekDetailsSchema = z.unknown().transform((raw): WeekDetails => {
  if (typeof raw !== 'object' || raw === null) return EMPTY_DETAILS;
  const r = raw as Record<string, unknown>;
  return {
    sizeLabel: textOrNull.parse(r.size_label),
    illustrationKey: textOrNull.parse(r.illustration_key),
    length: statText.parse(r.length_cm ?? r.length),
    weight: statText.parse(r.weight_g ?? r.weight),
    heartRate: statText.parse(r.heart_rate),
    headline: textOrNull.parse(r.headline),
    highlights: listOf(highlightSchema).parse(r.highlights),
    bodySymptoms: stringList.parse(r.body_symptoms),
    bodyText: textOrNull.parse(r.body_text),
    warning: textOrNull.parse(r.warning),
    reviewerName: textOrNull.parse(r.reviewer_name),
    reviewedAt: dateOrNull.parse(r.reviewed_at),
    sources: sourcesSchema.parse(r.sources),
  };
});

export const pregnancyWeekSchema = obj.transform((r, ctx): PregnancyWeek => {
  const week = intOrNull.parse(r.week ?? r.week_number);
  if (week === null) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'week needs week' });
    return z.NEVER;
  }
  // `tasks` may already carry `done`, or come as `{key,text}` + `done_task_keys`.
  const done = new Set(stringList.parse(r.done_task_keys));
  const details = weekDetailsSchema.parse(r.details ?? r);
  const rawTasks = Array.isArray(r.tasks)
    ? r.tasks
    : ((r.details as Record<string, unknown> | undefined)?.tasks ?? []);
  const tasks = listOf(taskSchema)
    .parse(rawTasks)
    .map((t) => ({ ...t, done: t.done || done.has(t.key) }));
  return {
    week,
    trimester: intOrNull.parse(r.trimester),
    relation: relationSchema.parse(r.relation) ?? 'future',
    range: rangeOrNull.parse(r.range ?? r.date_range),
    rangeLabel: textOrNull.parse(r.range_label),
    bookmarked: bool.parse(r.bookmarked),
    details,
    tasks,
    tip: orNull(tipSchema).parse(r.tip),
  };
});

export const weekStateSchema = obj.transform((r, ctx): WeekState => {
  const week = intOrNull.parse(r.week);
  if (week === null) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'state needs week' });
    return z.NEVER;
  }
  return { week, bookmarked: bool.parse(r.bookmarked), doneTaskKeys: stringList.parse(r.done_task_keys) };
});

// ── Alerts ─────────────────────────────────────────────────────

/** `{key,label}` or a bare key. */
const alertActionSchema = z.unknown().transform((raw, ctx): AlertActionV2 => {
  if (typeof raw === 'string' && raw.trim()) return { key: raw.trim(), label: null };
  if (typeof raw === 'object' && raw !== null) {
    const r = raw as Record<string, unknown>;
    const key = textOrNull.parse(r.key ?? r.action);
    if (key) return { key, label: textOrNull.parse(r.label) };
  }
  ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'action needs key' });
  return z.NEVER;
});

const contactSchema = z.unknown().transform((raw, ctx): AlertContact => {
  if (typeof raw === 'string' && raw.trim()) return { text: raw.trim(), phone: null };
  if (typeof raw === 'object' && raw !== null) {
    const r = raw as Record<string, unknown>;
    const t = textOrNull.parse(r.text ?? r.line);
    if (t) return { text: t, phone: textOrNull.parse(r.phone) };
  }
  ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'empty contact' });
  return z.NEVER;
});

/** Level from `level`, or from the v2 `level4` key the README puts in the payload. */
function levelOf(r: Record<string, unknown>): AlertLevelV2 | null {
  const level = enumOrNull(ALERT_LEVELS_V2);
  const payload = (typeof r.payload === 'object' && r.payload !== null ? r.payload : {}) as Record<
    string,
    unknown
  >;
  return level.parse(r.level) ?? level.parse(r.level4) ?? level.parse(payload.level4);
}

export const pregnancyAlertV2Schema = obj.transform((r, ctx): PregnancyAlertV2 => {
  const id = intOrNull.parse(r.id);
  const level = levelOf(r);
  const title = textOrNull.parse(r.title);
  // An alert whose level this bundle doesn't know can't be styled or ranked
  // honestly — drop it rather than guess (the list parser skips it).
  if (id === null || level === null || !title) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'alert needs id, level, title' });
    return z.NEVER;
  }
  const createdAt = textOrNull.parse(r.created_at);
  return {
    id,
    ruleKey: textOrNull.parse(r.rule_key ?? r.rule),
    level,
    title,
    whatWeSaw: textOrNull.parse(r.what_we_saw),
    howSure: textOrNull.parse(r.how_sure),
    advice: textOrNull.parse(r.advice),
    actions: listOf(alertActionSchema).parse(r.actions),
    contact: level === 'urgent' ? orNull(contactSchema).parse(r.contact) : null,
    createdAt,
    factDate: dateOrNull.parse(r.fact_date) ?? createdAt?.slice(0, 10) ?? null,
    dateLabel: textOrNull.parse(r.date_label),
    isRead: bool.parse(r.is_read ?? r.read),
    isAcked: bool.parse(r.is_acked ?? r.acked),
  };
});

const legendRowSchema = obj.transform((r, ctx): AlertLegendRow => {
  const level = enumOrNull(ALERT_LEVELS_V2).parse(r.level);
  if (!level) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'legend needs level' });
    return z.NEVER;
  }
  return { level, label: textOrNull.parse(r.label), text: textOrNull.parse(r.text ?? r.description) };
});

export const pregnancyAlertsV2Schema = z.unknown().transform((raw): PregnancyAlertsV2 => {
  // Accept a bare array of alerts as well as the documented object.
  const r = (Array.isArray(raw) ? { alerts: raw } : typeof raw === 'object' && raw !== null ? raw : {}) as Record<
    string,
    unknown
  >;
  const legend = listOf(legendRowSchema)
    .parse(r.legend)
    .sort((a, b) => ALERT_LEVELS_V2.indexOf(a.level) - ALERT_LEVELS_V2.indexOf(b.level));
  return {
    windowDays: intOrNull.parse(r.window_days) ?? 7,
    alerts: listOf(pregnancyAlertV2Schema).parse(r.alerts),
    legend,
  };
});

// ── Day log ────────────────────────────────────────────────────

const moodSchema = intOrNull.transform((n): Mood | null =>
  n !== null && (MOODS as readonly number[]).includes(n) ? (n as Mood) : null,
);

/**
 * `{symptom: severity}`; also accepts `[{key, severity}]`. Unknown symptoms and
 * severities are dropped; a symptom with a missing severity is `mild`.
 */
const symptomsSchema = z.unknown().transform((raw): DaySymptoms => {
  const out: DaySymptoms = {};
  const sev = enumOrNull(SEVERITIES);
  const put = (key: unknown, severity: unknown) => {
    const k = enumOrNull(DAY_SYMPTOMS).parse(key);
    if (!k || severity === null || severity === false) return;
    out[k] = sev.parse(severity) ?? 'mild';
  };
  if (Array.isArray(raw)) {
    for (const item of raw) {
      if (typeof item === 'string') put(item, 'mild');
      else if (typeof item === 'object' && item !== null) {
        const o = item as Record<string, unknown>;
        put(o.key ?? o.symptom, o.severity);
      }
    }
  } else if (typeof raw === 'object' && raw !== null) {
    for (const [k, v] of Object.entries(raw)) put(k, v);
  }
  return out;
});

const waterSchema = intOrNull.transform((n) =>
  n === null ? null : Math.min(WATER_MAX, Math.max(WATER_MIN, n)),
);

const weightSchema = numberOrNull.transform((n) =>
  n === null || n <= 0 ? null : Math.round(n * 10) / 10,
);

const lastWeightSchema = obj.transform((r, ctx): LastWeight => {
  const value = weightSchema.parse(r.value ?? r.weight);
  const date = dateOrNull.parse(r.date);
  if (value === null || !date) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'last weight needs value + date' });
    return z.NEVER;
  }
  return { value, date };
});

export const pregnancyDaySchema = obj.transform((r, ctx): PregnancyDay => {
  const date = dateOrNull.parse(r.date ?? r.log_date);
  if (!date) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'day needs date' });
    return z.NEVER;
  }
  return {
    date,
    week: intOrNull.parse(r.week ?? r.pregnancy_week),
    mood: moodSchema.parse(r.mood),
    symptoms: symptomsSchema.parse(r.symptoms),
    waterGlasses: waterSchema.parse(r.water_glasses),
    weight: weightSchema.parse(r.weight ?? r.weight_kg),
    visitNote: textOrNull.parse(r.visit_note),
    lastWeight: orNull(lastWeightSchema).parse(r.last_weight),
    alerts: listOf(pregnancyAlertV2Schema).parse(r.alerts),
  };
});

// ── Calendar ───────────────────────────────────────────────────

const dayMarkerSchema = obj.transform((r, ctx): CalendarDayMarker => {
  const date = dateOrNull.parse(r.date);
  if (!date) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'marker needs date' });
    return z.NEVER;
  }
  return {
    date,
    hasVisit: bool.parse(r.has_visit ?? r.visit),
    weekStart: intOrNull.parse(r.week_start),
    isToday: bool.parse(r.is_today ?? r.today),
  };
});

const calendarVisitSchema = obj.transform((r, ctx): CalendarVisit => {
  const title = textOrNull.parse(r.title);
  const date = dateOrNull.parse(r.date);
  if (!title || !date) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'visit needs title + date' });
    return z.NEVER;
  }
  return {
    appointmentId: intOrNull.parse(r.appointment_id ?? r.id),
    careItemKey: textOrNull.parse(r.care_item_key),
    title,
    date,
    dateLabel: textOrNull.parse(r.date_label),
    time: timeOrNull.parse(r.time),
    week: intOrNull.parse(r.week),
    weekLabel: textOrNull.parse(r.week_label),
    ageLabel: textOrNull.parse(r.age_label),
    stage: enumOrNull(VISIT_STAGES).parse(r.stage),
    prep: textOrNull.parse(r.prep),
    doctor: textOrNull.parse(r.doctor ?? r.doctor_name),
    place: textOrNull.parse(r.place ?? r.location),
    remindBefore: textOrNull.parse(r.remind_before),
    resultNote: textOrNull.parse(r.result_note),
    daysUntil: intOrNull.parse(r.days_until),
  };
});

const careItemSchema = obj.transform((r, ctx): CareItem => {
  const key = textOrNull.parse(r.key);
  const title = textOrNull.parse(r.title);
  if (!key || !title) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'care item needs key + title' });
    return z.NEVER;
  }
  const state: CareItemState = enumOrNull(CARE_ITEM_STATES).parse(r.state) ?? 'to_book';
  return {
    key,
    title,
    kind: enumOrNull(CARE_ITEM_KINDS).parse(r.kind),
    weekFrom: intOrNull.parse(r.week_from),
    weekTo: intOrNull.parse(r.week_to),
    window: rangeOrNull.parse(r.window),
    state,
    date: dateOrNull.parse(r.date),
    dateLabel: textOrNull.parse(r.date_label),
    suggestedDate: dateOrNull.parse(r.suggested_date),
    appointmentId: intOrNull.parse(r.appointment_id),
    prep: textOrNull.parse(r.prep),
  };
});

const weekRangeSchema = obj.transform((r, ctx) => {
  const from = intOrNull.parse(r.from);
  const to = intOrNull.parse(r.to);
  if (from === null || to === null) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'week range needs from/to' });
    return z.NEVER;
  }
  return { from, to };
});

export const pregnancyCalendarSchema = obj.transform((r, ctx): PregnancyCalendar => {
  const month = textOrNull.parse(r.month);
  if (!month || !/^\d{4}-\d{2}$/.test(month.slice(0, 7))) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'calendar needs month' });
    return z.NEVER;
  }
  return {
    month: month.slice(0, 7),
    monthLabel: textOrNull.parse(r.month_label),
    weekRange: orNull(weekRangeSchema).parse(r.week_range),
    days: listOf(dayMarkerSchema).parse(r.days),
    visits: listOf(calendarVisitSchema).parse(r.visits),
    nextVisit: orNull(calendarVisitSchema).parse(r.next_visit),
    carePlan: listOf(careItemSchema).parse(r.care_plan),
    sourceNote: textOrNull.parse(r.source_note),
  };
});

// ── Report ─────────────────────────────────────────────────────

const reportDaySchema = obj.transform((r, ctx): ReportDay => {
  const date = dateOrNull.parse(r.date);
  if (!date) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'report day needs date' });
    return z.NEVER;
  }
  return {
    date,
    week: intOrNull.parse(r.week),
    mood: moodSchema.parse(r.mood),
    symptoms: symptomsSchema.parse(r.symptoms),
    waterGlasses: waterSchema.parse(r.water_glasses),
    visitNote: textOrNull.parse(r.visit_note),
  };
});

const reportWeightSchema = obj.transform((r, ctx): ReportWeight => {
  const date = dateOrNull.parse(r.date);
  const value = weightSchema.parse(r.value ?? r.weight);
  if (!date || value === null) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'weight needs date + value' });
    return z.NEVER;
  }
  return { date, week: intOrNull.parse(r.week), value };
});

export const pregnancyReportSchema = obj.transform((r, ctx): PregnancyReport => {
  const range = rangeOrNull.parse(r.range ?? { from: r.from, to: r.to });
  if (!range) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'report needs from/to' });
    return z.NEVER;
  }
  const byDate = <T extends { date: string }>(a: T, b: T) => a.date.localeCompare(b.date);
  return {
    range,
    generatedAt: textOrNull.parse(r.generated_at),
    age: ageFrom((typeof r.profile === 'object' && r.profile !== null ? r.profile : r) as Record<string, unknown>),
    dueDate: dateOrNull.parse(r.due_date ?? (r.profile as Record<string, unknown> | undefined)?.due_date),
    days: listOf(reportDaySchema).parse(r.days).sort(byDate),
    weights: listOf(reportWeightSchema).parse(r.weights).sort(byDate),
  };
});
