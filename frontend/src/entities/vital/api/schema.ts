import { z } from 'zod';

import { DEFAULT_THRESHOLDS } from '../model/classify';
import {
  ARMS,
  GLUCOSE_CONTEXTS,
  GLUCOSE_METHODS,
  GLUCOSE_UNITS,
  HR_CONTEXTS,
  POSITIONS,
  REPORT_RANGES,
  VITAL_TYPES,
  type BpAverage,
  type BpReport,
  type DistributionEntry,
  type GlucoseAverage,
  type GlucoseReport,
  type HrAverage,
  type HrReport,
  type PlanItem,
  type PlanWeek,
  type ReportState,
  TARGET_CONTEXTS,
  type GlucoseTargets,
  type TargetContext,
  type SavedReading,
  type VitalAlert,
  type VitalClass,
  type VitalReading,
  type VitalReport,
  type VitalsHub,
  type VitalsPlan,
  type VitalThresholds,
  type VitalType,
} from '../model/types';

/*
 * Parsers of `/api/v1/vitals*` (B-N6-01; schemas in backend-go/api/openapi.yaml,
 * internal/vitals/views.go). Snake case → camel case. Lenient on display
 * fields (`.catch`) so one odd value never blanks a screen; a reading whose
 * type is unknown is dropped. The urgent alert is parsed so it can never be
 * lost: a bad action is dropped, never the message.
 */

const text = z.string().nullable().catch(null);
const num = z.number().nullable().catch(null);
const int0 = z.number().catch(0);

function oneOf<T extends readonly [string, ...string[]]>(values: T) {
  return z
    .unknown().optional()
    .transform((v) => ((values as readonly unknown[]).includes(v) ? (v as T[number]) : null));
}

const toneSchema = z.enum(['ok', 'watch', 'high', 'urgent']).catch('watch');
const classSchema = z
  .object({ code: z.string(), tone: toneSchema })
  .nullable()
  .catch(null)
  .transform((c): VitalClass | null => c);

const rawReading = z.object({
  id: z.number().nullable().catch(null),
  source: z.enum(['vitals', 'log']).catch('vitals'),
  type: z.enum(VITAL_TYPES),
  date: z.string(),
  time: text,
  measured_at: text,
  period: z.enum(['morning', 'afternoon', 'night']).nullable().catch(null),
  blood_pressure: z
    .object({
      systolic: z.number(),
      diastolic: z.number(),
      pulse: num,
      arm: oneOf(ARMS),
      position: oneOf(POSITIONS),
    })
    .nullable()
    .catch(null),
  glucose: z
    .object({
      mg_dl: z.number(),
      mmol_l: z.number(),
      value: z.number(),
      unit: z.enum(GLUCOSE_UNITS).catch('mg_dl'),
      context: oneOf(GLUCOSE_CONTEXTS),
      method: oneOf(GLUCOSE_METHODS),
    })
    .nullable()
    .catch(null),
  heart_rate: z.object({ bpm: z.number(), context: oneOf(HR_CONTEXTS) }).nullable().catch(null),
  classification: classSchema,
  urgent: z.boolean().catch(false),
  note: text,
  editable: z.boolean().catch(false),
});

function toReading(r: z.infer<typeof rawReading>): VitalReading {
  return {
    id: r.id,
    source: r.source,
    type: r.type,
    date: r.date,
    time: r.time,
    measuredAt: r.measured_at,
    period: r.period,
    bloodPressure: r.blood_pressure,
    glucose: r.glucose
      ? {
          mgDl: r.glucose.mg_dl,
          mmolL: r.glucose.mmol_l,
          value: r.glucose.value,
          unit: r.glucose.unit,
          context: r.glucose.context,
          method: r.glucose.method,
        }
      : null,
    heartRate: r.heart_rate,
    classification: r.classification,
    urgent: r.urgent,
    note: r.note,
    editable: r.editable && r.source === 'vitals' && r.id !== null,
  };
}

