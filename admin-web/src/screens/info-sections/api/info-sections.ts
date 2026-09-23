'use client';

import { z } from 'zod';

import { createResource, pagedSchema, translationsSchema } from '@/shared/api';

/** `InfoSection` — one box on a text screen of the app (admin-api.md §11). */
export const infoSectionSchema = z.object({
  id: z.number(),
  group: z.string(),
  key: z.string().nullable(),
  heading: translationsSchema,
  body: translationsSchema,
  link_label: translationsSchema,
  link_url: z.string().nullable(),
  is_active: z.boolean(),
  sort_order: z.number(),
});
export type InfoSection = z.infer<typeof infoSectionSchema>;

export const infoSectionsApi = createResource({
  key: 'info-sections',
  path: '/info-sections',
  list: pagedSchema(infoSectionSchema, { filters: z.object({ group: z.string() }).partial().optional() }),
  detail: z.object({ info_section: infoSectionSchema }),
  options: z.object({ groups: z.array(z.string()), group: z.string(), next_sort_order: z.number() }),
});

/** The app's text screens, in tab order (fallback until /options answers). */
export const INFO_GROUPS = ['help', 'privacy', 'terms', 'about'] as const;
