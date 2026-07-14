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

/** Query key for the logged-period history, outside `cycleKeys.all` like status. */
const periodHistoryKey = ['period-history'] as const;

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
    // The logged-period list changed too — without this the calendar keeps a
    // stale history and re-offers "start period" right after one was logged.
    void queryClient.invalidateQueries({ queryKey: periodHistoryKey });
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
 * Log a whole bleeding range in one action (the calendar's "mark period days"
 * flow). `start` re-anchors the cycle on the backend — which cascades every
 * future prediction — and `end` (when the range is finished) closes it. Both
 * dates cross the boundary as Gregorian `YYYY-MM-DD` from `@/shared/lib/date`
 * (§7) and are never logged (§11). Runs start→end sequentially because the end
 * endpoint operates on the period the start call just anchored.
 */
export function useLogPeriodRange() {
  const queryClient = useQueryClient();
  const invalidate = useInvalidateCycle();
  return useMutation<PeriodStatus, unknown, { start: string; end?: string }>({
    mutationFn: async ({ start, end }) => {
      let status = await postPeriod('/cycle/period/start', start);
      if (end) status = await postPeriod('/cycle/period/end', end);
      return status;
    },
    onSuccess: (status) => {
      queryClient.setQueryData(periodStatusKey, status);
      invalidate();
    },
  });
}

const loggedPeriodSchema = z.object({
  id: z.number(),
  period_start_date: z.string(),
  period_end_date: z.string().nullable(),
});

export type LoggedPeriod = z.infer<typeof loggedPeriodSchema>;

/**
 * GET /cycle/period/history — every period the user actually logged (as opposed
 * to predictions), so the calendar can offer "edit this period" on its days.
 */
export function usePeriodHistory() {
  return useQuery({
    queryKey: periodHistoryKey,
    queryFn: async (): Promise<LoggedPeriod[]> => {
      const { data } = await apiClient.get<ApiEnvelope<{ periods: unknown }>>('/cycle/period/history');
      return z.array(loggedPeriodSchema).parse(data.data?.periods ?? []);
    },
    enabled: isAuthenticated(),
    staleTime: 60_000,
  });
}

/** Invalidate the caches a period edit/delete touches (history + everything cyclic). */
function useInvalidateAfterEdit() {
  const queryClient = useQueryClient();
  const invalidate = useInvalidateCycle();
  return () => {
    void queryClient.invalidateQueries({ queryKey: periodHistoryKey });
    void queryClient.invalidateQueries({ queryKey: periodStatusKey });
    invalidate();
  };
}

/**
 * PUT /cycle/period/{id} — move a logged period's start/end. The backend
 * re-anchors the cycle and regenerates all predictions. Dates are Gregorian
 * `YYYY-MM-DD` from `@/shared/lib/date` (§7) and never logged (§11).
 */
export function useUpdatePeriod() {
  const invalidateAll = useInvalidateAfterEdit();
  return useMutation<void, unknown, { id: number; start: string; end?: string | null }>({
    mutationFn: async ({ id, start, end }) => {
      await apiClient.put(`/cycle/period/${id}`, { start_date: start, end_date: end ?? null });
    },
    onSuccess: invalidateAll,
  });
}

/**
 * DELETE /cycle/period/{id} — remove a logged period entirely. The backend
 * re-anchors on the most recent remaining period (or clears the anchor).
 */
export function useDeletePeriod() {
  const invalidateAll = useInvalidateAfterEdit();
  return useMutation<void, unknown, { id: number }>({
    mutationFn: async ({ id }) => {
      await apiClient.delete(`/cycle/period/${id}`);
    },
    onSuccess: invalidateAll,
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