export const readingSchema = rawReading.transform(toReading);

/** A reading, or null when it doesn't parse (unknown type) — lists drop those. */
const looseReading = z.unknown().optional().transform((v) => {
  const p = readingSchema.safeParse(v);
  return p.success ? p.data : null;
});

const readingList = z
  .array(z.unknown())
  .catch([])
  .transform((list) => list.map((v) => looseReading.parse(v)).filter((r): r is VitalReading => r !== null));

/** GET /vitals/readings: `{type, from, to, count, items}` → the items. */
export const readingsSchema = z.object({ items: readingList }).transform((r) => r.items);

const weekSchema = z
  .object({
    from: z.string().catch(''),
    to: z.string().catch(''),
    planned: int0,
    done: int0,
    items: z
      .array(
        z.object({
          type: z.enum(VITAL_TYPES),
          slot: z.string(),
          planned: int0,
          done: int0,
          days: z
            .array(z.object({ date: z.string(), state: z.enum(['done', 'missed', 'due']).catch('due') }))
            .catch([]),
        }),
      )
      .catch([]),
  })
  .transform((w): PlanWeek => w);

const statusSchema = z.enum(['ok', 'not_enough_data']).catch('ok');

const reportStateSchema = z
  .object({ range: z.enum(REPORT_RANGES), status: statusSchema, readings_needed: int0 })
  .nullable()
  .catch(null)
  .transform((s): ReportState | null => (s ? { range: s.range, status: s.status, readingsNeeded: s.readings_needed } : null));

const notificationsSchema = z.object({ enabled: z.boolean().catch(false) }).catch({ enabled: false });

export const hubSchema = z
  .object({
    date: z.string(),
    latest: z
      .object({ bp: looseReading.optional(), hr: looseReading.optional(), glucose: looseReading.optional() })
      .catch({}),
    plan: weekSchema,
    recent: readingList,
    notifications: notificationsSchema,
    reports: z
      .object({ bp: reportStateSchema, hr: reportStateSchema, glucose: reportStateSchema })
      .partial()
      .catch({}),
  })
  .transform(
    (h): VitalsHub => ({
      date: h.date,
      latest: { bp: h.latest.bp ?? null, hr: h.latest.hr ?? null, glucose: h.latest.glucose ?? null },
      plan: h.plan,
      recent: h.recent,
      remindersEnabled: h.notifications.enabled,
      reports: { bp: h.reports.bp ?? null, hr: h.reports.hr ?? null, glucose: h.reports.glucose ?? null },
    }),
  );

const planItemSchema = z
  .object({
    type: z.enum(VITAL_TYPES),
    slot: z.string(),
    days: z.array(z.number().int()).catch([]),
    remind_at: text,
  })
  .transform((i): PlanItem => ({ type: i.type, slot: i.slot, days: i.days, remindAt: i.remind_at }));

export const planSchema = z
  .object({
    items: z.array(planItemSchema).catch([]),
    week: weekSchema,
    slots: z
      .object({
        bp: z.array(z.string()).catch([]),
        hr: z.array(z.string()).catch([]),
        glucose: z.array(z.string()).catch([]),
      })
      .catch({ bp: ['morning', 'evening'], hr: ['morning', 'evening'], glucose: ['fasting', 'before_meal', 'after_meal', 'bedtime'] }),
    notifications: notificationsSchema,
  })
  .transform(
    (p): VitalsPlan => ({ items: p.items, week: p.week, slots: p.slots, remindersEnabled: p.notifications.enabled }),
  );

const actionSchema = z.object({ key: z.string().min(1), label: z.string().min(1), phone: z.string().regex(/^\d{3,6}$/).optional() });

