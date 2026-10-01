import { z } from 'zod';

import {
  CORRELATION_KEYS,
  type BodyReport,
  type CorrelationKey,
  type CorrelationsReport,
  type CycleLayout,
  type CycleReport,
  type PeriodReport,
  type SymptomsReport,
} from '../model/reports';
import type { Correlation } from '../model/types';
import { analysisRangeSchema, correlationSchema } from './schema';

/**
 * Boundary parsers of the detail reports (B-N3-09). Shapes:
 * backend-go/internal/analysis/{cycle,period,symptoms,correlations,body}.go and
 * their goldens. Same tolerance as the hub: an unknown enum value parses to
 * `null`, an unknown correlation pair is dropped.
 */

type Parser<T> = z.ZodType<T, z.ZodTypeDef, unknown>;

const num = z.number();
const nullableNum = z.number().nullable().catch(null);
const softEnum = <const T extends [string, ...string[]]>(values: T) => z.enum(values).nullable().catch(null);
const FLOW = ['light', 'medium', 'heavy', 'very_heavy'] as const;
const flow = softEnum([...FLOW]);
const relation = softEnum(['before_period', 'early', 'mid']);

const layoutSchema: Parser<CycleLayout> = z
  .object({
    cycle_length: num,
    ovulation_day: num,
    fertile_start_day: num,
    fertile_end_day: num,
    days: z.object({ period: num, follicular: num, fertile: num, luteal: num }),
  })
  .transform((l) => ({
    cycleLength: l.cycle_length,
    ovulationDay: l.ovulation_day,
    fertileStartDay: l.fertile_start_day,
    fertileEndDay: l.fertile_end_day,
    days: l.days,
  }));

/** `data` of `GET /analysis/cycle`. */
export const cycleReportSchema: Parser<CycleReport> = z
  .object({
    range: analysisRangeSchema,
    ready: z.boolean(),
    based_on_cycles: num,
    figo: z.object({
      applies: z.boolean(),
      cycle: z.object({ min: num, max: num }),
      period: z.object({ max: num }),
      variation: z.object({ max: num }),
    }),
    cycle_length: z.object({ median: nullableNum, status: softEnum(['normal', 'frequent', 'infrequent']) }),
    period_length: z.object({ median: nullableNum, status: softEnum(['normal', 'prolonged']) }),
    variation: z.object({
      days: nullableNum,
      max: num,
      status: z.enum(['regular', 'irregular', 'not_enough_data']).catch('not_enough_data'),
      cycles_needed: num,
    }),
    typical: layoutSchema,
    current: z.object({ start: z.string(), day: num }).nullable().catch(null),
    cycles: z.array(
      z.object({
        start: z.string(),
        end: z.string(),
        length: num,
        period_days: num,
        in_figo_range: z.boolean(),
        counted: z.boolean(),
        excluded: softEnum(['implausible', 'outlier']),
      }),
    ),
  })
  .transform((r) => ({
    range: r.range,
    ready: r.ready,
    basedOnCycles: r.based_on_cycles,
    figo: {
      applies: r.figo.applies,
      cycleMin: r.figo.cycle.min,
      cycleMax: r.figo.cycle.max,
      periodMax: r.figo.period.max,
      variationMax: r.figo.variation.max,
    },
    cycleLength: r.cycle_length,
    periodLength: r.period_length,
    variation: { days: r.variation.days, max: r.variation.max, status: r.variation.status, cyclesNeeded: r.variation.cycles_needed },
    typical: r.typical,
    current: r.current,
    cycles: r.cycles.map((c) => ({
      start: c.start,
      end: c.end,
      length: c.length,
      periodDays: c.period_days,
      inFigoRange: c.in_figo_range,
      counted: c.counted,
      excluded: c.excluded,
    })),
  }));

/** `data` of `GET /analysis/period`. */
export const periodReportSchema: Parser<PeriodReport> = z
  .object({
    range: analysisRangeSchema,
    ready: z.boolean(),
    based_on_periods: num,
    figo: z.object({ period: z.object({ max: num }) }),
    period_length: z.object({ median: nullableNum, status: softEnum(['normal', 'prolonged']) }),
    peak: z.object({ day: num, level: flow }).nullable().catch(null),
    spotting: z.object({ days: num, cycles: num, of_cycles: num }),
    average: z.array(z.object({ day: num, score: nullableNum, level: flow })),
    periods: z.array(
      z.object({
        start: z.string(),
        length: num,
        closed: z.boolean(),
        is_current: z.boolean(),
        days: z.array(z.object({ day: num, date: z.string(), flow })),
      }),
    ),
    co_symptoms: z.object({
      ready: z.boolean(),
      periods_needed: num,
      items: z.array(z.object({ key: z.string(), label: z.string(), periods: num, pct: num })),
    }),
  })
  .transform((r) => ({
    range: r.range,
    ready: r.ready,
    basedOnPeriods: r.based_on_periods,
    periodMax: r.figo.period.max,
    periodLength: r.period_length,
    peak: r.peak,
    spotting: { days: r.spotting.days, cycles: r.spotting.cycles, ofCycles: r.spotting.of_cycles },
    average: r.average,
    periods: r.periods.map((p) => ({ start: p.start, length: p.length, closed: p.closed, isCurrent: p.is_current, days: p.days })),
    coSymptoms: { ready: r.co_symptoms.ready, periodsNeeded: r.co_symptoms.periods_needed, items: r.co_symptoms.items },
  }));

