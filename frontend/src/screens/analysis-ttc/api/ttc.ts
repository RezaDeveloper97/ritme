'use client';

import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { analysisKeys, phraseSchema, sectionSchema } from '@/entities/analysis';
import { type ApiEnvelope, apiClient, getApiErrorStatus } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type {
  BbtPoint,
  FertilityCycle,
  LhTest,
  TimingDay,
  TtcBbt,
  TtcCycleSummary,
  TtcHub,
  TtcLh,
  TtcLuteal,
  TtcMucus,
  TtcRegularity,
  TtcTiming,
  TtcTrying,
} from '../model/types';

/*
 * Boundary parsers + reads of GET /analysis/ttc and /analysis/fertility
 * (B-N3-11; zod at the boundary, CLAUDE.md §10). Unknown enum values degrade to
 * null instead of failing the screen. Never log what these return (§11).
 */

type Parser<T> = z.ZodType<T, z.ZodTypeDef, unknown>;

const nullableNum = z.number().nullable().catch(null);
const softEnum = <const T extends [string, ...string[]]>(values: T) => z.enum(values).nullable().catch(null);
/** A decimal string ("36.55") → number; anything unreadable → null. */
const temp = z.union([z.string(), z.number()]).transform((v) => {
  const n = typeof v === 'number' ? v : Number.parseFloat(v);
  return Number.isFinite(n) ? n : null;
});
const nullableTemp = temp.nullable().catch(null);
const sourceEnum = softEnum(['bbt', 'lh', 'estimate']);
const lhEnum = z.enum(['negative', 'faint', 'positive']);

const pointsSchema: Parser<BbtPoint[]> = z
  .array(z.object({ day: z.number(), value: temp }))
  .transform((ps) => ps.filter((p): p is BbtPoint => p.value !== null));

const testsSchema: Parser<LhTest[]> = z
  .array(z.object({ day: z.number(), value: lhEnum.nullable().catch(null) }))
  .transform((ts) => ts.filter((t): t is LhTest => t.value !== null));

const tryingSchema: Parser<TtcTrying> = z
  .object({
    since: z.string().nullable(),
    cycles: z.number(),
    months: z.number(),
    confirmed_cycles: z.number(),
    age: nullableNum,
    referral: z.object({
      age_band: z.enum(['under_35', 'from_35', 'unknown']).catch('unknown'),
      threshold_months: z.number(),
      due: z.boolean(),
    }),
    summary: phraseSchema,
    advice: phraseSchema,
  })
  .transform((t) => ({
    since: t.since,
    cycles: t.cycles,
    months: t.months,
    confirmedCycles: t.confirmed_cycles,
    age: t.age,
    referral: { ageBand: t.referral.age_band, thresholdMonths: t.referral.threshold_months, due: t.referral.due },
    summary: t.summary,
    advice: t.advice,
  }));

const bbtSchema: Parser<TtcBbt> = z
  .object({
    cycle_start: z.string(),
    points: pointsSchema,
    coverline: nullableTemp,
    shift_day: nullableNum,
    ovulation_day: nullableNum,
    confirmed: z.boolean(),
  })
  .transform((b) => ({
    cycleStart: b.cycle_start,
    points: b.points,
    coverline: b.coverline,
    shiftDay: b.shift_day,
    ovulationDay: b.ovulation_day,
    confirmed: b.confirmed,
  }));

const lhSchema: Parser<TtcLh> = z
  .object({ cycle_start: z.string(), tests: testsSchema, positive_day: nullableNum, text: phraseSchema })
  .transform((l) => ({ cycleStart: l.cycle_start, tests: l.tests, positiveDay: l.positive_day, text: l.text }));

const timingDaySchema: Parser<TimingDay> = z.object({
  day: z.number(),
  phase: z.enum(['period', 'fertile', 'ovulation', 'other']).catch('other'),
  intercourse: z.boolean(),
  future: z.boolean(),
});

const timingSchema: Parser<TtcTiming> = z
  .object({
    cycle_start: z.string(),
    window_from: z.number(),
    window_to: z.number(),
    window_days: z.number(),
    window_source: sourceEnum,
    in_window: z.number(),
    days: z.array(timingDaySchema),
  })
  .transform((t) => ({
    cycleStart: t.cycle_start,
    windowFrom: t.window_from,
    windowTo: t.window_to,
    windowDays: t.window_days,
    windowSource: t.window_source,
    inWindow: t.in_window,
    days: t.days,
  }));

const mucusSchema: Parser<TtcMucus> = z
  .object({ cycles_with_egg_white: z.number(), cycles_analysed: z.number(), text: phraseSchema })
  .transform((m) => ({ cyclesWithEggWhite: m.cycles_with_egg_white, cyclesAnalysed: m.cycles_analysed, text: m.text }));

const lutealStatus = softEnum(['short', 'normal', 'long']);