export const alertSchema = z
  .object({
    rule: z.string().catch(''),
    title: text,
    what_we_saw: text,
    advice: text,
    contact: text,
    actions: z.array(z.unknown()).catch([]),
  })
  .transform(
    (a): VitalAlert => ({
      rule: a.rule,
      title: a.title,
      whatWeSaw: a.what_we_saw,
      advice: a.advice,
      contact: a.contact,
      actions: a.actions.flatMap((raw) => {
        const p = actionSchema.safeParse(raw);
        return p.success ? [{ key: p.data.key, label: p.data.label, phone: p.data.phone ?? null }] : [];
      }),
    }),
  );

export const savedSchema = z
  .object({ reading: readingSchema, alert: z.unknown().nullable().optional() })
  .transform((s): SavedReading => {
    let alert: VitalAlert | null = null;
    if (s.alert != null) {
      const p = alertSchema.safeParse(s.alert);
      // An urgent payload that fails to parse still opens the modal (with the bundled fallback copy).
      alert = p.success ? p.data : { rule: '', title: null, whatWeSaw: null, advice: null, contact: null, actions: [] };
    }
    return { reading: s.reading, alert };
  });

const distribution = z
  .array(z.object({ code: z.string(), tone: toneSchema, count: int0, percent: int0 }))
  .catch([])
  .transform((d): DistributionEntry[] => d);

const bpAvg = z
  .object({ systolic: z.number(), diastolic: z.number(), readings: int0, classification: classSchema })
  .nullable()
  .catch(null)
  .transform((a): BpAverage | null => a);
const glucoseAvg = z
  .object({ mg_dl: z.number(), mmol_l: z.number(), readings: int0 })
  .nullable()
  .catch(null)
  .transform((a): GlucoseAverage | null => (a ? { mgDl: a.mg_dl, mmolL: a.mmol_l, readings: a.readings } : null));
const hrAvg = z
  .object({ bpm: z.number(), readings: int0 })
  .nullable()
  .catch(null)
  .transform((a): HrAverage | null => a);

const rangeSchema = z.object({ key: z.enum(REPORT_RANGES), from: z.string(), to: z.string(), days: int0 });
const tir = z
  .object({ in_range: int0, below: int0, above: int0, readings: int0, percent: int0 })
  .catch({ in_range: 0, below: 0, above: 0, readings: 0, percent: 0 })
  .transform((t) => ({ inRange: t.in_range, below: t.below, above: t.above, readings: t.readings, percent: t.percent }));
const nullableReading = z.unknown().optional().transform((v) => (v == null ? null : looseReading.parse(v)));

function mvn<T extends z.ZodType>(avg: T) {
  return z
    .object({ morning: avg, night: avg, night_out_of_range: int0 })
    .catch({ morning: null, night: null, night_out_of_range: 0 } as never)
    .transform((m) => {
      const r = m as { morning: unknown; night: unknown; night_out_of_range: number };
      return { morning: r.morning as z.infer<T>, night: r.night as z.infer<T>, nightOutOfRange: r.night_out_of_range };
    });
}

const base = {
  range: rangeSchema,
  status: statusSchema,
  readings_needed: int0,
  readings: int0,
  min: nullableReading,
  max: nullableReading,
  distribution,
  time_in_range: tir,
};

const bpReport = z
  .object({
    ...base,
    type: z.literal('bp'),
    average: bpAvg,
    pulse: hrAvg,
    morning_vs_night: mvn(bpAvg),
    target: z
      .object({ systolic_max: z.number(), diastolic_max: z.number() })
      .catch({ systolic_max: 120, diastolic_max: 80 }),
    series: z
      .array(z.object({ date: z.string(), readings: int0, systolic: z.number(), diastolic: z.number() }))
      .catch([]),
  })
  .transform(
    (r): BpReport => ({
      type: 'bp',
      range: r.range,
      status: r.status,
      readingsNeeded: r.readings_needed,
      readings: r.readings,
      average: r.average,
      min: r.min,
      max: r.max,
      distribution: r.distribution,
      morningVsNight: r.morning_vs_night,
      timeInRange: r.time_in_range,
      pulse: r.pulse,
      target: { systolicMax: r.target.systolic_max, diastolicMax: r.target.diastolic_max },
      series: r.series,
    }),
  );

