'use client';

import { z } from 'zod';

import { createResource, optionSchema, pagedSchema, stringListSchema, translationsSchema } from '@/shared/api';

/** `Recommendation` (admin-api.md §11). */
export const recommendationSchema = z.object({
  id: z.number(),
  key: z.string().nullable(),
  type: z.string(),
  title: translationsSchema,
  text: translationsSchema,
  cycle_phase: z.string().nullable(),
  cycle_subphases: stringListSchema,
  symptom_trigger: z.string().nullable(),
  is_active: z.boolean(),
  sort_order: z.number(),
});
export type Recommendation = z.infer<typeof recommendationSchema>;

const optionsSchema = z.object({
  phases: z.array(optionSchema),
  subphases: z.array(optionSchema),
  subphase_phases: z.record(z.string(), z.string()),
  types: z.array(optionSchema),
  triggers: z.array(optionSchema),
});
export type RecommendationOptions = z.infer<typeof optionsSchema>;

export const recommendationsApi = createResource({
  key: 'recommendations',
  path: '/recommendations',
  list: pagedSchema(recommendationSchema, {
    filters: z.object({ phase: z.string().nullable(), type: z.string().nullable() }).partial().optional(),
  }),
  detail: z.object({ recommendation: recommendationSchema }),
  options: optionsSchema,
});
