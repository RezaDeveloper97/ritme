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

const topFindingSchema: Parser<TopFinding> = z.object({
  kind: softEnum([
    'cycle_frequent',
    'cycle_infrequent',
    'period_prolonged',
    'cycle_irregular',
    'cycle_regular',
    'pattern',
    'not_enough_data',
    'keep_logging',
    'no_data',
  ]),
  parts: z.array(phraseSchema),
  text: z.string(),
});

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

const moodByPhaseSchema: Parser<MoodByPhase> = z
  .object({
    phases: z.array(z.object({ phase: z.string(), good_pct: nullableNum, days: num })),
    finding: phraseSchema.nullable().catch(null),
  })
  .transform((m) => ({
    phases: m.phases
      .filter((p): p is typeof p & { phase: CyclePhase } => (PHASES as readonly string[]).includes(p.phase))
      .map((p) => ({ phase: p.phase, goodPct: p.good_pct, days: p.days })),
    finding: m.finding,
  }));

export const correlationSchema: Parser<Correlation> = z
  .object({
    key: z.string(),
    status: z.string(),
    strength: z.string().nullable().catch(null),
    n: num,
    min_days: num,
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
    groups: c.groups,
    ratio: c.ratio,
    finding: c.finding,
  }));

const hubWeightSchema: Parser<HubWeight> = z
  .object({
    ready: z.boolean(),
    unit: z.string(),
    current: nullableNum,
    as_of: z.string().nullable(),
    delta_7d: nullableNum,
    delta_30d: nullableNum,
    delta_range: nullableNum,
    bmi: nullableNum,
    // A day without a weigh-in has `value: null` (the 7-day average still runs).
    points: z.array(z.object({ date: z.string(), value: nullableNum, avg7: nullableNum })),
  })
  .transform((w) => ({
    ready: w.ready,
    unit: w.unit,
    current: w.current,
    asOf: w.as_of,
    delta7d: w.delta_7d,
    delta30d: w.delta_30d,
    deltaRange: w.delta_range,
    bmi: w.bmi,
    points: w.points,
  }));

const hubVitalsSchema: Parser<HubVitals> = z
  .object({
    blood_pressure: z.object({ systolic: num, diastolic: num, readings: num }).nullable(),
    blood_sugar: z.object({ avg: num, readings: num }).nullable(),
  })
  .transform((v) => ({ bloodPressure: v.blood_pressure, bloodSugar: v.blood_sugar }));

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
