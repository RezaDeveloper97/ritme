'use client';

import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api, type QueryValue } from './client';
import { pageMetaSchema } from './envelope';
import { ApiError } from './errors';

/**
 * A translatable column (`{fa: '…', en: '…'}`, admin-api.md §3). The API sends
 * `null` for an empty nullable column and PHP-packed `[]` for an empty object;
 * both become `{}` so forms can spread it. Non-string values are dropped.
 */
export const translationsSchema = z.preprocess((value) => {
  const out: Record<string, string> = {};
  if (value && typeof value === 'object' && !Array.isArray(value)) {
    for (const [code, text] of Object.entries(value)) {
      if (typeof text === 'string') out[code] = text;
      else if (typeof text === 'number') out[code] = String(text);
    }
  }
  return out;
}, z.record(z.string(), z.string()));

/** A JSON list column (`cycle_phases`, `cycle_subphases`): null / non-list → []. */
export const stringListSchema = z.preprocess(
  (value) => (Array.isArray(value) ? value.filter((v): v is string => typeof v === 'string') : []),
  z.array(z.string()),
);

/** `{items, meta, …extra}` — a paginated list whose extra keys vary per endpoint. */
export function pagedSchema<I extends z.ZodTypeAny, E extends z.ZodRawShape>(item: I, extra: E) {
  return z.object({ items: z.array(item), meta: pageMetaSchema }).extend(extra);
}

/** Any successful write: the record itself is re-read through the list/detail queries. */
const writeSchema = z.unknown();

type Query = Record<string, QueryValue>;

/**
 * Query keys + TanStack hooks for one admin CRUD resource (admin-api.md §11
 * conventions: list, `/options`, `/:id`, create, update, `/:id/<action>`, delete).
 * Each screen instantiates it in its own `api/` segment with its zod schemas;
 * every write invalidates the whole resource (`keys.all`).
 */
export function createResource<L extends z.ZodTypeAny, D extends z.ZodTypeAny, O extends z.ZodTypeAny>(config: {
  key: string;
  path: string;
  list: L;
  detail: D;
  options?: O;
  /** Other query roots a write makes stale (e.g. the dashboard counters). */
  alsoInvalidate?: readonly (readonly unknown[])[];
}) {
  const { key, path } = config;
  const keys = {
    all: [key] as const,
    list: (query: Query) => [key, 'list', query] as const,
    detail: (id: number) => [key, 'detail', id] as const,
    options: (query: Query) => [key, 'options', query] as const,
  };

  /** Invalidate the resource (except, after a delete, the removed record's detail). */
  function useInvalidate() {
    const client = useQueryClient();
    return (removedId?: number) => {
      void client.invalidateQueries({
        queryKey: keys.all,
        predicate: (q) => removedId === undefined || !(q.queryKey[1] === 'detail' && q.queryKey[2] === removedId),
      });
      for (const other of config.alsoInvalidate ?? []) void client.invalidateQueries({ queryKey: other });
    };
  }

  function useList(query: Query = {}, opts: { enabled?: boolean } = {}) {
    return useQuery({
      queryKey: keys.list(query),
      queryFn: ({ signal }) => api.get(path, { query, schema: config.list, signal }),
      placeholderData: keepPreviousData,
      enabled: opts.enabled ?? true,
    });
  }

  function useDetail(id: number | null) {
    return useQuery({
      queryKey: keys.detail(id ?? 0),
      queryFn: ({ signal }) => api.get(`${path}/${id}`, { schema: config.detail, signal }),
      enabled: id !== null,
    });
  }

  function useOptions(query: Query = {}) {
    const schema = (config.options ?? z.unknown()) as O;
    return useQuery({
      queryKey: keys.options(query),
      queryFn: ({ signal }) => api.get(`${path}/options`, { query, schema, signal }),
      enabled: Boolean(config.options),
      staleTime: 5 * 60_000,
    });
  }

  /**
   * Create (`id === null` → POST) or update (PUT; multipart bodies use
   * `POST /:id` because not every client sends a multipart PUT, admin-api.md §7).
   */
  function useSave(id: number | null) {
    const invalidate = useInvalidate();
    return useMutation({
      mutationFn: (body: unknown) => {
        if (id === null) return api.post(path, body, { schema: writeSchema });
        if (typeof FormData !== 'undefined' && body instanceof FormData) {
          return api.post(`${path}/${id}`, body, { schema: writeSchema });
        }
        return api.put(`${path}/${id}`, body, { schema: writeSchema });
      },
      onSuccess: () => invalidate(),
    });
  }

  /** `POST /:id/<action>` (toggle, approve, regenerate, …). */
  function useAction(action: string) {
    const invalidate = useInvalidate();
    return useMutation({
      mutationFn: ({ id, body }: { id: number; body?: unknown }) =>
        api.post(`${path}/${id}/${action}`, body, { schema: writeSchema }),
      onSuccess: () => invalidate(),
    });
  }

  function useRemove() {
    const invalidate = useInvalidate();
    return useMutation({
      mutationFn: (id: number) => api.delete(`${path}/${id}`, { schema: writeSchema }),
      onSuccess: (_data, id) => invalidate(id),
    });
  }

  return { keys, path, useList, useDetail, useOptions, useSave, useAction, useRemove };
}

/** The 422 bag of a failed write (for TranslatableField `errors` and `fieldError`). */
export function fieldErrorsOf(error: unknown): Record<string, string[]> | undefined {
  return error instanceof ApiError ? error.fieldErrors : undefined;
}

/** First message for one field of a failed write. */
export function fieldError(error: unknown, name: string): string | undefined {
  return error instanceof ApiError ? error.field(name) : undefined;
}
