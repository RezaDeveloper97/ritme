'use client';

import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { api, createResource, pagedSchema, translationsSchema, type QueryValue } from '@/shared/api';

/** A v6 rule {all_of, any_of, exclude} over the message fact registry (T-M10-10); `{invalid: true}` = unreadable stored value. */
const conditionSchema = z.object({ fact: z.string(), op: z.string(), value: z.unknown() });
export type RuleCondition = z.infer<typeof conditionSchema>;
export const challengeRuleSchema = z
  .object({
    all_of: z.array(conditionSchema).nullish().transform((v) => v ?? []),
    any_of: z.array(z.array(conditionSchema)).nullish().transform((v) => v ?? []),
    exclude: z.array(conditionSchema).nullish().transform((v) => v ?? []),
    invalid: z.boolean().optional(),
  })
  .nullable();
export type ChallengeRule = z.infer<typeof challengeRuleSchema>;

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
  /** v2 (B-N9-07): null for a legacy one-day challenge. Absent on the write responses of older servers. */
  program: z
    .object({
      duration_days: z.number(),
      start_rule: z.string(),
      starts_on: z.string().nullable(),
      reminder_time: z.string().nullable(),
      modes_include: z.array(z.string()),
      modes_exclude: z.array(z.string()),
      phases_include: z.array(z.string()),
      phases_exclude: z.array(z.string()),
      reward_badge_code: z.string().nullable(),
      reward_badge_title: translationsSchema.nullable(),
      reward_plus_days: z.number(),
      completion_min_days: z.number().nullable(),
      // T-M10-10
      eligibility: challengeRuleSchema.optional().transform((v) => v ?? null),
      contraindications: challengeRuleSchema.optional().transform((v) => v ?? null),
      start_phase: z.string().nullish().transform((v) => v ?? null),
    })
    .nullable()
    .optional(),
  tasks: z
    .array(z.object({ id: z.number(), day_from: z.number(), day_to: z.number(), title: translationsSchema, sort_order: z.number() }))
    .optional(),
  stats: z.object({ participants: z.number(), completed: z.number(), completion_rate: z.number() }).nullable().optional(),
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

/** GET /challenges/:id/stats — participants, completion, rewards granted and users per day (B-N9-07). */
const challengeStatsSchema = z.object({
  challenge_id: z.number(),
  stats: z.object({
    participants: z.number(),
    completed: z.number(),
    completion_rate: z.number(),
    active: z.number(),
    rewards_granted: z.number(),
    plus_days_granted: z.number(),
    per_day: z.array(z.object({ day: z.number(), users: z.number() })),
    /** Day-by-day drop-off of the runs (T-M10-10); rates are percent of the runs that reached the day, null when none did. */
    drop_off: z
      .array(
        z.object({
          day: z.number(),
          reached: z.number(),
          retained: z.number(),
          dropped: z.number(),
          retained_rate: z.number().nullable(),
          drop_off_rate: z.number().nullable(),
        }),
      )
      .nullish()
      .transform((v) => v ?? []),
  }),
});
export type ChallengeStats = z.infer<typeof challengeStatsSchema>['stats'];

export function useChallengeStats(id: number | null) {
  return useQuery({
    queryKey: ['challenge-stats', id],
    enabled: id !== null,
    queryFn: ({ signal }) => api.get(`/challenges/${id}/stats`, { schema: challengeStatsSchema, signal }),
  });
}
