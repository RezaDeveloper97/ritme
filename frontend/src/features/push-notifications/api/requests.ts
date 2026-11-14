import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { type NativeReminderItem, type PushSubscriptionBody, sanitizeAppPath } from '@/shared/pwa';

/*
 * Push + upcoming-dose requests. Privacy (§11): the dose titles/bodies in
 * /care/upcoming are health data — they go straight to the native bridge and
 * are never stored, logged or put in a query cache.
 */

/** GET /push/config → the VAPID public key, or null when web push is off on this server. */
export async function fetchPushPublicKey(): Promise<string | null> {
  const { data } = await apiClient.get<ApiEnvelope<{ public_key?: string | null }>>('/push/config');
  const key = data.data?.public_key;
  return typeof key === 'string' && key.length > 0 ? key : null;
}

/** POST /push/subscriptions — idempotent. */
export async function postPushSubscription(body: PushSubscriptionBody): Promise<void> {
  await apiClient.post('/push/subscriptions', body);
}

/** DELETE /push/subscriptions `{endpoint}`. */
export async function deletePushSubscription(endpoint: string): Promise<void> {
  await apiClient.delete('/push/subscriptions', { data: { endpoint } });
}

const upcomingSchema = z
  .object({
    items: z
      .array(
        z
          .object({
            id: z.union([z.string(), z.number()]),
            at_ms: z.number().finite(),
            title: z.string(),
            body: z.string().default(''),
            url: z.string().default('/'),
          })
          .passthrough(),
      )
      .default([]),
  })
  .passthrough();

/** GET /care/upcoming?hours=168 → what the shell should arm (already filtered by the server). */
export async function fetchUpcomingReminders(hours = 168): Promise<NativeReminderItem[]> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/care/upcoming', { params: { hours } });
  const parsed = upcomingSchema.parse(data.data ?? {});
  return parsed.items.map((item) => ({
    id: item.id,
    at_ms: item.at_ms,
    title: item.title,
    body: item.body,
    url: sanitizeAppPath(item.url),
  }));
}
