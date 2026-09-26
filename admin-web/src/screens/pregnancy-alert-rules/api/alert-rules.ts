'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api } from '@/shared/api';

/** A typed param of a rule (admin-api.md §13 `Schema`, the kinds a rule uses). */
export interface ParamField {
  key: string;
  kind: string;
  nullable: boolean;
  min?: number;
  max?: number;
  values?: string[];
  min_items?: number;
  max_items?: number;
}
const paramFieldSchema = z.object({
  key: z.string(),
  kind: z.string(),
  nullable: z.boolean(),
  min: z.number().optional(),
  max: z.number().optional(),
  values: z.array(z.string()).optional(),
  min_items: z.number().optional(),
  max_items: z.number().optional(),
});
const objectSchema = z.preprocess((v) => (v && typeof v === 'object' && !Array.isArray(v) ? v : {}), z.record(z.string(), z.unknown()));
const list = <T extends z.ZodTypeAny>(item: T) => z.preprocess((v) => (Array.isArray(v) ? v : []), z.array(item));

const actionSchema = z.object({ key: z.string(), label: z.string().catch('') });
export type AlertAction = z.infer<typeof actionSchema>;

const textsSchema = z.object({
  title: z.string().nullable().catch(''),
  what_we_saw: z.string().nullable().catch(''),
  how_sure: z.string().nullable().catch(''),
  advice: z.string().nullable().catch(''),
  actions: list(actionSchema),
  contact: z.string().nullable().catch(null),
});
export type AlertTexts = z.infer<typeof textsSchema>;

const summarySchema = z.object({
  key: z.string(),
  configured: z.boolean(),
  enabled: z.boolean().nullable().catch(false),
  level: z.string().nullable().catch(null),
  window_days: z.number().nullable().catch(null),
  params: objectSchema,
  title: z.string().nullable().catch(null),
  locales: list(z.string()),
  missing_locales: list(z.string()),
});
export type AlertRuleSummary = z.infer<typeof summarySchema>;

const listSchema = z.object({
  items: z.array(summarySchema),
  legend: z
    .object({
      group: z.string(),
      item_key: z.string(),
      rows: list(z.object({ id: z.number(), locale: z.string() })),
    })
    .nullable()
    .catch(null),
});

const optionsSchema = z.object({
  levels: z.array(z.string()),
  actions: z.array(z.string()),
  min_window_days: z.number(),
  max_window_days: z.number(),
  rules: z.array(
    z.object({
      key: z.string(),
      params_schema: list(paramFieldSchema),
      placeholders: list(z.string()),
    }),
  ),
});
export type AlertOptions = z.infer<typeof optionsSchema>;

const ruleSchema = z.object({
  key: z.string(),
  configured: z.boolean(),
  enabled: z.boolean().nullable().catch(false),
  level: z.string().nullable().catch(null),
  window_days: z.number().nullable().catch(null),
  params: objectSchema,
  texts: z.preprocess((v) => (v && typeof v === 'object' && !Array.isArray(v) ? v : {}), z.record(z.string(), textsSchema)),
  missing_locales: list(z.string()),
  params_schema: list(paramFieldSchema),
  placeholders: list(z.string()),
});
export type AlertRule = z.infer<typeof ruleSchema>;

const keys = {
  all: ['pregnancy-alert-rules'] as const,
  list: ['pregnancy-alert-rules', 'list'] as const,
  options: ['pregnancy-alert-rules', 'options'] as const,
  detail: (key: string) => ['pregnancy-alert-rules', 'detail', key] as const,
};

export function useAlertRules() {
  return useQuery({
    queryKey: keys.list,
    queryFn: ({ signal }) => api.get('/pregnancy-alert-rules', { schema: listSchema, signal }),
  });
}

export function useAlertOptions() {
  return useQuery({
    queryKey: keys.options,
    queryFn: ({ signal }) =>
      api.get('/pregnancy-alert-rules/options', {
        schema: optionsSchema,
        signal,
      }),
    staleTime: 5 * 60_000,
  });
}

export function useAlertRule(key: string) {
  return useQuery({
    queryKey: keys.detail(key),
    queryFn: ({ signal }) =>
      api.get(`/pregnancy-alert-rules/${encodeURIComponent(key)}`, {
        schema: z.object({ rule: ruleSchema }),
        signal,
      }),
  });
}

/** PUT /pregnancy-alert-rules/:key — behaviour for every locale row, texts per locale. */
export function useSaveAlertRule(key: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (body: unknown) => api.put(`/pregnancy-alert-rules/${encodeURIComponent(key)}`, body),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: keys.all });
      void client.invalidateQueries({ queryKey: ['messages'] });
    },
  });
}
