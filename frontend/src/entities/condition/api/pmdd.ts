'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import {
  PMDD_MAX,
  PMDD_MIN,
  PMDD_PATTERNS,
  type PmddAlert,
  type PmddChart,
  type PmddCycle,
  type PmddDay,
  type PmddItem,
  type PmddWindow,
} from '../model/pmdd';
import { conditionKeys } from './keys';

/*
 * `/api/v1/conditions/pmdd/*` + `GET /catalog/pmdd_items` (CB-COND-01, golden backend-go/contract/golden/conditions).
 * Lenient parsers: a malformed row is dropped, never the screen. Never log a payload, a response or an error body.
 */

const isoDate = z.string().regex(/^\d{4}-\d{2}-\d{2}$/);
const text = z.string().trim().min(1).nullable().catch(null);
const score = z.number().int().min(PMDD_MIN).max(PMDD_MAX);

function rows<S extends z.ZodTypeAny>(list: unknown[], schema: S): z.output<S>[] {
  return list.flatMap((raw) => {
    const parsed = schema.safeParse(raw);
    return parsed.success ? [parsed.data] : [];
  });
}

function scoresOf(raw: unknown): Record<string, number> {
  if (!raw || typeof raw !== 'object') return {};
  return Object.fromEntries(
    Object.entries(raw as Record<string, unknown>).filter(
      (e): e is [string, number] => score.safeParse(e[1]).success,
    ),
  );
}

export const pmddItemsSchema = z
  .object({ items: z.array(z.unknown()).catch([]) })
  .catch({ items: [] })
  .transform((g): PmddItem[] =>
    rows(g.items, z.object({ code: z.string().min(1), title: z.string().trim().min(1) })).map((i) => ({
      code: i.code,
      title: i.title,
    })),
  );

export const pmddDaySchema = z
  .object({ date: isoDate, scores: z.unknown(), mean: z.number().nullable().catch(null) })
  .transform((d): PmddDay => ({ date: d.date, scores: scoresOf(d.scores), mean: d.mean }));

const windowSchema = z
  .object({ from: isoDate, to: isoDate })
  .nullable()
  .catch(null)
  .transform((w): PmddWindow | null => w);

const chartDaySchema = z.object({
  date: isoDate,
  cycle_day: z.number().int().min(1),
  in_period: z.boolean().catch(false),
  late_luteal: z.boolean().catch(false),
  mean: z.number(),
});

const cycleSchema = z
  .object({
    start: isoDate,
    end: isoDate,
    closed: z.boolean().catch(false),
    complete: z.boolean().catch(false),
    period: windowSchema,
    late_luteal: windowSchema,
    follicular: windowSchema,
    luteal_mean: z.number().nullable().catch(null),
    follicular_mean: z.number().nullable().catch(null),
    days: z.array(z.unknown()).catch([]),
  })
  .transform(
    (c): PmddCycle => ({
      start: c.start,
      end: c.end,
      closed: c.closed,
      complete: c.complete,
      period: c.period,
      lateLuteal: c.late_luteal,
      follicular: c.follicular,
      lutealMean: c.luteal_mean,
      follicularMean: c.follicular_mean,
      days: rows(c.days, chartDaySchema).map((d) => ({
        date: d.date,
        cycleDay: d.cycle_day,
        inPeriod: d.in_period,
        lateLuteal: d.late_luteal,
        mean: d.mean,
      })),
    }),
  );

const hotlineSchema = z.object({ number: z.string().regex(/^[0-9+]{2,15}$/), label: text });

const alertSchema = z
  .object({
    code: z.string().min(1),
    title: text,
    body: text,
    meta: z.record(z.string(), z.unknown()).nullable().catch(null),
  })
  .transform((a): PmddAlert => {
    const severity = a.meta?.severity;
    return {
      code: a.code,
      title: a.title,
      body: a.body,
      severity: severity === 'urgent' || severity === 'caution' ? severity : 'info',
      hotlines: Array.isArray(a.meta?.hotlines) ? rows(a.meta.hotlines, hotlineSchema) : [],
    };
  });

export const pmddChartSchema = z
  .object({
    required_cycles: z.number().int().catch(2),
    complete_cycles: z.number().int().catch(0),
    ready: z.boolean().catch(false),
    pattern: z.enum(PMDD_PATTERNS).catch('not_enough_data'),
    cycles: z.array(z.unknown()).catch([]),
    alerts: z.array(z.unknown()).catch([]),
  })
  .transform(
    (c): PmddChart => ({
      requiredCycles: c.required_cycles,
      completeCycles: c.complete_cycles,
      ready: c.ready,
      pattern: c.pattern,
      cycles: rows(c.cycles, cycleSchema),
      alerts: rows(c.alerts, alertSchema),
    }),
  );

const pmddKeys = {
  all: () => [...conditionKeys.all, 'pmdd'] as const,
  items: (locale: string) => [...pmddKeys.all(), 'items', locale] as const,
  day: (date: string) => [...pmddKeys.all(), 'day', date] as const,
  chart: (locale: string) => [...pmddKeys.all(), 'chart', locale] as const,
};

/** GET /catalog/pmdd_items — the questions in catalog order (content, not health data). */
export function usePmddItems(locale: string) {
  return useQuery<PmddItem[]>({
    queryKey: pmddKeys.items(locale),
    queryFn: async () => pmddItemsSchema.parse((await apiClient.get<ApiEnvelope<unknown>>('/catalog/pmdd_items')).data.data),
    enabled: isAuthenticated(),
    staleTime: 10 * 60_000,
    retry: 1,
  });
}

/** GET /conditions/pmdd/{date} — that day's ratings (empty when nothing was rated). */
export function usePmddDay(date: string) {
  return useQuery<PmddDay>({
    queryKey: pmddKeys.day(date),
    queryFn: async () =>
      pmddDaySchema.parse((await apiClient.get<ApiEnvelope<unknown>>(`/conditions/pmdd/${date}`)).data.data),
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

/** GET /conditions/pmdd/chart — the current cycle and up to two closed ones, the pattern and the alerts. */
export function usePmddChart(locale: string) {
  return useQuery<PmddChart>({
    queryKey: pmddKeys.chart(locale),
    queryFn: async () =>
      pmddChartSchema.parse((await apiClient.get<ApiEnvelope<unknown>>('/conditions/pmdd/chart')).data.data),
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

/** PUT /conditions/pmdd/{date} — needs the `pmdd` enrolment; refreshes the day and the chart. */
export function useSavePmddDay() {
  const queryClient = useQueryClient();
  return useMutation<PmddDay, unknown, { date: string; scores: Record<string, number> }>({
    mutationFn: async ({ date, scores }) =>
      pmddDaySchema.parse(
        (await apiClient.put<ApiEnvelope<unknown>>(`/conditions/pmdd/${date}`, { scores })).data.data,
      ),
    onSuccess: (day) => {
      queryClient.setQueryData(pmddKeys.day(day.date), day);
      void queryClient.invalidateQueries({ queryKey: [...pmddKeys.all(), 'chart'] });
    },
  });
}
