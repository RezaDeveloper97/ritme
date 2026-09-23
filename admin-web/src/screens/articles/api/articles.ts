'use client';

import { z } from 'zod';

import { createResource, pagedSchema, stringListSchema, translationsSchema } from '@/shared/api';

/** `Article` (admin-api.md §11). */
export const articleSchema = z.object({
  id: z.number(),
  slug: z.string(),
  title: translationsSchema,
  excerpt: translationsSchema,
  body: translationsSchema,
  cycle_phases: stringListSchema,
  category: z.string().nullable(),
  read_time_minutes: z.number().nullable(),
  image_url: z.string().nullable(),
  image_path: z.string().nullable(),
  cover_url: z.string().nullable(),
  is_published: z.boolean(),
  published_at: z.string().nullable(),
  sort_order: z.number(),
});
export type Article = z.infer<typeof articleSchema>;

const phaseOptionSchema = z.object({ value: z.string(), label: z.string(), legacy: z.boolean().optional() });
export type PhaseOption = z.infer<typeof phaseOptionSchema>;

export const articlesApi = createResource({
  key: 'articles',
  path: '/articles',
  list: pagedSchema(articleSchema, {}),
  detail: z.object({ article: articleSchema, options: z.object({ phases: z.array(phaseOptionSchema) }) }),
  options: z.object({
    phases: z.array(phaseOptionSchema),
    max_image_width: z.number().optional(),
    max_image_kb: z.number().optional(),
  }),
  alsoInvalidate: [['dashboard']],
});
