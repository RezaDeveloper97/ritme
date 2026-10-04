'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api, createResource, pagedSchema } from '@/shared/api';

const list = <T extends z.ZodTypeAny>(item: T) => z.preprocess((v) => (Array.isArray(v) ? v : []), z.array(item));
const record = <T extends z.ZodTypeAny>(item: T) =>
  z.preprocess((v) => (v && typeof v === 'object' && !Array.isArray(v) ? v : {}), z.record(z.string(), item));

/** Where the text the companion panel shows today comes from (admin-api.md §16). */
export type TipSource = 'locale' | 'default_language' | 'built_in';
const sourceSchema = z.enum(['locale', 'default_language', 'built_in']).catch('built_in');

const slotSchema = z.object({
  slot: z.number(),
  title: z.string().catch(''),
  body: z.string().catch(''),
  source: sourceSchema,
});
export type TipSlot = z.infer<typeof slotSchema>;

const localeTextsSchema = z.object({
  customized: z.boolean().catch(false),
  note: z.object({ body: z.string().catch(''), source: sourceSchema }),
  tips: list(slotSchema),
});
export type LocaleTips = z.infer<typeof localeTextsSchema>;

const phaseTipsSchema = z.object({
  phase: z.string(),
  texts: record(localeTextsSchema),
  updated_at: z.string().nullable().catch(null),
});
export type PhaseTips = z.infer<typeof phaseTipsSchema>;

const tipsListSchema = z.object({
  items: z.array(phaseTipsSchema),
  phases: z.array(z.string()),
  tips_per_phase: z.number(),
  placeholders: list(z.string()),
  limits: z.object({ note: z.number(), title: z.number(), body: z.number() }),
  default_locale: z.string(),
});
export type TipsList = z.infer<typeof tipsListSchema>;

const tipsKeys = {
  all: ['companion-tips'] as const,
};

/** GET /companions/tips — every phase with its copy per active language. */
export function useCompanionTips() {
  return useQuery({
    queryKey: tipsKeys.all,
    queryFn: ({ signal }) => api.get('/companions/tips', { schema: tipsListSchema, signal }),
  });
}

/** Body of PUT /companions/tips/:phase. */
export interface TipsBody {
  texts: Record<string, { note: string | null; tips: { title: string; body: string | null }[] }>;
}

/** PUT /companions/tips/:phase — writes every slot of the phase for each sent language. */
export function useSaveCompanionTips() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ phase, body }: { phase: string; body: TipsBody }) =>
      api.put(`/companions/tips/${encodeURIComponent(phase)}`, body),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: tipsKeys.all });
      void client.invalidateQueries({ queryKey: ['messages'] });
    },
  });
}

/** DELETE /companions/tips/:phase?locale= — back to the default language / built-in copy. */
export function useResetCompanionTips() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ phase, locale }: { phase: string; locale: string }) =>
      api.delete(`/companions/tips/${encodeURIComponent(phase)}`, { query: { locale } }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: tipsKeys.all });
      void client.invalidateQueries({ queryKey: ['messages'] });
    },
  });
}

// ---------------------------------------------------------------------------
// Link overview (read-only, masked)

// No user id since B-N4-08b (CMP-L6): masked name + last 2 digits of the mobile only.
const personSchema = z.object({ name: z.string().nullable(), mobile: z.string().nullable() });

export const companionLinkSchema = z.object({
  id: z.number(),
  type: z.string(),
  status: z.string(),
  label: z.string().nullable(),
  owner: personSchema,
  companion: personSchema.nullable(),
  invite: z.object({ phone: z.string().nullable(), expires_at: z.string().nullable(), expired: z.boolean() }).nullable(),
  grants_count: z.number(),
  invited_at: z.string().nullable(),
  accepted_at: z.string().nullable(),
  revoked_at: z.string().nullable(),
  revoked_by: z.string().nullable(),
  created_at: z.string().nullable(),
});
export type CompanionLink = z.infer<typeof companionLinkSchema>;

const statusCountsSchema = z.object({
  all: z.number(),
  invited: z.number(),
  active: z.number(),
  revoked: z.number(),
});
export type StatusCounts = z.infer<typeof statusCountsSchema>;

export const companionLinksApi = createResource({
  key: 'companion-links',
  path: '/companions/links',
  list: pagedSchema(companionLinkSchema, {
    filters: z.object({ status: z.string(), type: z.string() }).partial().optional(),
    counts: z.object({ by_status: statusCountsSchema, by_type: record(statusCountsSchema) }).optional(),
    types: list(z.string()).optional(),
  }),
  detail: z.unknown(),
});
