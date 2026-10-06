'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { api, createResource, pagedSchema } from '@/shared/api';

/**
 * `Message` — one smart-message row (admin-api.md §11). Empty payloads arrive as `[]`.
 * Values are kept raw: untyped groups hold texts / text lists, the registered pregnancy
 * groups (§13) also numbers, booleans, objects and object lists.
 */
export const messageSchema = z.object({
  id: z.number(),
  group: z.string(),
  item_key: z.string(),
  locale: z.string(),
  label: z.string().nullable(),
  payload: z.preprocess((v) => (v && typeof v === 'object' && !Array.isArray(v) ? v : {}), z.record(z.string(), z.unknown())),
  is_active: z.boolean(),
  is_approved: z.boolean(),
  sort_order: z.number(),
});
export type Message = z.infer<typeof messageSchema>;

const missingSchema = z.object({
  group: z.string(),
  item_key: z.string(),
  locale: z.string(),
});
export type MissingMessage = z.infer<typeof missingSchema>;

export const messagesApi = createResource({
  key: 'messages',
  path: '/messages',
  list: pagedSchema(messageSchema, {
    groups: z.array(z.string()),
    locales: z.array(z.string()),
    registered_groups: z.array(z.string()).optional().default([]),
    missing: z.array(missingSchema).optional().default([]),
    super_only_groups: z.array(z.string()).optional(),
  }),
  detail: z.object({ message: messageSchema }),
  alsoInvalidate: [['dashboard']],
});

/** One payload field of a registered item (admin-api.md §13 `Schema`). */
export interface SchemaField {
  key: string;
  kind: 'text' | 'text_list' | 'integer' | 'boolean' | 'enum' | 'enum_list' | 'url' | 'object' | 'object_list';
  nullable: boolean;
  max_length?: number;
  min?: number;
  max?: number;
  values?: string[];
  min_items?: number;
  max_items?: number;
  fields?: SchemaField[];
}
export const schemaFieldSchema: z.ZodType<SchemaField> = z.lazy(() =>
  z.object({
    key: z.string(),
    kind: z.enum(['text', 'text_list', 'integer', 'boolean', 'enum', 'enum_list', 'url', 'object', 'object_list']),
    nullable: z.boolean(),
    max_length: z.number().optional(),
    min: z.number().optional(),
    max: z.number().optional(),
    values: z.array(z.string()).optional(),
    min_items: z.number().optional(),
    max_items: z.number().optional(),
    fields: z.array(schemaFieldSchema).optional(),
  }),
);

const registryItemSchema = z.object({
  group: z.string(),
  item_key: z.string(),
  typed: z.boolean(),
  fields: z.array(schemaFieldSchema),
  placeholders: z.array(z.string()).nullable().catch([]),
  existing: z
    .array(z.object({ id: z.number(), locale: z.string() }))
    .nullable()
    .catch([]),
  template: z.preprocess((v) => (v && typeof v === 'object' && !Array.isArray(v) ? v : {}), z.record(z.string(), z.unknown())),
});
export type RegistryItem = z.infer<typeof registryItemSchema>;

/** GET /messages/registry/:group/:key — the item's schema and a template in `locale`. Unregistered → 404. */
export function useRegistryItem(group: string, key: string, locale?: string) {
  return useQuery({
    queryKey: [...messagesApi.keys.all, 'registry', group, key, locale ?? ''],
    queryFn: ({ signal }) =>
      api.get(`/messages/registry/${encodeURIComponent(group)}/${encodeURIComponent(key)}`, {
        query: { locale },
        schema: registryItemSchema,
        signal,
      }),
    retry: false,
  });
}
