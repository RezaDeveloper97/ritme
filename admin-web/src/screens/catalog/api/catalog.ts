'use client';

import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api, pagedSchema, stringListSchema, translationsSchema, type QueryValue } from '@/shared/api';

import { partialUpdate, planReorder, type RowForWrite } from '../lib/payload';

/** `catalog_item` (docs/canvas-build/catalog.md §3; JSON columns decoded, every language). */
export const catalogItemSchema = z.object({
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
  created_at: z.string().nullable().optional(),
  updated_at: z.string().nullable().optional(),
});
export type CatalogItem = z.infer<typeof catalogItemSchema>;

export const catalogGroupSchema = z.object({ group: z.string(), items_count: z.number(), active_count: z.number() });
export type CatalogGroup = z.infer<typeof catalogGroupSchema>;

const groupsSchema = z.object({ items: z.array(catalogGroupSchema) });
const listSchema = pagedSchema(catalogItemSchema, {
  filters: z.object({ group: z.string(), q: z.string(), status: z.string() }).partial().optional(),
});
const detailSchema = z.object({ catalog_item: catalogItemSchema });

const path = (group: string, id?: number) => `/catalog/${encodeURIComponent(group)}${id === undefined ? '' : `/${id}`}`;

/** The requests, plain so they are testable without React (catalog.test.ts). */
export const catalogRequests = {
  groups: (signal?: AbortSignal) => api.get('/catalog', { schema: groupsSchema, signal }),
  list: (group: string, query: Record<string, QueryValue>, signal?: AbortSignal) =>
    api.get(path(group), { query, schema: listSchema, signal }),
  detail: (group: string, id: number, signal?: AbortSignal) => api.get(path(group, id), { schema: detailSchema, signal }),
  create: (group: string, body: unknown) => api.post(path(group), body, { schema: detailSchema }),
  update: (group: string, id: number, body: unknown) => api.put(path(group, id), body, { schema: detailSchema }),
  remove: (group: string, id: number) => api.delete(path(group, id)),
  /**
   * Store `ordered` as the new order: one partial PUT per row whose sort_order changes
   * (no bulk endpoint; sequential so a failure leaves a prefix applied, fixed by a retry).
   */
  reorder: async (group: string, ordered: readonly RowForWrite[]) => {
    const plan = planReorder(ordered);
    const byId = new Map(ordered.map((r) => [r.id, r]));
    for (const step of plan) {
      const row = byId.get(step.id) as RowForWrite;
      await api.put(path(group, step.id), partialUpdate(row, { sort_order: step.sort_order }));
    }
    return plan.length;
  },
};

export const catalogKeys = {
  all: ['catalog'] as const,
  groups: () => ['catalog', 'groups'] as const,
  group: (group: string) => ['catalog', 'group', group] as const,
  list: (group: string, query: Record<string, QueryValue>) => ['catalog', 'group', group, 'list', query] as const,
  detail: (group: string, id: number) => ['catalog', 'group', group, 'detail', id] as const,
};

export function useCatalogGroups() {
  return useQuery({ queryKey: catalogKeys.groups(), queryFn: ({ signal }) => catalogRequests.groups(signal) });
}

export function useCatalogItems(group: string, query: Record<string, QueryValue>) {
  return useQuery({
    queryKey: catalogKeys.list(group, query),
    queryFn: ({ signal }) => catalogRequests.list(group, query, signal),
    placeholderData: keepPreviousData,
  });
}

export function useCatalogItem(group: string, id: number | null) {
  return useQuery({
    queryKey: catalogKeys.detail(group, id ?? 0),
    queryFn: ({ signal }) => catalogRequests.detail(group, id as number, signal),
    enabled: id !== null,
  });
}

/** Every write makes the group's lists and the group counts stale. */
function useInvalidate(group: string) {
  const client = useQueryClient();
  return () =>
    Promise.all([
      client.invalidateQueries({ queryKey: catalogKeys.group(group) }),
      client.invalidateQueries({ queryKey: catalogKeys.groups() }),
    ]);
}

/** Create (`id === null`) or full update from the form. */
export function useSaveItem(group: string, id: number | null) {
  const invalidate = useInvalidate(group);
  return useMutation({
    mutationFn: (body: unknown) =>
      id === null ? catalogRequests.create(group, body) : catalogRequests.update(group, id, body),
    onSuccess: () => invalidate(),
  });
}

/** A list-level partial PUT (the active switch). */
export function usePatchItem(group: string) {
  const invalidate = useInvalidate(group);
  return useMutation({
    mutationFn: ({ row, patch }: { row: RowForWrite; patch: { is_active?: boolean; sort_order?: number } }) =>
      catalogRequests.update(group, row.id, partialUpdate(row, patch)),
    onSuccess: () => invalidate(),
  });
}

export function useReorderItems(group: string) {
  const invalidate = useInvalidate(group);
  return useMutation({
    mutationFn: (ordered: readonly RowForWrite[]) => catalogRequests.reorder(group, ordered),
    onSettled: () => invalidate(),
  });
}

export function useRemoveItem(group: string) {
  const invalidate = useInvalidate(group);
  return useMutation({
    mutationFn: (id: number) => catalogRequests.remove(group, id),
    onSuccess: () => invalidate(),
  });
}
