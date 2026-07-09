'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { cycleKeys } from '@/entities/cycle';
import { messageKeys } from '@/entities/message';
import { userKeys } from '@/entities/user';
import { apiClient } from '@/shared/api';
import { today, toApiDate } from '@/shared/lib/date';

/**
 * POST /profile — record that the user's period started on `date` (defaults to
 * today). Setting `last_period_start` re-anchors the cycle, so the backend
 * re-runs its calculations; we then invalidate every cache that reflects the
 * cycle (calculations, the daily message, the profile) so the UI refetches the
 * fresh state.
 *
 * Sensitive health data (§11): the period date is sent to the API and never
 * logged. `date` crosses the boundary as Gregorian `YYYY-MM-DD`, produced only
 * via `@/shared/lib/date` (§7).
 */
export function useStartPeriod() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, { date?: string } | void>({
    mutationFn: async (input) => {
      const date = (input && input.date) || toApiDate(today());
      await apiClient.post('/profile', { last_period_start: date });
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: cycleKeys.all });
      void queryClient.invalidateQueries({ queryKey: messageKeys.all });
      void queryClient.invalidateQueries({ queryKey: userKeys.all });
    },
  });
}
