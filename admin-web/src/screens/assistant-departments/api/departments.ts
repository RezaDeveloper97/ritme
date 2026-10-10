'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api, translationsSchema } from '@/shared/api';

/** Assistant departments editor (admin-api.md §19c, B-N7-08) — super admins only. */

const listsSchema = z.preprocess((v) => {
  const out: Record<string, string[]> = {};
  if (v && typeof v === 'object' && !Array.isArray(v)) {
    for (const [code, items] of Object.entries(v)) if (Array.isArray(items)) out[code] = items.filter((i): i is string => typeof i === 'string');
  }
  return out;
}, z.record(z.string(), z.array(z.string())));

export const departmentSchema = z.object({
  id: z.number(),
  code: z.string(),
  icon: z.string().nullable(),
  sort_order: z.number(),
  is_active: z.boolean(),
  name: translationsSchema,
  tagline: translationsSchema,
  intro: translationsSchema,
  does_not_do: listsSchema,
  quick_replies: listsSchema,
  system_prompt: z.string(),
  guardrails: z.string().nullable(),
  needs_review: z.boolean(),
  updated_at: z.string().nullable().optional(),
});
export type Department = z.infer<typeof departmentSchema>;

const listSchema = z.object({
  items: z.array(departmentSchema),
  limits: z.record(z.string(), z.number()),
});

const KEY = ['assistant-departments'] as const;

export function useDepartments() {
  return useQuery({ queryKey: KEY, queryFn: ({ signal }) => api.get('/assistant/departments', { schema: listSchema, signal }) });
}

export function useSaveDepartment(id: number | null) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (body: unknown) => (id === null ? api.post('/assistant/departments', body) : api.put(`/assistant/departments/${id}`, body)),
    onSuccess: () => client.invalidateQueries({ queryKey: KEY }),
  });
}

export function useApproveDepartment() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => api.post(`/assistant/departments/${id}/approve`),
    onSuccess: () => client.invalidateQueries({ queryKey: KEY }),
  });
}
