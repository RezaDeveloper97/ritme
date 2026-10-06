import { z } from 'zod';

import type {
  CatalogMarker,
  Lab,
  LabConsent,
  LabList,
  LabMarker,
  LabStage,
  LabStatus,
  LabStatusPoll,
  LabTrends,
  MarkerDetail,
  MarkerState,
  MarkerTrend,
  RedFlag,
} from '../model/types';

/*
 * Parsers of `/api/v1/labs/*` (B-N6-06, `internal/labs/views.go`). Strict on
 * identity (ids, status), lenient (`.catch`) on display fields so one odd value
 * never blanks a screen. AI text stays plain strings — rendered as text only.
 */

const text = z.string().nullable().catch(null);
const num = z.number().nullable().catch(null);
const str = z.string().catch('');
const STATUSES = ['queued', 'extracting', 'needs_review', 'interpreting', 'ready', 'failed'] as const;
const STAGES = ['queued', 'reading', 'extracting', 'review', 'explaining', 'done', 'failed'] as const;
const STATES = ['low', 'borderline_low', 'normal', 'borderline_high', 'high', 'unknown'] as const;
const status = z.enum(STATUSES) as z.ZodType<LabStatus>;
const stage = z.enum(STAGES).catch('queued') as z.ZodType<LabStage>;
const state = z.enum(STATES).catch('unknown') as z.ZodType<MarkerState>;
const strings = z
  .array(z.string().nullable())
  .catch([])
  .transform((l) => l.filter((s): s is string => typeof s === 'string' && s.trim() !== ''));

export const markerSchema = z
  .object({
    id: z.number().int(),
    code: text,
    name: str,
    printed_name: str,
    subtitle: text,
    value: num,
    value_text: text,
    unit: text,
    reference: z
      .object({ low: num, high: num, text, source: text })
      .catch({ low: null, high: null, text: null, source: null }),
    state,
    state_label: str,
    attention: z.boolean().catch(false),
    confidence: num,
    low_confidence: z.boolean().catch(false),
    source: str,
  })
  .transform(
    (m): LabMarker => ({
      id: m.id,
      code: m.code,
      name: m.name || m.printed_name,
      printedName: m.printed_name || m.name,
      subtitle: m.subtitle,
      value: m.value,
      valueText: m.value_text,
      unit: m.unit,
      reference: m.reference,
      state: m.state,
      stateLabel: m.state_label,
      attention: m.attention,
      confidence: m.confidence,
      lowConfidence: m.low_confidence,
      source: m.source,
    }),
  );

const redFlagSchema = z
  .object({
    marker_id: z.number().int().catch(0),
    name: str,
    severity: z.enum(['urgent', 'soon']).catch('soon'),
    message: str,
  })
  .transform((f): RedFlag => ({ markerId: f.marker_id, name: f.name, severity: f.severity, message: f.message }));

const redFlags = z
  .array(redFlagSchema.nullable().catch(null))
  .catch([])
  .transform((l) => l.filter((f): f is RedFlag => f !== null && f.message !== ''));

const interpretationSchema = z
  .object({
    summary: str,
    source: z.enum(['ai', 'rules']).catch('rules'),
    stale: z.boolean().catch(false),
    red_flags: redFlags,
    doctor_questions: strings,
    disclaimer: str,
  })
  .transform((i) => ({
    summary: i.summary,
    source: i.source,
    stale: i.stale,
    redFlags: i.red_flags,
    doctorQuestions: i.doctor_questions,
    disclaimer: i.disclaimer,
  }));

const countsSchema = z
  .object({
    total: z.number().int().catch(0),
    normal: z.number().int().catch(0),
    attention: z.number().int().catch(0),
    low: z.number().int().catch(0),
    borderline_low: z.number().int().catch(0),
    high: z.number().int().catch(0),
    borderline_high: z.number().int().catch(0),
    unknown: z.number().int().catch(0),
  })
  .transform((c) => ({
    total: c.total,
    normal: c.normal,
    attention: c.attention,
    low: c.low,
    borderlineLow: c.borderline_low,
    high: c.high,
    borderlineHigh: c.borderline_high,
    unknown: c.unknown,
  }));

