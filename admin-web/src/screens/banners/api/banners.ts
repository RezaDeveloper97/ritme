'use client';

import { z } from 'zod';

import { createResource, optionSchema, pagedSchema, translationsSchema } from '@/shared/api';

/** `Banner` (admin-api.md §11). */
export const bannerSchema = z.object({
  id: z.number(),
  title: translationsSchema,
  image_path: z.string(),
  image_url: z.string().nullable(),
  position: z.string(),
  link_url: z.string().nullable(),
  link_type: z.string().nullable(),
  starts_at: z.string().nullable(),
  ends_at: z.string().nullable(),
  is_active: z.boolean(),
  sort_order: z.number(),
});
export type Banner = z.infer<typeof bannerSchema>;

const optionsSchema = z.object({
  positions: z.array(optionSchema),
  link_types: z.array(optionSchema),
  image: z.object({
    max_kb: z.number(),
    min_width: z.number(),
    min_height: z.number(),
    recommended_width: z.number(),
    recommended_height: z.number(),
    types: z.array(z.string()),
  }),
});
export type BannerOptions = z.infer<typeof optionsSchema>;

export const bannersApi = createResource({
  key: 'banners',
  path: '/banners',
  list: pagedSchema(bannerSchema, {}),
  detail: z.object({ banner: bannerSchema }),
  options: optionsSchema,
});
