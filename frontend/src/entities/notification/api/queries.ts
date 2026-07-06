'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { NotificationList } from '../model/types';
import { notificationListSchema } from './schema';

/** Query-key factory for notifications (CLAUDE.md §8). */
export const notificationKeys = {
  all: ['notification'] as const,
  list: (perPage: number = 50) => [...notificationKeys.all, 'list', perPage] as const,
};

/**
 * GET /home/notifications — the user's in-app notifications plus unread count.
 *
 * Privacy (§11): notification titles/bodies may reference health topics; the
 * payload is display-only and must never be logged.
 */
export async function fetchNotifications(perPage: number = 50): Promise<NotificationList> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/home/notifications', {
    params: { per_page: perPage },
  });
  return notificationListSchema.parse(data.data);
}

/** Reads the notification list (§8 — server state stays in TanStack Query). */
export function useNotifications(perPage: number = 50) {
  return useQuery({
    queryKey: notificationKeys.list(perPage),
    queryFn: () => fetchNotifications(perPage),
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: false,
  });
}
