'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { cycleKeys } from '@/entities/cycle';
import { messageKeys } from '@/entities/message';
import { userKeys } from '@/entities/user';
import { type ApiEnvelope, apiClient } from '@/shared/api';
import { today, toApiDate } from '@/shared/lib/date';
import { isAuthenticated } from '@/shared/session';

/**
 * Query key for the period logging state. Kept OUTSIDE `cycleKeys.all` so the
 * broad cycle invalidation can't race a refetch on top of the authoritative
 * value we write from the start/end response (see below).
 */
const periodStatusKey = ['period-status'] as const;

const periodStatusSchema = z.object({
  active: z.boolean(),
  period_start_date: z.string().nullable().optional(),
  period_end_date: z.string().nullable().optional(),
});

export type PeriodStatus = z.infer<typeof periodStatusSchema>;

/**
 * GET /cycle/period/status — whether the user currently has an ongoing period,
 * so the button can offer "Start" vs "End". Disabled until authenticated so it
 * never fires on public screens.
 */
export function usePeriodStatus() {
  return useQuery({
    queryKey: periodStatusKey,
    queryFn: async (): Promise<PeriodStatus> => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/cycle/period/status');
      return periodStatusSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: false,
  });
}

/**
 * Invalidate every cache that reflects the cycle after a period is logged
 * (calculations, the daily message, the profile) so the UI refetches the fresh
 * state. The period-status cache is set authoritatively by the caller instead.
 */
function useInvalidateCycle() {
  const queryClient = useQueryClient();
  return () => {
    void queryClient.invalidateQueries({ queryKey: cycleKeys.all });
    void queryClient.invalidateQueries({ queryKey: messageKeys.all });
    void queryClient.invalidateQueries({ queryKey: userKeys.all });
  };
}

/** Extract the period status the start/end endpoints echo back in their response. */
async function postPeriod(path: string, date: string): Promise<PeriodStatus> {
  const { data } = await apiClient.post<ApiEnvelope<unknown>>(path, { date });
  return periodStatusSchema.parse(data.data);
}

/**
 * POST /cycle/period/start — record that the user's period started on `date`
 * (defaults to today). This re-anchors the cycle on the backend; the response
 * carries the new period status, which we write straight into the cache so the
 * button flips to "End period" immediately — without depending on a refetch.
 *
 * Sensitive health data (§11): the period date is sent to the API and never
 * logged. `date` crosses the boundary as Gregorian `YYYY-MM-DD`, produced only
 * via `@/shared/lib/date` (§7).
 */
export function useStartPeriod() {
  const queryClient = useQueryClient();
  const invalidate = useInvalidateCycle();
  return useMutation<PeriodStatus, unknown, { date?: string } | void>({
    mutationFn: (input) => postPeriod('/cycle/period/start', (input && input.date) || toApiDate(today())),
    onSuccess: (status) => {
      queryClient.setQueryData(periodStatusKey, status);
      invalidate();
    },
  });
}

/**
 * POST /cycle/period/end — close the ongoing period with an end date (defaults
 * to today). Writes the returned status into the cache so the button flips back
 * to "Start period" at once.
 */
export function useEndPeriod() {
  const queryClient = useQueryClient();
  const invalidate = useInvalidateCycle();
  return useMutation<PeriodStatus, unknown, { date?: string } | void>({
    mutationFn: (input) => postPeriod('/cycle/period/end', (input && input.date) || toApiDate(today())),
    onSuccess: (status) => {
      queryClient.setQueryData(periodStatusKey, status);
      invalidate();
    },
  });
}
