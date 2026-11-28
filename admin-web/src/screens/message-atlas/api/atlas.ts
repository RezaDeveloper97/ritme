'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { api, pagedSchema } from '@/shared/api';

/**
 * The rules of the message engine as the atlas needs them (GET /messages/rules, admin-api.md §22): what is shown
 * (content), where (placement), when (conditions / exceptions) and whether it is live. Every page is read — the
 * atlas is the whole picture, not a paged list.
 */
const conditionSchema = z.object({ fact: z.string(), op: z.string(), value: z.unknown() });
const groupSchema = z.object({ op: z.enum(['and', 'or']), items: z.array(conditionSchema).default([]) });
const variantSchema = z.object({
  title: z.string().default(''),
  body: z.string().default(''),
  cta_label: z.string().optional().default(''),
  cta_target: z.string().optional().default(''),
});

export const atlasRuleSchema = z.object({
  id: z.number(),
  title: z.string(),
  rule_key: z.string().nullish().transform((v) => v ?? null),
  level: z.string(),
  kind: z.string(),
  priority: z.number(),
  placement: z.string(),
  content: z.object({ a: variantSchema }),
  conditions: z.array(groupSchema).nullable().transform((v) => v ?? []),
  exceptions: z.array(conditionSchema).nullable().transform((v) => v ?? []),
  status: z.string(),
  is_active: z.boolean(),
  locked: z.boolean().default(false),
});
export type AtlasRule = z.infer<typeof atlasRuleSchema>;
export type AtlasCondition = z.infer<typeof conditionSchema>;

const pageSchema = pagedSchema(atlasRuleSchema, {});
const PER_PAGE = 100; // httpadmin.MaxPerPage
const MAX_PAGES = 30; // a guard, not a limit anyone should reach

export async function fetchAllRules(signal?: AbortSignal): Promise<AtlasRule[]> {
  const out: AtlasRule[] = [];
  for (let page = 1; page <= MAX_PAGES; page += 1) {
    const res = await api.get('/messages/rules', { query: { page, per_page: PER_PAGE }, schema: pageSchema, signal });
    out.push(...res.items);
    if (page >= res.meta.last_page) break;
  }
  return out;
}

export function useAtlasRules() {
  return useQuery({ queryKey: ['message-atlas', 'rules'], queryFn: ({ signal }) => fetchAllRules(signal) });
}
