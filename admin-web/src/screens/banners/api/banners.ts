'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { api, createResource, optionSchema, pagedSchema, translationsSchema } from '@/shared/api';

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
  // B-N10-02 (D-93): sponsored slots.
  kind: z.string().default('banner'),
  advertiser: z.string().nullable().default(null),
  cta_label: translationsSchema.optional(),
  modes: z.array(z.string()).default([]),
});
export type Banner = z.infer<typeof bannerSchema>;

const optionsSchema = z.object({
  positions: z.array(optionSchema),
  link_types: z.array(optionSchema),
  kinds: z.array(z.string()).default(['banner', 'ad']),
  ad_slots: z.array(z.string()).default([]),
  modes: z.array(z.string()).default([]),
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

const statsSchema = z.object({
  banner_id: z.number(),
  days: z.number(),
  totals: z.object({ impressions: z.number(), clicks: z.number(), ctr: z.number() }),
  items: z.array(z.object({ day: z.string(), impressions: z.number(), clicks: z.number() })),
});
export type BannerStats = z.infer<typeof statsSchema>;

/** GET /banners/:id/stats — per-day impression / click aggregates of an ad (no user data). */
export function useBannerStats(id: number | null, enabled: boolean) {
  return useQuery({
    queryKey: ['banners', 'stats', id],
    enabled: enabled && id !== null,
    queryFn: ({ signal }) => api.get(`/banners/${id}/stats`, { schema: statsSchema, signal }),
  });
}
