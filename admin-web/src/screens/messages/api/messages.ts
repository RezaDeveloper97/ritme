'use client';

import { z } from 'zod';

import { createResource, pagedSchema } from '@/shared/api';

/** A payload value: a text, or a list of texts (one per line in the editor). */
const payloadValueSchema = z.union([z.string(), z.array(z.string())]).catch('');

/** `Message` — one smart-message row (admin-api.md §11). Empty payloads arrive as `[]`. */
export const messageSchema = z.object({
  id: z.number(),
  group: z.string(),
  item_key: z.string(),
  locale: z.string(),
  label: z.string().nullable(),
  payload: z.preprocess(
    (v) => (v && typeof v === 'object' && !Array.isArray(v) ? v : {}),
    z.record(z.string(), payloadValueSchema),
  ),
  is_active: z.boolean(),
  is_approved: z.boolean(),
  sort_order: z.number(),
});
export type Message = z.infer<typeof messageSchema>;

export const messagesApi = createResource({
  key: 'messages',
  path: '/messages',
  list: pagedSchema(messageSchema, {
    groups: z.array(z.string()),
    locales: z.array(z.string()),
  }),
  detail: z.object({ message: messageSchema }),
  alsoInvalidate: [['dashboard']],
});
