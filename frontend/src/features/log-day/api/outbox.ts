import type { LogDayChanges } from '@/entities/health-log';
import { apiClient, getApiErrorStatus } from '@/shared/api';
import { getOutbox, type OutboxEntry, type SendResult } from '@/shared/lib/outbox';

/**
 * Offline saves of the log sheet (CB-MENO-06) through bloom's outbox (`shared/lib/outbox`, T-M7-12). The
 * PUT is partial, so a second offline save of the same day is merged into the queued body param by
 * param (the later value wins) instead of replacing it — replaying the merged body is idempotent.
 */

/** Outbox key of one day's log. */
export const logDayOutboxKey = (date: string) => `log-day:${date}`;

/** The PUT body the outbox replays. */
export interface QueuedLogDay {
  categories: LogDayChanges;
  voice_params?: string[];
}

/** Merges a newer partial body into the queued one: params of `next` override, the rest are kept. */
export function mergeQueuedDay(queued: QueuedLogDay | null, next: QueuedLogDay): QueuedLogDay {
  if (!queued) return next;
  const categories: LogDayChanges = {};
  for (const [cat, params] of Object.entries(queued.categories)) categories[cat] = { ...params };
  for (const [cat, params] of Object.entries(next.categories)) categories[cat] = { ...(categories[cat] ?? {}), ...params };
  const voice = [...new Set([...(queued.voice_params ?? []), ...(next.voice_params ?? [])])].filter((key) => {
    // A param the newer save sets by hand is no longer a voice value.
    const [cat, param] = key.split('.');
    const fromNext = next.categories[cat]?.[param];
    return fromNext === undefined || (next.voice_params ?? []).includes(key);
  });
  return voice.length ? { categories, voice_params: voice } : { categories };
}

/** No HTTP response at all = the network, not the server, failed. */
export const isOfflineError = (error: unknown) => getApiErrorStatus(error) === undefined;

/** Queues (or merges into) the day's pending PUT. */
export async function queueLogDay(date: string, body: QueuedLogDay): Promise<void> {
  const outbox = getOutbox();
  const key = logDayOutboxKey(date);
  const pending = (await outbox.pending()).find((e) => e.key === key);
  await outbox.enqueue({
    key,
    method: 'put',
    url: `/logs/days/${date}`,
    body: mergeQueuedDay((pending?.body as QueuedLogDay | undefined) ?? null, body),
  });
}

/**
 * Replays one queued write. Entries are self-describing (method + url + body) and the outbox is shared,
 * so a replay started here also sends another screen's queued writes (pregnancy days) the same way.
 */
export async function sendQueuedEntry(entry: OutboxEntry): Promise<SendResult> {
  try {
    if (entry.method === 'delete') await apiClient.delete(entry.url);
    else await apiClient[entry.method](entry.url, entry.body);
    return 'sent';
  } catch (error) {
    // 5xx may pass on retry; 4xx (validation) never will.
    const status = getApiErrorStatus(error);
    return status === undefined || status >= 500 ? 'offline' : 'rejected';
  }
}
