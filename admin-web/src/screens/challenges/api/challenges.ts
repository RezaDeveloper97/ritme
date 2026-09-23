'use client';

import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { api, createResource, pagedSchema, translationsSchema, type QueryValue } from '@/shared/api';

/** `Challenge` (admin-api.md §11). */
export const challengeSchema = z.object({
  id: z.number(),
  slug: z.string().nullable(),
  title: translationsSchema,
  description: translationsSchema,
  cycle_day_from: z.number().nullable(),
  cycle_day_to: z.number().nullable(),
  category: z.string().nullable(),
  is_active: z.boolean(),
  sort_order: z.number(),
});
export type Challenge = z.infer<typeof challengeSchema>;

export const challengesApi = createResource({
  key: 'challenges',
  path: '/challenges',
  list: pagedSchema(challengeSchema, {
    filters: z
      .object({ q: z.string(), status: z.string(), cycle_day: z.number().nullable() })
      .partial()
      .optional(),
  }),
  detail: z.object({ challenge: challengeSchema }),
  options: z.object({ max_cycle_day: z.number() }),
  alsoInvalidate: [['dashboard'], ['challenge-completions']],
});

/** GET /challenge-completions — the report (admin-api.md §11). */
const completionSchema = z.object({
  id: z.number(),
  completion_date: z.string().nullable(),
  completed_at: z.string().nullable(),
  user: z.object({ id: z.number(), name: z.string().nullable(), mobile: z.string().nullable() }),
  challenge: z.object({ id: z.number(), title: translationsSchema }),
});
export type Completion = z.infer<typeof completionSchema>;

const completionsSchema = pagedSchema(completionSchema, {
  stats: z.object({ total: z.number(), users: z.number(), today: z.number() }),
  per_challenge: z.array(
    z.object({ challenge_id: z.number(), title: translationsSchema, completions: z.number(), users: z.number() }),
  ),
  challenges: z.array(z.object({ id: z.number(), title: translationsSchema })),
});
export type CompletionsReport = z.infer<typeof completionsSchema>;
export type PerChallenge = CompletionsReport['per_challenge'][number];

export const completionKeys = {
  all: ['challenge-completions'] as const,
  list: (q: Record<string, QueryValue>) => [...completionKeys.all, 'list', q] as const,
};

export function useCompletions(query: Record<string, QueryValue>) {
  return useQuery({
    queryKey: completionKeys.list(query),
    queryFn: ({ signal }) => api.get('/challenge-completions', { query, schema: completionsSchema, signal }),
    placeholderData: keepPreviousData,
  });
}
