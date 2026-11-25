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
import { applyReminderPatch, type Reminder, type RemindersPatch, remindersSchema } from '../model/reminders';

/** Query-key factory (CLAUDE.md §8). */
export const notificationSettingsKeys = {
  all: ['notification-settings'] as const,
};

/**
 * The cycle-settings cache root: the cycle-settings screen's own key (a
 * sibling slice, hence the literal). Reminders live under it so a save on
 * either screen repaints the other.
 */
const CYCLE_SETTINGS_KEY = ['cycle-settings'] as const;
export const cycleRemindersKeys = {
  all: [...CYCLE_SETTINGS_KEY, 'reminders'] as const,
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
      // The cycle reminders' switches are these categories (stored once, server side).
      void queryClient.invalidateQueries({ queryKey: cycleRemindersKeys.all });
    },
  });
}

// ── Cycle reminders (merged in from /cycle/settings) ─────────────────────────

const CYCLE_SETTINGS_PATH = '/profile/cycle-settings';

/** The timed cycle reminders from GET /profile/cycle-settings (schedule + pill). */
export function useCycleReminders() {
  return useQuery({
    queryKey: cycleRemindersKeys.all,
    queryFn: async (): Promise<Reminder[]> => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(CYCLE_SETTINGS_PATH);
      return remindersSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}

/**
 * PUT /profile/cycle-settings with only `reminders` (time, days before, the
 * pill switch). Optimistic like the category switches; the server stores the
 * switches as notification preferences, so the categories are refetched too.
 */
export function useUpdateCycleReminders() {
  const queryClient = useQueryClient();
  const key = cycleRemindersKeys.all;
  return useMutation<Reminder[], unknown, RemindersPatch, { previous?: Reminder[] }>({
    mutationKey: [...key, 'update'],
    mutationFn: async (patch) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(CYCLE_SETTINGS_PATH, { reminders: patch });
      return remindersSchema.parse(data.data);
    },
    onMutate: async (patch) => {
      await queryClient.cancelQueries({ queryKey: key });
      const previous = queryClient.getQueryData<Reminder[]>(key);
      if (previous) queryClient.setQueryData(key, applyReminderPatch(previous, patch));
      return { previous };
    },
    onError: (_error, _patch, context) => {
      if (context?.previous) queryClient.setQueryData(key, context.previous);
    },
    onSuccess: (saved) => {
      if (queryClient.isMutating({ mutationKey: [...key, 'update'] }) <= 1) queryClient.setQueryData(key, saved);
      void queryClient.invalidateQueries({ queryKey: [...CYCLE_SETTINGS_KEY], exact: true });
      void queryClient.invalidateQueries({ queryKey: notificationSettingsKeys.all });
    },
  });
}
