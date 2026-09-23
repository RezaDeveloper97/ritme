'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { apiClient } from '@/shared/api';
import { applyIntake, type CareToday, careKeys } from '@/entities/care-reminder';

/*
 * Tick / untick one dose (home «یادآورهای امروز», Reminders «امروز» strip).
 * The tick lands instantly: every cached `careKeys.today` for that date is
 * updated optimistically, rolled back on failure, and refetched afterwards.
 *
 * Privacy (§11): which dose was taken when is health data — API only.
 */

export interface LogIntakeVars {
  reminderId: number;
  /** `Y-m-d` — the day being shown (the `date` of the cached CareToday). */
  date: string;
  /** `HH:MM`. */
  slot: string;
  taken: boolean;
}

type Snapshot = Array<[readonly unknown[], CareToday | undefined]>;

export function useLogIntake() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, LogIntakeVars, { snapshot: Snapshot }>({
    mutationFn: async ({ reminderId, date, slot, taken }) => {
      const path = `/care/medications/${reminderId}/intakes`;
      if (taken) {
        await apiClient.post(path, { date, slot });
      } else {
        // The contract puts `{date, slot}` in the DELETE body, but the shared
        // client sends DELETE without one (shared/api is outside this slice),
        // so they ride as query params — dates already do on /health-logs/{date}.
        await apiClient.delete(path, { params: { date, slot } });
      }
    },
    onMutate: async ({ reminderId, date, slot, taken }) => {
      await queryClient.cancelQueries({ queryKey: careKeys.todayAll() });
      const snapshot = queryClient.getQueriesData<CareToday>({ queryKey: careKeys.todayAll() });
      for (const [key, today] of snapshot) {
        if (today && today.date === date) {
          queryClient.setQueryData(key, applyIntake(today, { reminderId, slot, taken }));
        }
      }
      return { snapshot };
    },
    onError: (_error, _vars, context) => {
      for (const [key, today] of context?.snapshot ?? []) queryClient.setQueryData(key, today);
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: careKeys.todayAll() });
    },
  });
}
