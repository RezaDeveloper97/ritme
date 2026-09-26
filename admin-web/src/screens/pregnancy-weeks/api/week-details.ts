'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api, translationsSchema } from '@/shared/api';

const list = <T extends z.ZodTypeAny>(item: T) => z.preprocess((v) => (Array.isArray(v) ? v : []), z.array(item));
const nullableText = z.string().nullable().catch(null);

const highlightSchema = z.object({
  icon: nullableText.optional().transform((v) => v ?? null),
  tone: nullableText.optional().transform((v) => v ?? null),
  title: translationsSchema,
  body: translationsSchema,
});
const symptomSchema = z.object({ key: z.string(), label: translationsSchema });
const taskSchema = z.object({ key: z.string(), text: translationsSchema });
const sourceSchema = z.object({
  title: translationsSchema,
  url: nullableText.optional().transform((v) => v ?? null),
});

export type Highlight = z.infer<typeof highlightSchema>;
export type BodySymptom = z.infer<typeof symptomSchema>;
export type WeekTask = z.infer<typeof taskSchema>;
export type WeekSource = z.infer<typeof sourceSchema>;

/** `week_details` — the structured v2 content of one week (admin-api.md §13). */
const weekDetailsSchema = z.object({
  week_number: z.number(),
  exists: z.boolean(),
  size_label: translationsSchema,
  illustration_key: nullableText,
  length_cm: nullableText,
  weight_g: nullableText,
  heart_rate: nullableText,
  headline: translationsSchema,
  highlights: list(highlightSchema),
  body_symptoms: list(symptomSchema),
  body_text: translationsSchema,
  tasks: list(taskSchema),
  warning: translationsSchema,
  reviewer_name: translationsSchema,
  reviewed_at: nullableText,
  sources: list(sourceSchema),
});
export type WeekDetails = z.infer<typeof weekDetailsSchema>;

const optionsSchema = z.object({
  min_week: z.number(),
  max_week: z.number(),
  illustration_keys: z.array(z.string()),
  highlight_icons: z.array(z.string()),
  highlight_tones: z.array(z.string()),
  log_symptom_keys: z.array(z.string()),
  max_items: z.number(),
});
export type WeekDetailsOptions = z.infer<typeof optionsSchema>;

const summarySchema = z.object({
  items: z.array(
    z.object({
      week_number: z.number(),
      exists: z.boolean(),
      illustration_key: nullableText,
    }),
  ),
});

const keys = {
  all: ['pregnancy-week-details'] as const,
  list: ['pregnancy-week-details', 'list'] as const,
  options: ['pregnancy-week-details', 'options'] as const,
  detail: (week: number) => ['pregnancy-week-details', 'detail', week] as const,
};

/** GET /pregnancy-week-details — which weeks 1…42 have structured details. */
export function useWeekDetailsList() {
  return useQuery({
    queryKey: keys.list,
    queryFn: ({ signal }) => api.get('/pregnancy-week-details', { schema: summarySchema, signal }),
  });
}

export function useWeekDetailsOptions() {
  return useQuery({
    queryKey: keys.options,
    queryFn: ({ signal }) =>
      api.get('/pregnancy-week-details/options', {
        schema: optionsSchema,
        signal,
      }),
    staleTime: 5 * 60_000,
  });
}

export function useWeekDetails(week: number) {
  return useQuery({
    queryKey: keys.detail(week),
    queryFn: ({ signal }) =>
      api.get(`/pregnancy-weeks/${week}/details`, {
        schema: z.object({ week_details: weekDetailsSchema }),
        signal,
      }),
  });
}

/** PUT /pregnancy-weeks/:n/details — upsert. */
export function useSaveWeekDetails(week: number) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (body: unknown) => api.put(`/pregnancy-weeks/${week}/details`, body),
    onSuccess: () => client.invalidateQueries({ queryKey: keys.all }),
  });
}
