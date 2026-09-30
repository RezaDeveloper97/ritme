'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { cycleKeys } from '@/entities/cycle';
import { userKeys } from '@/entities/user';
import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { applyPatch, type CycleSettings, type CycleSettingsPatch, cycleSettingsSchema } from '../model/settings';

/** Query-key factory (CLAUDE.md §8). */
export const cycleSettingsKeys = {
  all: ['cycle-settings'] as const,
};

/**
 * Caches the save makes stale: manual lengths re-run the cycle engine (every
 * cycle query, the profile), and the reminder switches are shared with the
 * /profile/notifications screen (a sibling slice, hence its literal key).
 */
const DEPENDENT_KEYS: readonly (readonly string[])[] = [cycleKeys.all, userKeys.profile(), ['notification-settings']];

const PATH = '/profile/cycle-settings';

/** GET /profile/cycle-settings — automatic lengths and default reminders when never saved. */
export function useCycleSettings() {
  return useQuery({
    queryKey: cycleSettingsKeys.all,
    queryFn: async (): Promise<CycleSettings> => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(PATH);
      return cycleSettingsSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}

/**
 * PUT /profile/cycle-settings with a partial body. Switches and steppers
 * update at once (optimistic); a failed save rolls the cache back and the
 * screen shows the error line.
 */
export function useUpdateCycleSettings() {
  const queryClient = useQueryClient();
  const key = cycleSettingsKeys.all;
  return useMutation<CycleSettings, unknown, CycleSettingsPatch, { previous?: CycleSettings }>({
    mutationKey: [...key, 'update'],
    mutationFn: async (patch) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(PATH, patch);
      return cycleSettingsSchema.parse(data.data);
    },
    onMutate: async (patch) => {
      await queryClient.cancelQueries({ queryKey: key });
      const previous = queryClient.getQueryData<CycleSettings>(key);
      if (previous) queryClient.setQueryData(key, applyPatch(previous, patch));
      return { previous };
    },
    onError: (_error, _patch, context) => {
      if (context?.previous) queryClient.setQueryData(key, context.previous);
    },
    onSuccess: (saved) => {
      // Only the last save in flight writes the server copy, so an older
      // response never undoes a newer optimistic change.
      if (queryClient.isMutating({ mutationKey: [...key, 'update'] }) <= 1) queryClient.setQueryData(key, saved);
      for (const k of DEPENDENT_KEYS) void queryClient.invalidateQueries({ queryKey: [...k] });
    },
  });
}