const lutealSchema: Parser<TtcLuteal> = z
  .object({ days: nullableNum, status: lutealStatus, cycles: z.number().catch(0) })
  .transform((l) => ({ days: l.days, status: l.status, cycles: l.cycles }));

const regularitySchema: Parser<TtcRegularity> = z.object({
  cycles: z.array(z.object({ index: z.number(), start: z.string(), length: z.number(), current: z.boolean() })),
  median: nullableNum,
  variation: nullableNum,
  status: softEnum(['regular', 'irregular', 'not_enough_data']),
});

const cycleSummarySchema: Parser<TtcCycleSummary> = z
  .object({
    index: z.number(),
    start: z.string(),
    end: z.string(),
    current: z.boolean(),
    length: nullableNum,
    ovulation_day: nullableNum,
    ovulation_source: sourceEnum,
    lh_day: nullableNum,
    luteal_days: nullableNum,
  })
  .transform((c) => ({
    index: c.index,
    start: c.start,
    end: c.end,
    current: c.current,
    length: c.length,
    ovulationDay: c.ovulation_day,
    ovulationSource: c.ovulation_source,
    lhDay: c.lh_day,
    lutealDays: c.luteal_days,
  }));

export const ttcHubSchema: Parser<TtcHub> = z.object({
  trying: tryingSchema,
  bbt: sectionSchema(bbtSchema),
  lh: sectionSchema(lhSchema),
  timing: sectionSchema(timingSchema),
  mucus: sectionSchema(mucusSchema),
  luteal: sectionSchema(lutealSchema),
  regularity: sectionSchema(regularitySchema),
  cycles: z.array(cycleSummarySchema),
});

export const fertilityCycleSchema: Parser<FertilityCycle> = z
  .object({
    cycle: z.object({
      index: z.number(),
      total: z.number(),
      start: z.string(),
      end: z.string(),
      current: z.boolean(),
      length: nullableNum,
      days: z.number(),
      span: z.number(),
    }),
    prev_start: z.string().nullable(),
    next_start: z.string().nullable(),
    chart: z.object({
      points: z
        .array(z.object({ day: z.number(), value: temp }))
        .transform((ps) => ps.filter((p): p is BbtPoint => p.value !== null)),
      coverline: nullableTemp,
      shift_day: nullableNum,
      high_days: z.array(z.number()).catch([]),
      confirmed: z.boolean(),
    }),
    period_days: z.number(),
    fertile_window: z.object({ from: z.number(), to: z.number() }).nullable().catch(null),
    ovulation: z.object({ day: nullableNum, source: sourceEnum, confirmed: z.boolean() }),
    lh: z.object({ tests: testsSchema, positive_day: nullableNum, days_before_ovulation: nullableNum }),
    intercourse_days: z.array(z.number()).catch([]),
    luteal: sectionSchema(z.object({ days: nullableNum, status: lutealStatus })),
    explanation: phraseSchema,
    tip: phraseSchema,
  })
  .transform((f) => ({
    cycle: f.cycle,
    prevStart: f.prev_start,
    nextStart: f.next_start,
    chart: {
      points: f.chart.points,
      coverline: f.chart.coverline,
      shiftDay: f.chart.shift_day,
      highDays: f.chart.high_days,
      confirmed: f.chart.confirmed,
    },
    periodDays: f.period_days,
    fertileWindow: f.fertile_window,
    ovulation: f.ovulation,
    lh: { tests: f.lh.tests, positiveDay: f.lh.positive_day, daysBeforeOvulation: f.lh.days_before_ovulation },
    intercourseDays: f.intercourse_days,
    luteal: f.luteal,
    explanation: f.explanation,
    tip: f.tip,
  }));

export const ttcKeys = {
  hub: () => [...analysisKeys.all, 'ttc'] as const,
  fertility: (cycle: string | null) => [...analysisKeys.all, 'fertility', cycle ?? 'latest'] as const,
};

/** GET /analysis/ttc (An_Hub_TTC). */
export function useTtcHub() {
  return useQuery({
    queryKey: ttcKeys.hub(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/analysis/ttc');
      return ttcHubSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: false,
  });
}

/**
 * GET /analysis/fertility?cycle= (An_Fertility). `null` data = the user has no
 * cycle yet (the API answers 404) — the screen shows its empty state.
 */
export function useFertilityCycle(cycle: string | null) {
  return useQuery({
    queryKey: ttcKeys.fertility(cycle),
    queryFn: async (): Promise<FertilityCycle | null> => {
      try {
        const { data } = await apiClient.get<ApiEnvelope<unknown>>('/analysis/fertility', {
          params: cycle ? { cycle } : undefined,
        });
        return fertilityCycleSchema.parse(data.data);
      } catch (err) {
        if (getApiErrorStatus(err) === 404) return null;
        throw err;
      }
    },
    enabled: isAuthenticated(),
    placeholderData: keepPreviousData,
    staleTime: 60_000,
    retry: false,
  });
}
