'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api } from '@/shared/api';

/** Versioned clinical thresholds with two medical approvals (admin-api.md §23, B-N9-11). */

export const CONFIG_GROUPS = ['engine', 'vitals', 'pregnancy', 'lab'] as const;
export type ConfigGroup = (typeof CONFIG_GROUPS)[number];

const personSchema = z.object({ id: z.number(), name: z.string().nullable() });

const versionSchema = z.object({
  id: z.number().nullable(),
  version: z.number(),
  value: z.record(z.string(), z.number()),
  source: z.string(),
  note: z.string(),
  effective_from: z.string().nullable(),
  status: z.enum(['pending', 'approved', 'rejected']),
  proposed_by: personSchema.nullable(),
  approvals: z.array(personSchema.extend({ at: z.string().nullable() })),
  required_approvals: z.number(),
  decided_at: z.string().nullable(),
  created_at: z.string().nullable(),
});
export type ConfigVersion = z.infer<typeof versionSchema>;

const fieldSchema = z.object({ name: z.string(), default: z.number(), min: z.number(), max: z.number() });
export type ConfigField = z.infer<typeof fieldSchema>;

const keySchema = z.object({
  key: z.string(),
  group: z.enum(CONFIG_GROUPS),
  unit: z.string(),
  fields: z.array(fieldSchema),
  current: versionSchema,
  upcoming: versionSchema.nullable(),
  pending: versionSchema.nullable(),
});
export type ConfigKey = z.infer<typeof keySchema>;

const listSchema = z.object({
  groups: z.array(z.string()),
  keys: z.array(keySchema),
  can_approve: z.boolean(),
  required_approvals: z.number(),
  today: z.string(),
});
export type ConfigList = z.infer<typeof listSchema>;

const LIST_KEY = ['clinical-config'] as const;

export function useClinicalConfig() {
  return useQuery({
    queryKey: LIST_KEY,
    queryFn: ({ signal }) => api.get('/clinical-config', { schema: listSchema, signal }),
  });
}

export interface ProposalInput {
  key: string;
  value: Record<string, number>;
  effective_from: string;
  source: string;
  note: string;
}

export function usePropose() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ key, ...body }: ProposalInput) => api.post(`/clinical-config/${encodeURIComponent(key)}/proposals`, body),
    onSuccess: () => client.invalidateQueries({ queryKey: LIST_KEY }),
  });
}

export function useDecide() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, action }: { id: number; action: 'approve' | 'reject' }) =>
      api.post(`/clinical-config/proposals/${id}/${action}`),
    onSuccess: () => client.invalidateQueries({ queryKey: LIST_KEY }),
  });
}
