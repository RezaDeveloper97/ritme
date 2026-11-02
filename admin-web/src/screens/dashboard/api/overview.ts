'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { api } from '@/shared/api';

/** GET /overview (admin-api.md §6b): the latest aggregate snapshot. Every nullable is "not available / suppressed". */
const bucket = z.object({ key: z.string(), pct: z.number().nullable(), count: z.number().nullable(), suppressed: z.boolean() });
const distribution = z.object({ day: z.string().nullable().optional(), of: z.number().nullable(), suppressed: z.boolean(), buckets: z.array(bucket) });
/** An alert of the metric alert rules (T-M10-31, D-180). */
const alert = z.object({
  key: z.string(),
  severity: z.string(),
  text: z.string(),
  subject: z.string(),
  state: z.string(),
  day: z.string(),
  value: z.number(),
  threshold: z.number(),
  opened_at: z.string(),
  resolved_at: z.string().nullable().optional(),
});
const value = z.object({ value: z.number().nullable(), prev: z.number().nullable() });
const metricsSchema = z.object({
  dau: value,
  mau: value,
  stickiness_pct: value,
  signups_today: value,
  plus_active: value,
  trial_to_paid: z.object({
    pct: z.number().nullable(),
    count: z.number(),
    of: z.number().nullable(),
    suppressed: z.boolean(),
  }),
  mrr_toman: z.number().nullable(),
  modes: z.array(z.object({ mode: z.string(), count: z.number().nullable(), suppressed: z.boolean() })),
  engine: z.object({
    status: z.string().nullable(),
    mean_error_days: z.number().nullable(),
    within_2_days_pct: z.number().nullable(),
    users: z.number(),
    samples: z.number(),
    suppressed: z.boolean(),
    confidence: distribution.nullable().optional(),
  }),
  alerts: z.array(alert),
  resolved_alerts: z.array(alert).optional().default([]),
  kpi: z.object({ distributions: z.object({ data_quality: distribution.nullable().optional() }).optional() }).nullable().optional(),
  okr: z
    .object({ objectives: z.number(), key_results: z.number(), on_track: z.number(), at_risk: z.number(), done: z.number(), no_data: z.number() })
    .nullable()
    .optional(),
  series: z.array(z.object({ date: z.string(), dau: z.number(), loggers: z.number(), signups: z.number() })),
});
const overviewSchema = z.object({
  day: z.string().nullable(),
  computed_at: z.string().nullable(),
  metrics: metricsSchema.nullable(),
});
export type Overview = z.infer<typeof overviewSchema>;
export type OverviewMetrics = z.infer<typeof metricsSchema>;
export type Distribution = z.infer<typeof distribution>;
export type SeriesPoint = OverviewMetrics['series'][number];

export const overviewKeys = { all: ['overview'] as const };

export function useOverview() {
  return useQuery({
    queryKey: overviewKeys.all,
    queryFn: ({ signal }) => api.get('/overview', { schema: overviewSchema, signal }),
    refetchInterval: 10 * 60 * 1000,
  });
}
