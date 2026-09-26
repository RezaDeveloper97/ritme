'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api, createResource, translationsSchema } from '@/shared/api';

/** `CareItem` — one row of the pregnancy care plan (admin-api.md §13). */
export const careItemSchema = z.object({
  id: z.number(),
  key: z.string(),
  title: translationsSchema,
  prep: translationsSchema,
  kind: z.string(),
  week_from: z.number(),
  week_to: z.number(),
  remind_before: z.number(),
  sort_order: z.number(),
  is_active: z.boolean(),
  appointments_count: z.number().optional().default(0),
});
export type CareItem = z.infer<typeof careItemSchema>;

export const careOptionsSchema = z.object({
  kinds: z.array(z.string()),
  min_week: z.number(),
  max_week: z.number(),
  max_remind_before: z.number(),
  default_remind_before: z.number(),
  next_sort_order: z.number(),
});
export type CareOptions = z.infer<typeof careOptionsSchema>;

export const careItemsApi = createResource({
  key: 'pregnancy-care-items',
  path: '/pregnancy-care-items',
  list: z.object({ items: z.array(careItemSchema) }),
  detail: z.object({ care_item: careItemSchema }),
  options: careOptionsSchema,
});

/** POST /pregnancy-care-items/reorder — every item id once, in the new order. */
export function useReorderCareItems() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (ids: number[]) => api.post('/pregnancy-care-items/reorder', { ids }),
    onSettled: () => client.invalidateQueries({ queryKey: careItemsApi.keys.all }),
  });
}
