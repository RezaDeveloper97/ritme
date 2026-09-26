'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api, createResource, pagedSchema, translationsSchema } from '@/shared/api';

const guideStepSchema = z.object({ title: translationsSchema, body: translationsSchema });
const findingSchema = z.object({
  key: z.string(),
  exclusive: z.boolean().optional().default(false),
  label: translationsSchema,
});
const listOf = <T extends z.ZodTypeAny>(item: T) =>
  z.preprocess((v) => (Array.isArray(v) ? v : []), z.array(item));

/** `CheckupType` — one shared catalog row (admin-api.md §12). */
export const checkupTypeSchema = z.object({
  id: z.number(),
  key: z.string(),
  category: z.string(),
  title: translationsSchema,
  subtitle: translationsSchema,
  why: translationsSchema,
  performed_by: z.string(),
  icon: z.string().nullable(),
  tone: z.string().nullable(),
  interval_months: z.number(),
  interval_months_max: z.number().nullable(),
  age_min: z.number().nullable(),
  age_max: z.number().nullable(),
  cycle_day_from: z.number().nullable(),
  cycle_day_to: z.number().nullable(),
  remind_lead_days: z.number().nullable(),
  prep_steps: listOf(translationsSchema),
  guide_steps: listOf(guideStepSchema),
  finding_options: listOf(findingSchema),
  hide_in_pregnancy: z.boolean(),
  is_active: z.boolean(),
  sort_order: z.number(),
  source_note: z.string().nullable(),
  records_count: z.number().optional().default(0),
});
export type CheckupType = z.infer<typeof checkupTypeSchema>;
export type GuideStep = z.infer<typeof guideStepSchema>;
export type FindingOption = z.infer<typeof findingSchema>;

export const checkupOptionsSchema = z.object({
  categories: z.array(z.string()),
  performed_by: z.array(z.string()),
  icons: z.array(z.string()),
  tones: z.array(z.string()),
  default_tone: z.string(),
  default_remind_lead_days: z.number(),
  max_steps: z.number(),
  max_cycle_day: z.number(),
  max_interval_months: z.number(),
  next_sort_order: z.number(),
});
export type CheckupOptions = z.infer<typeof checkupOptionsSchema>;

export const checkupTypesApi = createResource({
  key: 'checkup-types',
  path: '/checkup-types',
  list: pagedSchema(checkupTypeSchema, {}),
  detail: z.object({ checkup_type: checkupTypeSchema }),
  options: checkupOptionsSchema,
});

const statsSchema = z.object({
  items: z.array(
    z.object({
      id: z.number(),
      records_total: z.number(),
      users_with_records: z.number(),
      records_last_30_days: z.number(),
      overdue_users: z.number(),
    }),
  ),
  window_days: z.number(),
});
export type CheckupStats = z.infer<typeof statsSchema>;

export function useCheckupStats() {
  return useQuery({
    queryKey: [...checkupTypesApi.keys.all, 'stats'],
    queryFn: ({ signal }) => api.get('/checkup-types/stats', { schema: statsSchema, signal }),
  });
}

/** POST /checkup-types/reorder — every catalog id once, in the new order. */
export function useReorder() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (ids: number[]) => api.post('/checkup-types/reorder', { ids }),
    onSettled: () => client.invalidateQueries({ queryKey: checkupTypesApi.keys.all }),
  });
}

/** PUT /checkup-types/:id for any row (the list's active switch). */
export function useUpdateRow() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: unknown }) => api.put(`/checkup-types/${id}`, body),
    onSuccess: () => client.invalidateQueries({ queryKey: checkupTypesApi.keys.all }),
  });
}