/** `data` of `GET /analysis/symptoms`. */
export const symptomsReportSchema: Parser<SymptomsReport> = z
  .object({
    range: analysisRangeSchema,
    days_logged: num,
    symptom_days: num,
    top: z.array(z.object({ key: z.string(), label: z.string(), days: num, cycles: num })),
    trend: z.object({
      ready: z.boolean(),
      bucket: z.enum(['day', 'week']).catch('day'),
      keys: z.array(z.string()),
      points: z.array(z.object({ start: z.string(), values: z.array(num) })),
    }),
    pattern: z.object({
      ready: z.boolean(),
      cycles_counted: num,
      cycles_needed: num,
      typical: layoutSchema,
      items: z.array(
        z.object({
          key: z.string(),
          label: z.string(),
          cycles: num,
          strip: z.array(num),
          window: z.object({ start_day: num, end_day: num, relation, days: num }).nullable().catch(null),
        }),
      ),
    }),
    highlight: z
      .object({
        key: z.string(),
        label: z.string(),
        cycles: num,
        of_cycles: num,
        relation,
        days: nullableNum,
        start_day: num,
        end_day: num,
      })
      .nullable()
      .catch(null),
  })
  .transform((r) => ({
    range: r.range,
    daysLogged: r.days_logged,
    symptomDays: r.symptom_days,
    top: r.top,
    trend: r.trend,
    pattern: {
      ready: r.pattern.ready,
      cyclesCounted: r.pattern.cycles_counted,
      cyclesNeeded: r.pattern.cycles_needed,
      typical: r.pattern.typical,
      items: r.pattern.items.map((i) => ({
        key: i.key,
        label: i.label,
        cycles: i.cycles,
        strip: i.strip,
        window: i.window
          ? { startDay: i.window.start_day, endDay: i.window.end_day, relation: i.window.relation, days: i.window.days }
          : null,
      })),
    },
    highlight: r.highlight
      ? {
          key: r.highlight.key,
          label: r.highlight.label,
          cycles: r.highlight.cycles,
          ofCycles: r.highlight.of_cycles,
          relation: r.highlight.relation,
          days: r.highlight.days,
          startDay: r.highlight.start_day,
          endDay: r.highlight.end_day,
        }
      : null,
  }));

const isKnownPair = (c: Correlation): c is Correlation & { key: CorrelationKey } =>
  (CORRELATION_KEYS as readonly string[]).includes(c.key);

/** `data` of `GET /analysis/correlations` (the `{plus, locked, ready, data}` section plus range + `not_causal`). */
export const correlationsReportSchema: Parser<CorrelationsReport> = z
  .object({
    range: analysisRangeSchema,
    not_causal: z.boolean().catch(true),
    plus: z.boolean(),
    locked: z.boolean(),
    ready: z.boolean(),
    data: z
      .object({ days_logged: num, items: z.array(correlationSchema.nullable().catch(null)) })
      .nullable()
      .catch(null),
  })
  .transform((r) => ({
    range: r.range,
    notCausal: r.not_causal,
    plus: r.plus,
    locked: r.locked,
    ready: r.ready && !r.locked && r.data != null,
    data:
      r.locked || !r.data
        ? null
        : { daysLogged: r.data.days_logged, items: r.data.items.filter((c): c is Correlation => c != null).filter(isKnownPair) },
  }));

/** `data` of `GET /analysis/body`. */
export const bodyReportSchema: Parser<BodyReport> = z
  .object({
    range: analysisRangeSchema,
    weight: z.object({
      ready: z.boolean(),
      unit: z.string(),
      current: nullableNum,
      as_of: z.string().nullable(),
      delta_7d: nullableNum,
      delta_30d: nullableNum,
      delta_range: nullableNum,
      bmi: nullableNum,
      moving_average_days: num.catch(7),
      points: z.array(z.object({ date: z.string(), value: nullableNum, avg7: nullableNum })).catch([]),
    }),
    sleep: z.object({
      ready: z.boolean(),
      nights: num,
      avg_hours: nullableNum,
      luteal_avg_hours: nullableNum,
      by_weekday: z.array(nullableNum),
    }),
    activity: z.object({
      active_days: num,
      days: num,
      weeks: z.array(z.object({ start: z.string(), days: num, active_days: num })),
    }),
  })
  .transform((r) => ({
    range: r.range,
    weight: {
      ready: r.weight.ready,
      unit: r.weight.unit,
      current: r.weight.current,
      asOf: r.weight.as_of,
      delta7d: r.weight.delta_7d,
      delta30d: r.weight.delta_30d,
      deltaRange: r.weight.delta_range,
      bmi: r.weight.bmi,
      movingAverageDays: r.weight.moving_average_days,
      points: r.weight.points,
    },
    sleep: {
      ready: r.sleep.ready,
      nights: r.sleep.nights,
      avgHours: r.sleep.avg_hours,
      lutealAvgHours: r.sleep.luteal_avg_hours,
      byWeekday: r.sleep.by_weekday,
    },
    activity: {
      activeDays: r.activity.active_days,
      days: r.activity.days,
      weeks: r.activity.weeks.map((w) => ({ start: w.start, days: w.days, activeDays: w.active_days })),
    },
  }));
