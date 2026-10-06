'use client';

import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api, pagedSchema, stringListSchema, translationsSchema } from '@/shared/api';

/**
 * The child catalogs through the generic catalog API (`/catalog/{group}`, admin-api.md §18; writes super-only)
 * and the read-only WHO viewer (`/children/who`).
 */
export const childItemSchema = z.object({
  id: z.number(),
  group: z.string(),
  code: z.string(),
  sort_order: z.number(),
  is_active: z.boolean(),
  audiences: stringListSchema,
  title: translationsSchema,
  body: translationsSchema,
  meta: z.preprocess(
    (v) => (v && typeof v === 'object' && !Array.isArray(v) ? v : null),
    z.record(z.string(), z.unknown()).nullable(),
  ),
  needs_review: z.boolean(),
  updated_at: z.string().nullable().optional(),
});
export type ChildItem = z.infer<typeof childItemSchema>;

const listSchema = pagedSchema(childItemSchema, {});
const detailSchema = z.object({ catalog_item: childItemSchema });

/** A group is a short list; 100 is the API's page maximum. */
const PER_PAGE = 100;
const MAX_PAGES = 20;

const path = (group: string, id?: number) => `/catalog/${encodeURIComponent(group)}${id === undefined ? '' : `/${id}`}`;

/** Every item of a group (all pages), in catalog order. */
export async function fetchAllItems(group: string, signal?: AbortSignal): Promise<ChildItem[]> {
  const out: ChildItem[] = [];
  for (let page = 1; page <= MAX_PAGES; page++) {
    const res = await api.get(path(group), { query: { page, per_page: PER_PAGE }, schema: listSchema, signal });
    out.push(...res.items);
    if (page >= res.meta.last_page) break;
  }
  return out;
}

export const childKeys = {
  all: ['catalog'] as const, // shared with the generic catalog screens, so both refresh together
  group: (group: string) => ['catalog', 'group', group] as const,
  items: (group: string) => ['catalog', 'group', group, 'all'] as const,
  detail: (group: string, id: number) => ['catalog', 'group', group, 'detail', id] as const,
  who: (query: WhoQuery) => ['children-who', query] as const,
};

export function useChildItems(group: string) {
  return useQuery({ queryKey: childKeys.items(group), queryFn: ({ signal }) => fetchAllItems(group, signal) });
}

export function useChildItem(group: string, id: number | null) {
  return useQuery({
    queryKey: childKeys.detail(group, id ?? 0),
    queryFn: ({ signal }) => api.get(path(group, id as number), { schema: detailSchema, signal }),
    enabled: id !== null,
  });
}

function useInvalidate(group: string) {
  const client = useQueryClient();
  return () =>
    Promise.all([
      client.invalidateQueries({ queryKey: childKeys.group(group) }),
      client.invalidateQueries({ queryKey: ['catalog', 'groups'] }),
    ]);
}

/** Create (`id === null`) or full update from the form. */
export function useSaveChildItem(group: string, id: number | null) {
  const invalidate = useInvalidate(group);
  return useMutation({
    mutationFn: (body: unknown) =>
      id === null
        ? api.post(path(group), body, { schema: detailSchema })
        : api.put(path(group, id), body, { schema: detailSchema }),
    onSuccess: () => invalidate(),
  });
}

/** A list-level partial PUT («بازبینی شد», active): title is required on every PUT, the rest is kept. */
export function usePatchChildItem(group: string) {
  const invalidate = useInvalidate(group);
  return useMutation({
    mutationFn: ({ row, patch }: { row: ChildItem; patch: { needs_review?: boolean; is_active?: boolean } }) =>
      api.put(path(group, row.id), { title: row.title, ...patch }, { schema: detailSchema }),
    onSuccess: () => invalidate(),
  });
}

export function useRemoveChildItem(group: string) {
  const invalidate = useInvalidate(group);
  return useMutation({
    mutationFn: (id: number) => api.delete(path(group, id)),
    onSuccess: () => invalidate(),
  });
}

// ---------------------------------------------------------------------------
// WHO viewer

export interface WhoQuery {
  indicator: string;
  sex: string;
  step: string;
  page: number;
}

export const whoRowSchema = z.object({
  age: z.number(),
  day: z.number(),
  l: z.number(),
  m: z.number(),
  s: z.number(),
  p3: z.number(),
  p15: z.number(),
  p50: z.number(),
  p85: z.number(),
  p97: z.number(),
});
export type WhoRow = z.infer<typeof whoRowSchema>;

const whoSchema = pagedSchema(whoRowSchema, {
  filters: z.object({ indicator: z.string(), sex: z.string(), step: z.string() }),
  unit: z.string(),
  options: z.object({
    indicators: z.array(z.string()),
    sexes: z.array(z.string()),
    steps: z.array(z.string()),
    max_day: z.number(),
  }),
  source: z.string(),
});
export type WhoPage = z.infer<typeof whoSchema>;

/** GET /children/who — one page (≤ 100 rows) of an indicator × sex sampled by step. */
export function useWho(query: WhoQuery) {
  return useQuery({
    queryKey: childKeys.who(query),
    queryFn: ({ signal }) => api.get('/children/who', { query: { ...query, per_page: 100 }, schema: whoSchema, signal }),
    placeholderData: keepPreviousData,
  });
}
