'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

/**
 * «اعلان‌های محرمانه» (CB-PRIV-01) is B-N1-11's «متن خنثی» — the `neutral_copy` field of
 * `/profile/notification-settings`, not a second flag. The backend applies it to every push, web-push and SMS
 * sender. Cached under the notification-settings screen's literal key (`['notification-settings']`, a sibling
 * slice) as a sub-key, so a save here refreshes that screen and the other way round.
 */
const ROOT = ['notification-settings'] as const;
export const discreetKeys = {
  flag: () => [...ROOT, 'discreet'] as const,
};

const PATH = '/profile/notification-settings';

/** `neutral_copy` of a notification-settings body (missing → the server default, on). */
export const discreetSchema = z
  .object({ neutral_copy: z.boolean().default(true) })
  .passthrough()
  .transform((raw) => raw.neutral_copy);

/** GET /profile/notification-settings → the discreet flag. */
export function useDiscreetNotifications() {
  return useQuery({
    queryKey: discreetKeys.flag(),
    queryFn: async (): Promise<boolean> => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(PATH);
      return discreetSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 0,
    retry: 1,
  });
}

/** PUT /profile/notification-settings `{neutral_copy}` — optimistic, rolled back on error. */
export function useSetDiscreetNotifications() {
  const queryClient = useQueryClient();
  const key = discreetKeys.flag();
  return useMutation<boolean, unknown, boolean, { previous?: boolean }>({
    mutationFn: async (on) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(PATH, { neutral_copy: on });
      return discreetSchema.parse(data.data);
    },
    onMutate: async (on) => {
      await queryClient.cancelQueries({ queryKey: key });
      const previous = queryClient.getQueryData<boolean>(key);
      queryClient.setQueryData(key, on);
      return { previous };
    },
    onError: (_error, _on, context) => {
      if (context?.previous !== undefined) queryClient.setQueryData(key, context.previous);
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: ROOT }),
  });
}