const markers = z
  .array(markerSchema.nullable().catch(null))
  .catch([])
  .transform((l) => l.filter((m): m is LabMarker => m !== null));

export const labSchema = z
  .object({
    id: z.number().int(),
    source: z.enum(['upload', 'manual']).catch('upload'),
    category: str,
    display_title: str,
    taken_on: text,
    date: str,
    fasting: z.boolean().nullable().catch(null),
    lab_name: text,
    status,
    stage,
    progress: z.number().catch(0),
    error_code: text,
    error_message: text,
    editable: z.boolean().catch(false),
    counts: countsSchema,
    low_confidence_count: z.number().int().catch(0),
    markers,
    files: z
      .array(z.object({ id: z.number().int(), page: z.number().int().catch(1), mime: str }).nullable().catch(null))
      .catch([])
      .transform((l) => l.filter((f) => f !== null)),
    interpretation: interpretationSchema.nullable().catch(null),
    feedback: z.object({ helpful: z.boolean() }).nullable().catch(null),
  })
  .transform(
    (l): Lab => ({
      id: l.id,
      source: l.source,
      category: l.category,
      title: l.display_title,
      takenOn: l.taken_on,
      date: l.date,
      fasting: l.fasting,
      labName: l.lab_name,
      status: l.status,
      stage: l.stage,
      progress: l.progress,
      errorCode: l.error_code,
      errorMessage: l.error_message,
      editable: l.editable,
      counts: l.counts,
      lowConfidenceCount: l.low_confidence_count,
      markers: l.markers,
      files: l.files,
      interpretation: l.interpretation,
      feedback: l.feedback,
    }),
  );

const listItemSchema = z
  .object({
    id: z.number().int(),
    source: z.enum(['upload', 'manual']).catch('upload'),
    category: str,
    display_title: str,
    date: str,
    status,
    stage,
    progress: z.number().catch(0),
    marker_count: z.number().int().catch(0),
    attention_count: z.number().int().catch(0),
    all_normal: z.boolean().catch(false),
  })
  .transform((l) => ({
    id: l.id,
    source: l.source,
    category: l.category,
    title: l.display_title,
    date: l.date,
    status: l.status,
    stage: l.stage,
    progress: l.progress,
    markerCount: l.marker_count,
    attentionCount: l.attention_count,
    allNormal: l.all_normal,
  }));

export const labListSchema = z
  .object({
    labs: z
      .array(listItemSchema.nullable().catch(null))
      .catch([])
      .transform((l) => l.filter((x) => x !== null)),
    limits: z
      .object({
        max_files: z.number().int().catch(5),
        max_image_kb: z.number().int().catch(5120),
        max_pdf_kb: z.number().int().catch(10240),
        categories: z.array(z.string()).catch([]),
      })
      .catch({ max_files: 5, max_image_kb: 5120, max_pdf_kb: 10240, categories: [] }),
  })
  .transform(
    (d): LabList => ({
      labs: d.labs,
      limits: {
        maxFiles: d.limits.max_files,
        maxImageKb: d.limits.max_image_kb,
        maxPdfKb: d.limits.max_pdf_kb,
        categories: d.limits.categories,
      },
    }),
  );

export const statusSchema = z
  .object({
    id: z.number().int(),
    status,
    stage,
    progress: z.number().catch(0),
    marker_count: z.number().int().catch(0),
    error_message: text,
  })
  .transform(
    (s): LabStatusPoll => ({
      id: s.id,
      status: s.status,
      stage: s.stage,
      progress: s.progress,
      markerCount: s.marker_count,
      errorMessage: s.error_message,
    }),
  );

