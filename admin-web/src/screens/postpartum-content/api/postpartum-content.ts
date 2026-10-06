'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { api, pagedSchema } from '@/shared/api';

/** The postpartum copy groups (B-N5-01 `internal/postpartum/guide`), edited row by row in /messages. */
export const POSTPARTUM_GROUPS = ['postpartum_week_tip', 'postpartum_alert', 'postpartum_safety'] as const;
export type PostpartumGroup = (typeof POSTPARTUM_GROUPS)[number];

const rowSchema = z.object({
  id: z.number(),
  item_key: z.string(),
  locale: z.string(),
  is_active: z.boolean(),
  is_approved: z.boolean(),
});

const listSchema = pagedSchema(rowSchema, {
  missing: z.array(z.object({ item_key: z.string(), locale: z.string() })).optional().default([]),
});

export interface GroupCoverage {
  group: PostpartumGroup;
  rows: number;
  live: number; // active + approved: what users see instead of the built-in copy
  missing: number; // registered slot × active language without a row (falls back to the built-in copy)
}

/** Row and missing-slot counts of one group (GET /messages?group=…; registry slots are < 100 per language). */
export function usePostpartumCoverage(group: PostpartumGroup) {
  return useQuery({
    queryKey: ['messages', 'postpartum-coverage', group],
    queryFn: async ({ signal }): Promise<GroupCoverage> => {
      const res = await api.get('/messages', { query: { group, per_page: 100 }, schema: listSchema, signal });
      return {
        group,
        rows: res.meta.total,
        live: res.items.filter((r) => r.is_active && r.is_approved).length,
        missing: res.missing.length,
      };
    },
  });
}
