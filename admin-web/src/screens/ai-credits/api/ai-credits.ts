'use client';

import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api, createResource, translationsSchema } from '@/shared/api';

/**
 * AI credits admin (admin-api.md §29, docs/ai-credits.md): the model catalogue, the feature slots, the credit packs
 * and the usage report (§17). Model prices are milli-USD per 1M tokens; pack prices are rials (the UI shows toman).
 * API keys are server env only and never part of this API.
 */

export const CAPABILITIES = ['chat', 'vision', 'audio', 'json'] as const;
export type Capability = (typeof CAPABILITIES)[number];

// ---------------------------------------------------------------------------
// Models

export const modelSchema = z.object({
  id: z.number(),
  code: z.string(),
  provider: z.string(),
  upstream_model: z.string(),
  title: z.string(),
  capabilities: z.array(z.string()),
  input_price_milli: z.number(),
  output_price_milli: z.number(),
  audio_price_milli: z.number(),
  is_active: z.boolean(),
  configured: z.boolean(),
  slots_count: z.number(),
});
export type AiModel = z.infer<typeof modelSchema>;

const modelsListSchema = z.object({
  items: z.array(modelSchema),
  providers: z.array(z.string()),
  configured_providers: z.array(z.string()),
  capabilities: z.array(z.string()),
  limits: z.object({ max_price_milli: z.number() }),
});

export const modelsApi = createResource({
  key: 'ai-models',
  path: '/ai/models',
  list: modelsListSchema,
  detail: z.object({ model: modelSchema, slots: z.array(z.string()) }),
  alsoInvalidate: [['ai-features']],
});

const syncResultSchema = z.object({
  provider: z.string(),
  added: z.array(z.object({ code: z.string(), upstream_model: z.string() })),
  existing: z.number(),
  skipped: z.number(),
});
export type SyncResult = z.infer<typeof syncResultSchema>;

/** POST /ai/models/sync — adds the gateway's unknown models as inactive rows. */
export function useSyncModels() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (provider: string) => api.post('/ai/models/sync', { provider }, { schema: syncResultSchema }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: modelsApi.keys.all });
      void client.invalidateQueries({ queryKey: featureKeys.all });
    },
  });
}

/** DELETE /ai/models/:id — refused (422 ai_model_in_use) while a feature slot points at the model. */
export function useRemoveModel() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => api.delete(`/ai/models/${id}`, { schema: z.unknown() }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: modelsApi.keys.all });
      void client.invalidateQueries({ queryKey: featureKeys.all });
    },
  });
}

// ---------------------------------------------------------------------------
// Feature slots

const modelRefSchema = z
  .object({ id: z.number(), code: z.string(), title: z.string(), provider: z.string(), is_active: z.boolean() })
  .nullable();

export const featureSchema = z.object({
  id: z.number(),
  key: z.string(),
  title: translationsSchema,
  needs: z.array(z.string()),
  declared: z.boolean(),
  model_id: z.number().nullable(),
  fallback_model_id: z.number().nullable(),
  model: modelRefSchema,
  fallback_model: modelRefSchema,
  credit_cost: z.number(),
  is_active: z.boolean(),
  sort_order: z.number(),
  usable: z.boolean(),
  routes: z.number(),
  usable_model_ids: z.array(z.number()),
});
export type AiFeature = z.infer<typeof featureSchema>;

export const modelOptionSchema = z.object({
  id: z.number(),
  code: z.string(),
  title: z.string(),
  provider: z.string(),
  upstream_model: z.string(),
  capabilities: z.array(z.string()),
  is_active: z.boolean(),
  configured: z.boolean(),
});
export type AiModelOption = z.infer<typeof modelOptionSchema>;

const featuresBodySchema = z.object({
  items: z.array(featureSchema),
  models: z.array(modelOptionSchema),
  capabilities: z.array(z.string()),
  limits: z.object({ max_credit_cost: z.number() }),
});

