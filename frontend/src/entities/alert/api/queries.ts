'use client';

import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { AlertListState, AlertPage } from '../model/types';
import { alertKeys } from './keys';
import { parseAlertPage } from './schema';

/** GET /alerts?state=open|history&cursor= — 20 per page, ordered by level then newest. */
export async function fetchAlerts(state: AlertListState, cursor?: string | null): Promise<AlertPage> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/alerts', {
    params: { state, ...(cursor ? { cursor } : {}) },
  });
  return parseAlertPage(data.data);
}

export function useAlerts(state: AlertListState) {
  return useInfiniteQuery({
    queryKey: alertKeys.list(state),
    queryFn: ({ pageParam }) => fetchAlerts(state, pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor,
    enabled: isAuthenticated(),
    retry: false,
  });
}

export type AlertAction = 'ack' | 'visit-note' | 'dismiss';

/** POST /alerts/{id}/ack | visit-note | dismiss. The list, the home counters and the feed all refresh afterwards. */
export function useAlertAction() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, { id: number; action: AlertAction }>({
    mutationFn: async ({ id, action }) => {
      await apiClient.post(`/alerts/${id}/${action}`);
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: alertKeys.all });
      // The home bell reads `alerts` counts from the message feed (entities/message `feedKeys.all`).
      void queryClient.invalidateQueries({ queryKey: ['message', 'feed'] });
    },
  });
}
