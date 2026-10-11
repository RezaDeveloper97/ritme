'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api } from '@/shared/api';

/** Safety & clinical content (admin-api.md §24, B-N9-10): aggregates and review metadata only. */

export const SOURCES = ['message', 'threshold', 'lesson', 'assistant', 'catalog'] as const;
export type Source = (typeof SOURCES)[number];

export const SIGNALS = [
  'bp_crisis',
  'glucose_urgent_low',
  'epds_q10',
  'reduced_kicks',
  'contractions_511',
  'postpartum_heavy_bleeding',
] as const;

const signalSchema = z.object({ key: z.string(), count: z.number(), previous: z.number(), delta: z.number() });
export type Signal = z.infer<typeof signalSchema>;

const claimSchema = z.object({ id: z.number(), name: z.string(), at: z.string().nullable() });

const itemSchema = z.object({
  source: z.enum(SOURCES),
  id: z.number(),
  title: z.string(),
  ref: z.string(),
  status: z.enum(['pending', 'changed', 'partially_approved']),
  approvals: z.number(),
  required_approvals: z.number(),
  items: z.number(),
  submitted_by: z.string().nullable(),
  submitted_at: z.string().nullable(),
  age_hours: z.number(),
  overdue: z.boolean(),
  claimed_by: claimSchema.nullable(),
});
export type ReviewItem = z.infer<typeof itemSchema>;

const safetySchema = z.object({
  window_days: z.number(),
  today: z.string(),
  signals: z.array(signalSchema),
  queue: z.object({
    pending: z.number(),
    overdue: z.number(),
    sla_hours: z.number(),
    by_source: z.record(z.string(), z.number()),
    items: z.array(itemSchema),
  }),
  checks: z.object({
    modal_latency_p95_ms: z.number().nullable(),
    call_115_ctr: z.number().nullable(),
    text_parity: z.object({ ok: z.boolean(), unapproved: z.number(), groups: z.array(z.string()) }),
    scenarios: z
      .object({ ran_at: z.string().nullable(), passed: z.number(), total: z.number(), failed: z.array(z.string()) })
      .nullable(),
  }),
});
export type SafetyData = z.infer<typeof safetySchema>;

const KEY = ['safety'] as const;

export function useSafety() {
  return useQuery({
    queryKey: KEY,
    queryFn: ({ signal }) => api.get('/safety', { schema: safetySchema, signal }),
  });
}

export function useClaim() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ source, id, action }: { source: Source; id: number; action: 'claim' | 'release' }) =>
      api.post(`/safety/queue/${source}/${id}/${action}`),
    onSuccess: () => client.invalidateQueries({ queryKey: KEY }),
  });
}