export const featureKeys = { all: ['ai-features'] as const };

export function useFeatures() {
  return useQuery({
    queryKey: featureKeys.all,
    queryFn: ({ signal }) => api.get('/ai/features', { schema: featuresBodySchema, signal }),
  });
}

export interface FeatureInput {
  title?: Record<string, string>;
  model_id: number | null;
  fallback_model_id: number | null;
  credit_cost: number | null;
  is_active: boolean;
  sort_order?: number | null;
}

export function useSaveFeature() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: number; body: FeatureInput }) =>
      api.put(`/ai/features/${id}`, body, { schema: z.object({ feature: featureSchema }) }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: featureKeys.all });
      void client.invalidateQueries({ queryKey: modelsApi.keys.all });
    },
  });
}

// ---------------------------------------------------------------------------
// Credit packs

export const packSchema = z.object({
  id: z.number(),
  code: z.string(),
  title: translationsSchema,
  badge: translationsSchema,
  credits: z.number(),
  price_rials: z.number(),
  price_per_credit_rials: z.number(),
  is_highlighted: z.boolean(),
  is_active: z.boolean(),
  sort_order: z.number(),
  invoices_count: z.number(),
  paid_count: z.number(),
});
export type CreditPack = z.infer<typeof packSchema>;

export const packsApi = createResource({
  key: 'ai-credit-packs',
  path: '/ai/credit-packs',
  list: z.object({
    items: z.array(packSchema),
    limits: z.object({ min_price_rials: z.number(), max_price_rials: z.number(), max_credits: z.number() }),
    next_sort_order: z.number(),
  }),
  detail: z.object({ pack: packSchema }),
});

/** DELETE answers whether the pack was deleted or only deactivated (an invoice references it). */
export function useRemovePack() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id: number) =>
      api.delete(`/ai/credit-packs/${id}`, {
        schema: z.object({ id: z.number(), deleted: z.boolean(), deactivated: z.boolean() }),
      }),
    onSuccess: () => void client.invalidateQueries({ queryKey: packsApi.keys.all }),
  });
}

// ---------------------------------------------------------------------------
// Usage report (GET /ai/usage)

const costRow = {
  calls: z.number(),
  ok_calls: z.number(),
  input_tokens: z.number(),
  output_tokens: z.number(),
  cost_micros: z.number(),
  cost_usd: z.number(),
};

const creditRow = { uses: z.number(), credits: z.number(), plan_credits: z.number(), extra_credits: z.number() };

export const usageSchema = z.object({
  from: z.string(),
  to: z.string(),
  today: z.object({
    spent_usd: z.number().nullable(),
    cap_usd: z.number().nullable(),
    exhausted: z.boolean().nullable(),
  }),
  totals: z.object(costRow),
  by_day: z.array(z.object({ day: z.string(), ...costRow })),
  by_feature: z.array(
    z.object({
      feature: z.string(),
      provider: z.string(),
      model: z.string(),
      users: z.number(),
      avg_latency_ms: z.number(),
      ...costRow,
    }),
  ),
  credits: z
    .object({
      total: z.number(),
      uses: z.number(),
      plan_credits: z.number(),
      extra_credits: z.number(),
      by_day: z.array(z.object({ day: z.string(), ...creditRow })),
      by_feature: z.array(z.object({ feature: z.string(), users: z.number(), ...creditRow })),
    })
    .default({ total: 0, uses: 0, plan_credits: 0, extra_credits: 0, by_day: [], by_feature: [] }),
  usd_irr_rate: z.number().nullable().default(null),
});
export type UsageReport = z.infer<typeof usageSchema>;

export function useUsage(range: { from: string; to: string } | null) {
  return useQuery({
    queryKey: ['ai-usage', range ?? {}] as const,
    queryFn: ({ signal }) => api.get('/ai/usage', { query: range ?? {}, schema: usageSchema, signal }),
    placeholderData: keepPreviousData,
  });
}
