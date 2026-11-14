import { z } from 'zod';

import {
  ANALYSIS_RANGES,
  type AnalysisPhrase,
  type AnalysisSection,
  type AnalysisSummary,
  type Correlation,
  type CyclePhase,
  type HubCycle,
  type HubRecentCycle,
  type HubSymptoms,
  type HubVitals,
  type HubWeight,
  type MoodByPhase,
  type TopFinding,
} from '../model/types';

/**
 * Boundary parsers for `/api/v1/analysis/*` (CLAUDE.md §10 — zod at the
 * boundary). Shapes: backend-go/internal/analysis (goldens under testdata/golden).
 * Tolerant where a newer backend may move ahead of this bundle: an unknown enum
 * value parses to `null`, an unknown phase row is dropped. B-N3-09…12 add their
 * report parsers here next to the summary.
 */

/** Output type `T`, any input (the transforms rename snake_case keys). */
type Parser<T> = z.ZodType<T, z.ZodTypeDef, unknown>;

const num = z.number();
const nullableNum = z.number().nullable().catch(null);
/** An enum that degrades to `null` instead of failing the whole report. */
const softEnum = <const T extends [string, ...string[]]>(values: T) =>
  z.enum(values).nullable().catch(null);

export const analysisRangeSchema = z.object({
  key: z.enum(ANALYSIS_RANGES),
  from: z.string(),
  to: z.string(),
  days: num,
});

export const phraseSchema: Parser<AnalysisPhrase> = z.object({
  key: z.string(),
  params: z.record(z.string(), z.union([z.string(), z.number()])).catch({}),
  text: z.string(),
});

const topFindingSchema: Parser<TopFinding> = z
  .object({
    kind: softEnum([
      'cycle_frequent',
      'cycle_infrequent',
      'period_prolonged',
      'cycle_irregular',
      'cycle_regular',
      'pattern',
      'correlation',
      'mood_by_phase',
      'not_enough_data',
      'keep_logging',
      'no_data',
    ]),
    parts: z.array(phraseSchema),
    text: z.string(),
    strength: nullableNum.optional(),
    score: nullableNum.optional(),
    evidence_date: z.string().nullable().catch(null).optional(),
    days_needed: nullableNum.optional(),
  })
  .transform((f) => ({
    kind: f.kind,
    parts: f.parts,
    text: f.text,
    strength: f.strength ?? null,
    score: f.score ?? null,
    evidenceDate: f.evidence_date ?? null,
    daysNeeded: f.days_needed ?? null,
  }));

/**
 * `{plus, locked, ready, data}` around a body parser; a locked section never
 * carries data. A body this bundle can't read becomes `null` (the card shows its
 * not-ready line) instead of failing the whole hub.
 */
export function sectionSchema<T>(data: Parser<T>): Parser<AnalysisSection<T>> {
  return z
    .object({ plus: z.boolean(), locked: z.boolean(), ready: z.boolean(), data: data.nullable().catch(null) })
    .transform((s) => ({
      plus: s.plus,
      locked: s.locked,
      ready: s.ready && s.data != null,
      data: s.locked ? null : ((s.data ?? null) as T | null),
    }));
}

const hubCycleSchema: Parser<HubCycle> = z
  .object({
    median_cycle: nullableNum,
    median_period: nullableNum,
    cycle_status: softEnum(['normal', 'frequent', 'infrequent']),
    regularity: z.enum(['regular', 'irregular', 'not_enough_data']).catch('not_enough_data'),
    variation_days: nullableNum,
    variation_max: num,
    based_on_cycles: num,
    bars: z.array(z.object({ start: z.string(), length: num, in_figo_range: z.boolean() })),
  })
  .transform((c) => ({
    medianCycle: c.median_cycle,
    medianPeriod: c.median_period,
    cycleStatus: c.cycle_status,
    regularity: c.regularity,
    variationDays: c.variation_days,
    variationMax: c.variation_max,
    basedOnCycles: c.based_on_cycles,
    bars: c.bars.map((b) => ({ start: b.start, length: b.length, inFigoRange: b.in_figo_range })),
  }));