const glucoseReport = z
  .object({
    ...base,
    type: z.literal('glucose'),
    filter: z.string().catch('all'),
    average: glucoseAvg,
    morning_vs_night: mvn(glucoseAvg),
    by_context: z
      .array(
        z.object({
          context: z.enum(GLUCOSE_CONTEXTS),
          readings: int0,
          average: glucoseAvg,
          above_target: int0,
          target: z.object({ min: z.number(), max: z.number().nullable().catch(null) }),
        }),
      )
      .catch([]),
    series: z
      .array(z.object({ date: z.string(), readings: int0, fasting: num, after_meal: num, other: num }))
      .catch([]),
  })
  .transform(
    (r): GlucoseReport => ({
      type: 'glucose',
      range: r.range,
      status: r.status,
      readingsNeeded: r.readings_needed,
      readings: r.readings,
      average: r.average,
      min: r.min,
      max: r.max,
      distribution: r.distribution,
      morningVsNight: r.morning_vs_night,
      timeInRange: r.time_in_range,
      filter: (['all', ...GLUCOSE_CONTEXTS] as readonly string[]).includes(r.filter) ? (r.filter as GlucoseReport['filter']) : 'all',
      byContext: r.by_context.map((c) => ({
        context: c.context,
        readings: c.readings,
        average: c.average,
        aboveTarget: c.above_target,
        target: c.target,
      })),
      series: r.series.map((p) => ({ date: p.date, readings: p.readings, fasting: p.fasting, afterMeal: p.after_meal, other: p.other })),
    }),
  );

const hrReport = z
  .object({
    ...base,
    type: z.literal('hr'),
    average: hrAvg,
    resting_average: hrAvg,
    morning_vs_night: mvn(hrAvg),
    by_context: z
      .array(z.object({ context: z.enum(HR_CONTEXTS), readings: int0, average: hrAvg }))
      .catch([]),
    target: z.object({ min: z.number(), max: z.number() }).catch({ min: 60, max: 100 }),
    series: z.array(z.object({ date: z.string(), readings: int0, bpm: z.number() })).catch([]),
  })
  .transform(
    (r): HrReport => ({
      type: 'hr',
      range: r.range,
      status: r.status,
      readingsNeeded: r.readings_needed,
      readings: r.readings,
      average: r.average,
      min: r.min,
      max: r.max,
      distribution: r.distribution,
      morningVsNight: r.morning_vs_night,
      timeInRange: r.time_in_range,
      restingAverage: r.resting_average,
      byContext: r.by_context,
      target: r.target,
      series: r.series,
    }),
  );

export function parseReport(type: VitalType, data: unknown): VitalReport {
  switch (type) {
    case 'bp':
      return bpReport.parse(data);
    case 'glucose':
      return glucoseReport.parse(data);
    case 'hr':
      return hrReport.parse(data);
  }
}

const band = z.object({ target_min: z.number(), target_max: z.number(), very_high_from: z.number() });

