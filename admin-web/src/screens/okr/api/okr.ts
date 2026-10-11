'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api } from '@/shared/api';

/** OKR (admin-api.md §6d, B-N9-09): aggregates only; progress is computed by the API from the live metrics. */
export const STATUSES = ['on_track', 'at_risk', 'done', 'no_data'] as const;
export type KrStatus = (typeof STATUSES)[number];

const keyResult = z.object({
  id: z.number(),
  objective_id: z.number(),
  title: z.string(),
  metric_key: z.string().nullable(),
  source: z.enum(['live', 'manual']),
  unit: z.string(),
  baseline: z.number(),
  target: z.number(),
  manual_value: z.number().nullable(),
  current: z.number().nullable(),
  progress_pct: z.number().nullable(),
  status: z.enum(STATUSES),
  owner: z.string().nullable(),
});
const objective = z.object({
  id: z.number(),
  title: z.string(),
  owner: z.string().nullable(),
  period: z.string(),
  starts_on: z.string(),
  ends_on: z.string(),
  sort_order: z.number(),
  progress_pct: z.number().nullable(),
  status: z.enum(STATUSES),
  key_results: z.array(keyResult),
});
const okrSchema = z.object({
  objectives: z.array(objective),
  periods: z.array(z.string()),
  summary: z
    .object({ objectives: z.number(), key_results: z.number(), on_track: z.number(), at_risk: z.number(), done: z.number(), no_data: z.number() })
    .nullable(),
  metrics: z.array(z.object({ key: z.string(), unit: z.string(), current: z.number().nullable() })),
  units: z.array(z.string()),
  notes: z.object({ live: z.string() }),
});
export type Okr = z.infer<typeof okrSchema>;
export type Objective = z.infer<typeof objective>;
export type KeyResult = z.infer<typeof keyResult>;

export interface ObjectiveInput {
  title: string;
  owner: string;
  period: string;
  starts_on: string;
  ends_on: string;
}
export interface KeyResultInput {
  title: string;
  metric_key: string;
  unit: string;
  baseline: string;
  target: string;
  manual_value: string;
  owner: string;
}

const KEY = ['okr'] as const;

export function useOkr(period: string) {
  return useQuery({
    queryKey: [...KEY, period],
    queryFn: ({ signal }) => api.get('/okr', { schema: okrSchema, query: period ? { period } : undefined, signal }),
    staleTime: 30_000,
  });
}

export function useSaveObjective() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id?: number; input: ObjectiveInput }) =>
      id ? api.put(`/okr/objectives/${id}`, input) : api.post('/okr/objectives', input),
    onSuccess: () => client.invalidateQueries({ queryKey: KEY }),
  });
}

export function useDeleteObjective() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => api.delete(`/okr/objectives/${id}`),
    onSuccess: () => client.invalidateQueries({ queryKey: KEY }),
  });
}

export function useSaveKeyResult() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ objectiveId, id, input }: { objectiveId: number; id?: number; input: KeyResultInput }) =>
      id ? api.put(`/okr/key-results/${id}`, input) : api.post(`/okr/objectives/${objectiveId}/key-results`, input),
    onSuccess: () => client.invalidateQueries({ queryKey: KEY }),
  });
}

export function useDeleteKeyResult() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => api.delete(`/okr/key-results/${id}`),
    onSuccess: () => client.invalidateQueries({ queryKey: KEY }),
  });
}