const recentCyclesSchema: Parser<HubRecentCycle[]> = z.array(
  z
    .object({
      start: z.string(),
      length: num,
      is_current: z.boolean(),
      days_so_far: nullableNum,
      period_days: num,
      fertile_start_day: num,
      fertile_end_day: num,
      ovulation_day: num,
    })
    .transform((c) => ({
      start: c.start,
      length: c.length,
      isCurrent: c.is_current,
      daysSoFar: c.days_so_far,
      periodDays: c.period_days,
      fertileStartDay: c.fertile_start_day,
      fertileEndDay: c.fertile_end_day,
      ovulationDay: c.ovulation_day,
    })),
);

const hubSymptomsSchema: Parser<HubSymptoms> = z
  .object({
    cycles_counted: num,
    cycles_needed: num,
    top: z.array(z.object({ key: z.string(), label: z.string(), days: num, cycles: num })),
    highlight: z
      .object({
        key: z.string(),
        label: z.string(),
        cycles: num,
        of_cycles: num,
        relation: softEnum(['before_period', 'early', 'mid']),
        days: nullableNum,
        start_day: num,
        end_day: num,
      })
      .nullable(),
  })
  .transform((s) => ({
    cyclesCounted: s.cycles_counted,
    cyclesNeeded: s.cycles_needed,
    top: s.top,
    highlight: s.highlight
      ? {
          key: s.highlight.key,
          label: s.highlight.label,
          cycles: s.highlight.cycles,
          ofCycles: s.highlight.of_cycles,
          relation: s.highlight.relation,
          days: s.highlight.days,
          startDay: s.highlight.start_day,
          endDay: s.highlight.end_day,
        }
      : null,
  }));

const PHASES = ['period', 'follicular', 'fertile', 'luteal'] as const satisfies readonly CyclePhase[];

const moodShareSchema = z.object({ mood: z.string(), label: z.string(), days: num, pct: num });

/** «حال در فازها» (§4.1): % of days per mood in each phase with ≥ 10 mood-logged days. */
export const moodByPhaseSchema: Parser<MoodByPhase> = z
  .object({
    ready: z.boolean().catch(false),
    min_days: num.catch(10),
    days_needed: num.catch(0),
    phases: z.array(
      z.object({
        phase: z.string(),
        ready: z.boolean().catch(false),
        good_pct: nullableNum,
        days: num,
        moods: z.array(moodShareSchema).catch([]),
      }),
    ),
    finding: phraseSchema.nullable().catch(null),
  })
  .transform((m) => ({
    ready: m.ready,
    minDays: m.min_days,
    daysNeeded: m.days_needed,
    phases: m.phases
      .filter((p): p is typeof p & { phase: CyclePhase } => (PHASES as readonly string[]).includes(p.phase))
      .map((p) => ({ phase: p.phase, ready: p.ready, goodPct: p.good_pct, days: p.days, moods: p.moods })),
    finding: m.finding,
  }));

export const correlationSchema: Parser<Correlation> = z
  .object({
    key: z.string(),
    status: z.string(),
    strength: z.string().nullable().catch(null),
    n: num,
    min_days: num,
    days_needed: num.catch(0),
    strong_rule: z.object({ days: num, ratio: num }).nullable().catch(null),
    groups: z.array(z.object({ key: z.string(), days: num, hits: num, pct: num })),
    ratio: nullableNum,
    finding: phraseSchema.nullable().catch(null),
  })
  .transform((c) => ({
    key: c.key,
    status: c.status,
    strength: c.strength,
    n: c.n,
    minDays: c.min_days,
    daysNeeded: c.days_needed,
    strongRule: c.strong_rule,
    groups: c.groups,
    ratio: c.ratio,
    finding: c.finding,
  }));

/** The weight block (hub and body report): §4.1 — 3 weigh-ins in the last 14 days. */
export const weightSchema = z.object({
  ready: z.boolean(),
  logs_in_window: num.catch(0),
  min_logs: num.catch(3),
  window_days: num.catch(14),
  days_needed: num.catch(0),
  unit: z.string(),
  current: nullableNum,
  as_of: z.string().nullable(),
  delta_7d: nullableNum,
  delta_30d: nullableNum,
  delta_range: nullableNum,
  bmi: nullableNum,
  // A day without a weigh-in has `value: null` (the 7-day average still runs).
  points: z.array(z.object({ date: z.string(), value: nullableNum, avg7: nullableNum })).catch([]),
});

