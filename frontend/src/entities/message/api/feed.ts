'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, ApiError, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { FEED_SLOTS, type FeedItem, type MessageEventName, type MessageFeed } from '../model/feed';
import { parseFeed } from './feed-schema';
import { messageKeys } from './queries';

/** Every home slot, in the server's order. One cache entry, so every widget shares one request. */
export const HOME_FEED_SLOTS = FEED_SLOTS;

export const feedKeys = {
  all: [...messageKeys.all, 'feed'] as const,
  home: () => [...feedKeys.all, HOME_FEED_SLOTS.join(',')] as const,
};

/**
 * Is this the API saying "this backend has no feed"? Production still runs Laravel, which answers 404 (or 405) to
 * the route. That is a settled answer ("use the legacy endpoints"), not a failure to retry.
 */
export function isFeedUnsupported(error: unknown): boolean {
  if (!(error instanceof ApiError)) return false;
  const status = error.response?.status;
  return status === 404 || status === 405;
}

/**
 * GET /messages/feed — every slot in one call, or `null` when the backend has no feed (Laravel). Any other failure
 * (network, 5xx, offline) throws, and the home then behaves exactly as before the feed existed.
 *
 * Privacy (§11): the text is display-only health copy and is never logged.
 */
export async function fetchFeed(slots: readonly string[] = HOME_FEED_SLOTS): Promise<MessageFeed | null> {
  try {
    const { data } = await apiClient.get<ApiEnvelope<unknown>>('/messages/feed', {
      params: { slots: slots.join(',') },
    });
    return parseFeed(data.data);
  } catch (error) {
    if (isFeedUnsupported(error)) return null;
    throw error;
  }
}

/** The home feed. `data === null` / `isError` both mean "legacy path" — see {@link feedState}. */
export function useMessageFeed(enabled = true) {
  return useQuery({
    queryKey: feedKeys.home(),
    queryFn: () => fetchFeed(),
    enabled: isAuthenticated() && enabled,
    staleTime: 60_000,
    retry: false,
  });
}

/** POST /messages/events — `clicked` | `dismissed` | `goal`. Never `shown`: the server logs it. */
export async function postMessageEvent(item: Pick<FeedItem, 'id' | 'variant' | 'version' | 'slot'>, event: MessageEventName) {
  await apiClient.post('/messages/events', {
    rule_id: item.id,
    variant: item.variant,
    version: item.version,
    slot: item.slot,
    event,
  });
}

/**
 * Report a click or a dismissal. Fire-and-forget: telemetry must never block or break the action itself, so
 * failures are swallowed. A dismissal refreshes the feed (the server counts it against the cap).
 */
export function useMessageEvent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ item, event }: { item: FeedItem; event: MessageEventName }) => postMessageEvent(item, event),
    onSuccess: (_d, { event }) => {
      if (event === 'dismissed') void queryClient.invalidateQueries({ queryKey: feedKeys.all });
    },
    onError: () => undefined,
  });
}

const alertRowsSchema = z.object({
  items: z.array(z.object({ id: z.number(), rule_id: z.number(), state: z.string() })).catch([]),
  next_cursor: z.string().nullish(),
});

/**
 * The inbox row id of an open alert opened for rule `ruleId`. The feed's `id` is the *rule* id, while
 * `/alerts/{id}/ack` takes the inbox row id, so the row is looked up in `GET /alerts` (at most 3 pages).
 */
export async function findAlertRowId(ruleId: number): Promise<number | null> {
  let cursor: string | undefined;
  for (let page = 0; page < 3; page += 1) {
    const { data } = await apiClient.get<ApiEnvelope<unknown>>('/alerts', {
      params: { state: 'open', ...(cursor ? { cursor } : {}) },
    });
    const parsed = alertRowsSchema.parse(data.data);
    const row = parsed.items.find((r) => r.rule_id === ruleId && r.state === 'active');
    if (row) return row.id;
    if (!parsed.next_cursor) return null;
    cursor = parsed.next_cursor;
  }
  return null;
}

/**
 * POST /alerts/{id}/ack — «دیدم». No active row left for the rule (already acknowledged elsewhere) counts as
 * done. Refreshes the feed and the alerts counters afterwards.
 */
export function useAckAlert() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ ruleId, alertId }: { ruleId: number; alertId?: number | null }) => {
      // The feed's alert_id is the row id (one request); the GET /alerts lookup is only the fallback.
      const rowId = alertId ?? (await findAlertRowId(ruleId));
      if (rowId !== null) await apiClient.post(`/alerts/${rowId}/ack`);
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: feedKeys.all });
      void queryClient.invalidateQueries({ queryKey: ['alerts'] });
    },
  });
}