const trendSchema = z
  .object({
    count: z.number().int().catch(0),
    direction: text,
    sentence: str,
    points: z
      .array(
        z
          .object({
            lab_id: z.number().int(),
            marker_id: z.number().int(),
            date: z.string(),
            value: z.number(),
            state,
          })
          .nullable()
          .catch(null),
      )
      .catch([]),
  })
  .transform(
    (t): MarkerTrend => ({
      count: t.count,
      direction: t.direction,
      sentence: t.sentence,
      points: t.points
        .filter((p) => p !== null)
        .map((p) => ({ labId: p.lab_id, markerId: p.marker_id, date: p.date, value: p.value, state: p.state })),
    }),
  );

export const markerDetailSchema = z
  .object({
    lab: z.object({ id: z.number().int(), display_title: str, date: str, status }),
    marker: markerSchema,
    about: z
      .object({ name: str, subtitle: text, body: text, typical_range: text })
      .nullable()
      .catch(null),
    factors: strings,
    see_doctor: text,
    context_notes: strings,
    red_flag: redFlagSchema.nullable().catch(null),
    trend: trendSchema,
    disclaimer: str,
  })
  .transform(
    (d): MarkerDetail => ({
      lab: { id: d.lab.id, title: d.lab.display_title, date: d.lab.date, status: d.lab.status },
      marker: d.marker,
      about: d.about
        ? { name: d.about.name, subtitle: d.about.subtitle, body: d.about.body, typicalRange: d.about.typical_range }
        : null,
      factors: d.factors,
      seeDoctor: d.see_doctor,
      contextNotes: d.context_notes,
      redFlag: d.red_flag && d.red_flag.message ? d.red_flag : null,
      trend: d.trend,
      disclaimer: d.disclaimer,
    }),
  );

const seriesSchema = z
  .object({
    key: z.string(),
    name: str,
    unit: text,
    reference_text: text,
    latest: z.object({
      lab_id: z.number().int(),
      marker_id: z.number().int(),
      value: num,
      value_text: text,
      state,
      state_label: str,
      attention: z.boolean().catch(false),
    }),
    trend: trendSchema,
  })
  .transform((s) => ({
    key: s.key,
    name: s.name,
    unit: s.unit,
    referenceText: s.reference_text,
    latest: {
      labId: s.latest.lab_id,
      markerId: s.latest.marker_id,
      value: s.latest.value,
      valueText: s.latest.value_text,
      state: s.latest.state,
      stateLabel: s.latest.state_label,
      attention: s.latest.attention,
    },
    trend: s.trend,
  }));

export const trendsSchema = z
  .object({
    labs_count: z.number().int().catch(0),
    min_points: z.number().int().catch(2),
    markers: z
      .array(seriesSchema.nullable().catch(null))
      .catch([])
      .transform((l) => l.filter((s) => s !== null)),
  })
  .transform((t): LabTrends => ({ labsCount: t.labs_count, minPoints: t.min_points, markers: t.markers }));

export const catalogSchema = z
  .object({
    markers: z
      .array(
        z
          .object({ code: z.string(), name: str, unit: text, typical_range: text })
          .nullable()
          .catch(null),
      )
      .catch([]),
  })
  .transform((c): CatalogMarker[] =>
    c.markers
      .filter((m) => m !== null)
      .map((m) => ({ code: m.code, name: m.name, unit: m.unit, typicalRange: m.typical_range })),
  );

export const consentSchema = z
  .object({
    consent: z.object({
      code: z.string(),
      version: z.number().int(),
      title: str,
      body: str,
      points: strings,
      granted: z.boolean().catch(false),
      needs_consent: z.boolean().catch(true),
    }),
  })
  .transform(
    ({ consent: c }): LabConsent => ({
      code: c.code,
      version: c.version,
      title: c.title,
      body: c.body,
      points: c.points,
      granted: c.granted,
      needsConsent: c.needs_consent,
    }),
  );
