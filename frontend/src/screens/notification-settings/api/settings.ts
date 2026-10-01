'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

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

const pmsDaySchema = z
  .object({
    reminders: z
      .array(z.object({ code: z.string(), cycle_day: z.number().int().nullable().optional() }).passthrough())
      .default([]),
  })
  .passthrough()
  .transform((raw) => raw.reminders.find((r) => r.code === 'pms')?.cycle_day ?? null);

/**
 * The PMS reminder's cycle day from GET /profile/cycle-settings — the value
 * `/cycle/settings` shows (n1-stage B-1), so both screens name the same day.
 * Its own key under the sibling slice's literal `cycle-settings` prefix: a save
 * there invalidates `['cycle-settings']` and repaints this row too.
 */
export function usePmsReminderDay() {
  return useQuery({
    queryKey: ['cycle-settings', 'pms-day'] as const,
    queryFn: async (): Promise<number | null> => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/profile/cycle-settings');
      return pmsDaySchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}
