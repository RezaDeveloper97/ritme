'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import {
  applyPatch,
  type NotificationSettings,
  type SettingsPatch,
  settingsSchema,
} from '../model/settings';

/** Query-key factory (CLAUDE.md §8). */
export const notificationSettingsKeys = {
  all: ['notification-settings'] as const,
};

const PATH = '/profile/notification-settings';

/** GET /profile/notification-settings — defaults when the user never saved. */
export function useNotificationSettings() {
  return useQuery({
    queryKey: notificationSettingsKeys.all,
    queryFn: async (): Promise<NotificationSettings> => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(PATH);
      return settingsSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: 1,
  });
}

/**
 * PUT /profile/notification-settings with a partial body. Switches flip at
 * once (optimistic); a failed save rolls the cache back and the screen shows
 * the error line.
 */
export function useUpdateNotificationSettings() {
  const queryClient = useQueryClient();
  const key = notificationSettingsKeys.all;
  return useMutation<NotificationSettings, unknown, SettingsPatch, { previous?: NotificationSettings }>({
    mutationKey: [...key, 'update'],
    mutationFn: async (patch) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(PATH, patch);
      return settingsSchema.parse(data.data);
    },
    onMutate: async (patch) => {
      await queryClient.cancelQueries({ queryKey: key });
      const previous = queryClient.getQueryData<NotificationSettings>(key);
      if (previous) queryClient.setQueryData(key, applyPatch(previous, patch));
      return { previous };
    },
    onError: (_error, _patch, context) => {
      if (context?.previous) queryClient.setQueryData(key, context.previous);
    },
    onSuccess: (saved) => {
      // Rapid taps overlap: only the last save in flight writes the server copy,
      // so an older response never undoes a newer optimistic flip.
      if (queryClient.isMutating({ mutationKey: [...key, 'update'] }) <= 1) queryClient.setQueryData(key, saved);
    },
  });
}
