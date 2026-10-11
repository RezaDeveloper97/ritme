'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { api } from '@/shared/api';

/** GET /kpi (admin-api.md §6c, B-N9-08): aggregates only; every nullable is «no data / suppressed (k < 10)». */
const step = z.object({
  key: z.string(),
  pct: z.number().nullable(),
  count: z.number().nullable(),
  of: z.number().nullable(),
  suppressed: z.boolean(),
});
const cohort = z.object({
  week_start: z.string(),
  size: z.number().nullable(),
  suppressed: z.boolean(),
  cells: z.array(z.number().nullable()),
});
const kpiSchema = z.object({
  generated_at: z.string(),
  events_since: z.string().nullable(),
  funnel: z.object({ days: z.number(), steps: z.array(step) }),
  retention: z.object({ event: z.string(), offsets: z.array(z.number()), cohorts: z.array(cohort) }),
  engine: z
    .object({
      status: z.string().nullable(),
      mean_error_days: z.number().nullable(),
      within_2_days_pct: z.number().nullable(),
      users: z.number(),
      samples: z.number(),
      suppressed: z.boolean(),
    })
    .nullable(),
  data_quality: z.unknown().nullable(),
  ai: z.object({
    days: z.number(),
    labs_per_day: z.number().nullable(),
    ocr_correction_pct: z.number().nullable(),
    cost_per_analysis_usd: z.number().nullable(),
    assistant_msgs_per_day: z.number().nullable(),
    referral_pct: z.number().nullable(),
    suppressed: z.boolean(),
  }),
  notes: z.record(z.string(), z.string()),
});
export type Kpi = z.infer<typeof kpiSchema>;
export type KpiCohort = z.infer<typeof cohort>;

export const kpiKeys = { all: ['kpi'] as const, of: (days: number, event: string) => ['kpi', days, event] as const };

export function useKpi(days: number, event: 'any' | 'log_saved') {
  return useQuery({
    queryKey: kpiKeys.of(days, event),
    queryFn: ({ signal }) => api.get('/kpi', { schema: kpiSchema, query: { days, event }, signal }),
    staleTime: 60_000,
  });
}