export const thresholdsSchema = z
  .object({
    version: z.string().catch(DEFAULT_THRESHOLDS.version),
    blood_pressure: z.object({
      elevated_systolic: z.number(),
      stage1: z.object({ systolic: z.number(), diastolic: z.number() }),
      stage2: z.object({ systolic: z.number(), diastolic: z.number() }),
      urgent_above: z.object({ systolic: z.number(), diastolic: z.number() }),
    }),
    glucose: z.object({
      low_below: z.number(),
      urgent_below: z.number(),
      mmol_factor: z.number(),
      contexts: z.object({ fasting: band, before_meal: band, after_meal: band, bedtime: band, random: band }),
      contexts_spec: z.number().catch(0),
      diabetic: z
        .object({ fasting_lo: z.number(), fasting_hi: z.number(), post_meal_max: z.number() })
        .catch({ fasting_lo: 80, fasting_hi: 130, post_meal_max: 180 }),
    }),
    heart_rate: z.object({
      resting: z.object({ min: z.number(), max: z.number() }),
      classified_contexts: z.array(z.string()),
    }),
  })
  .transform((t): VitalThresholds => {
    const ctx = (b: z.infer<typeof band>) => ({ targetMin: b.target_min, targetMax: b.target_max, veryHighFrom: b.very_high_from });
    const g = t.glucose.contexts;
    return {
      version: t.version,
      bp: {
        elevatedSystolic: t.blood_pressure.elevated_systolic,
        stage1: t.blood_pressure.stage1,
        stage2: t.blood_pressure.stage2,
        urgentAbove: t.blood_pressure.urgent_above,
      },
      glucose: {
        lowBelow: t.glucose.low_below,
        urgentBelow: t.glucose.urgent_below,
        mmolFactor: t.glucose.mmol_factor,
        contexts: {
          fasting: ctx(g.fasting),
          before_meal: ctx(g.before_meal),
          after_meal: ctx(g.after_meal),
          bedtime: ctx(g.bedtime),
          random: ctx(g.random),
        },
        contextsSpec: t.glucose.contexts_spec === 1,
        diabetic: {
          fastingLo: t.glucose.diabetic.fasting_lo,
          fastingHi: t.glucose.diabetic.fasting_hi,
          postMealMax: t.glucose.diabetic.post_meal_max,
        },
      },
      hr: {
        min: t.heart_rate.resting.min,
        max: t.heart_rate.resting.max,
        classifiedContexts: t.heart_rate.classified_contexts,
      },
    };
  });

const targetNum = z.number().nullable().catch(null);
const targetDefault = z
  .object({ low_mgdl: targetNum, high_mgdl: targetNum, high_inclusive: z.boolean().catch(true) })
  .nullable()
  .catch(null)
  .transform((d) => (d ? { lowMgDl: d.low_mgdl, highMgDl: d.high_mgdl, highInclusive: d.high_inclusive } : null));

const targetItem = z.object({
  context: z.enum(TARGET_CONTEXTS),
  low_mgdl: targetNum,
  high_mgdl: targetNum,
  high_inclusive: z.boolean().catch(true),
  set_by: z.enum(['doctor', 'user']).catch('user'),
  note: z.string().nullable().catch(null),
}).transform((i) => ({
  context: i.context as TargetContext,
  lowMgDl: i.low_mgdl,
  highMgDl: i.high_mgdl,
  highInclusive: i.high_inclusive,
  setBy: i.set_by,
  note: i.note,
}));

/** GET|PUT|DELETE /vitals/targets — the diabetic glucose targets (T-M10-19). */
export const targetsSchema = z
  .object({
    diabetic: z.boolean().catch(false),
    bounds: z.object({ min_mgdl: z.number(), max_mgdl: z.number() }).catch({ min_mgdl: 50, max_mgdl: 300 }),
    defaults: z
      .object({ fasting: targetDefault, pre_meal: targetDefault, post_meal: targetDefault, bedtime: targetDefault })
      .partial()
      .catch({}),
    items: z.array(z.unknown()).catch([]),
  })
  .transform(
    (t): GlucoseTargets => ({
      diabetic: t.diabetic,
      bounds: { min: t.bounds.min_mgdl, max: t.bounds.max_mgdl },
      defaults: {
        fasting: t.defaults.fasting ?? null,
        pre_meal: t.defaults.pre_meal ?? null,
        post_meal: t.defaults.post_meal ?? null,
        bedtime: t.defaults.bedtime ?? null,
      },
      items: t.items.flatMap((raw) => {
        const p = targetItem.safeParse(raw);
        return p.success ? [p.data] : [];
      }),
    }),
  );

