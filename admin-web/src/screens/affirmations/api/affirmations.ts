'use client';

import { z } from 'zod';

import { createResource, optionSchema, pagedSchema, translationsSchema } from '@/shared/api';

/** `Affirmation` (admin-api.md §11). */
export const affirmationSchema = z.object({
  id: z.number(),
  text: translationsSchema,
  cycle_phase: z.string().nullable(),
  is_active: z.boolean(),
  sort_order: z.number(),
});
export type Affirmation = z.infer<typeof affirmationSchema>;

export const affirmationsApi = createResource({
  key: 'affirmations',
  path: '/affirmations',
  list: pagedSchema(affirmationSchema, {}),
  detail: z.object({ affirmation: affirmationSchema }),
  options: z.object({ phases: z.array(optionSchema) }),
  alsoInvalidate: [['dashboard']],
});