export function toWeight(w: z.infer<typeof weightSchema>): HubWeight {
  return {
    ready: w.ready,
    logsInWindow: w.logs_in_window,
    minLogs: w.min_logs,
    windowDays: w.window_days,
    daysNeeded: w.days_needed,
    unit: w.unit,
    current: w.current,
    asOf: w.as_of,
    delta7d: w.delta_7d,
    delta30d: w.delta_30d,
    deltaRange: w.delta_range,
    bmi: w.bmi,
    points: w.points,
  };
}

const hubWeightSchema: Parser<HubWeight> = weightSchema.transform(toWeight);

const readiness = {
  ready: z.boolean().catch(false),
  readings: num,
  days: num.catch(0),
  readings_needed: num.catch(0),
  classes: z.record(z.string(), num).nullable().catch(null),
};
const bpPair = z.object({ systolic: num, diastolic: num }).nullable().catch(null);

/** The vitals card (§4.1: mean, min–max, class distribution from 2 readings on 2 days). */
const hubVitalsSchema: Parser<HubVitals> = z
  .object({
    min_readings: num.catch(2),
    min_days: num.catch(2),
    blood_pressure: z
      .object({ ...readiness, systolic: nullableNum, diastolic: nullableNum, min: bpPair, max: bpPair })
      .nullable(),
    blood_sugar: z.object({ ...readiness, avg: nullableNum, min: nullableNum, max: nullableNum }).nullable(),
    // Added after the first release: an older server omits it.
    heart_rate: z.object({ ...readiness, avg: nullableNum, min: nullableNum, max: nullableNum }).nullable().catch(null),
  })
  .transform((v) => ({
    minReadings: v.min_readings,
    minDays: v.min_days,
    bloodPressure: v.blood_pressure
      ? {
          ready: v.blood_pressure.ready,
          readings: v.blood_pressure.readings,
          days: v.blood_pressure.days,
          readingsNeeded: v.blood_pressure.readings_needed,
          classes: v.blood_pressure.classes,
          systolic: v.blood_pressure.systolic,
          diastolic: v.blood_pressure.diastolic,
          min: v.blood_pressure.min,
          max: v.blood_pressure.max,
        }
      : null,
    bloodSugar: v.blood_sugar
      ? {
          ready: v.blood_sugar.ready,
          readings: v.blood_sugar.readings,
          days: v.blood_sugar.days,
          readingsNeeded: v.blood_sugar.readings_needed,
          classes: v.blood_sugar.classes,
          avg: v.blood_sugar.avg,
          min: v.blood_sugar.min,
          max: v.blood_sugar.max,
        }
      : null,
    heartRate: v.heart_rate
      ? {
          ready: v.heart_rate.ready,
          readings: v.heart_rate.readings,
          days: v.heart_rate.days,
          readingsNeeded: v.heart_rate.readings_needed,
          classes: v.heart_rate.classes,
          avg: v.heart_rate.avg,
          min: v.heart_rate.min,
          max: v.heart_rate.max,
        }
      : null,
  }));

/** `data` of `GET /analysis/summary`. */
export const analysisSummarySchema: Parser<AnalysisSummary> = z
  .object({
    range: analysisRangeSchema,
    top_finding: topFindingSchema,
    sections: z.object({
      cycle: sectionSchema(hubCycleSchema),
      recent_cycles: sectionSchema(recentCyclesSchema),
      symptoms: sectionSchema(hubSymptomsSchema),
      mood_by_phase: sectionSchema(moodByPhaseSchema),
      sleep_mood: sectionSchema(correlationSchema),
      weight: sectionSchema(hubWeightSchema),
      vitals: sectionSchema(hubVitalsSchema),
      labs: sectionSchema(z.unknown()),
    }),
  })
  .transform((s) => ({
    range: s.range,
    topFinding: s.top_finding,
    sections: {
      cycle: s.sections.cycle,
      recentCycles: s.sections.recent_cycles,
      symptoms: s.sections.symptoms,
      moodByPhase: s.sections.mood_by_phase,
      sleepMood: s.sections.sleep_mood,
      weight: s.sections.weight,
      vitals: s.sections.vitals,
      labs: s.sections.labs,
    },
  }));
