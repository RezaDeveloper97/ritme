'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { apiClient } from '@/shared/api';
import { notificationKeys } from '@/entities/notification';

/**
 * Marking notifications as read (CLAUDE.md §8.1). Both mutations invalidate
 * through the entity's key factory so the list and unread count refetch —
 * never hand-written key arrays.
 *
 * Privacy (§11): only the opaque notification id crosses the wire; nothing
 * here is logged.
 */

/** POST /home/notifications/{id}/read — marks a single notification read. */
export function useMarkNotificationRead() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, string>({
    mutationFn: async (id) => {
      await apiClient.post(`/home/notifications/${id}/read`);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    },
  });
}

/** POST /home/notifications/read-all — marks every notification read. */
export function useMarkAllNotificationsRead() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, void>({
    mutationFn: async () => {
      await apiClient.post('/home/notifications/read-all');
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    },
  });
}
