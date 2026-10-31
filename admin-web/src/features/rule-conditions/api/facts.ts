'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { api } from '@/shared/api';

/** GET /messages/facts (T-M10-03): the fact registry every condition picks from. */
export const FACT_TYPES = ['int', 'float', 'bool', 'enum', 'set', 'date'] as const;
export type FactType = (typeof FACT_TYPES)[number];

export const factDefSchema = z.object({
  name: z.string(),
  type: z.enum(FACT_TYPES),
  enum: z.array(z.string()).nullish().transform((v) => v ?? null),
  unit: z.string().nullish().transform((v) => v ?? null),
  group: z.string().default(''),
  description: z.record(z.string(), z.string()).nullish().transform((v) => v ?? {}),
  member_safe: z.boolean().default(false),
  pending: z.string().nullish().transform((v) => v ?? null),
});
export type FactDef = z.infer<typeof factDefSchema>;

const factsSchema = z.object({
  facts: z.array(factDefSchema),
  groups: z.array(z.string()).default([]),
  fertility_facts: z.array(z.string()).default([]),
});
export type FactRegistry = z.infer<typeof factsSchema>;

export function useFacts() {
  return useQuery({
    queryKey: ['message-rules', 'facts'],
    queryFn: ({ signal }) => api.get('/messages/facts', { schema: factsSchema, signal }),
    staleTime: 10 * 60_000,
  });
}
