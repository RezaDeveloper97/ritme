'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

/**
 * «دفعه بعد سریع‌تر» (nbl_Voice_Saved): the nightly 21:00 nudge is the existing `daily_log` reminder of
 * `/profile/cycle-settings` (switch on the notification preferences + its Tehran wall-clock time) — no new
 * backend. The card is on when that reminder is enabled at 21:00.
 */
export const VOICE_REMINDER_TIME = '21:00';

/**
 * Under the cycle-settings screen's literal key (`['cycle-settings']`, a sibling slice) so its saves
 * refresh this card and ours refresh it; a sub-key because the cached shape differs.
 */
const KEY = ['cycle-settings', 'voice-reminder'] as const;
const PATH = '/profile/cycle-settings';

const reminderSchema = z.object({
  reminders: z
    .array(z.object({ code: z.string(), enabled: z.boolean(), time: z.string().nullable().optional() }))
    .default([]),
});

export interface DailyLogReminder {
  enabled: boolean;
  time: string | null;
}

/** Whether the daily-log reminder is the board's nightly 21:00 one. */
export function isNightlyReminder(r: DailyLogReminder | null | undefined): boolean {
  return !!r && r.enabled && r.time === VOICE_REMINDER_TIME;
}

function dailyLogOf(data: unknown): DailyLogReminder | null {
  const parsed = reminderSchema.parse(data);
  const r = parsed.reminders.find((x) => x.code === 'daily_log');
  return r ? { enabled: r.enabled, time: r.time ?? null } : null;
}

export function useDailyLogReminder() {
  return useQuery({
    queryKey: KEY,
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(PATH);
      return dailyLogOf(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}

/** On → `daily_log` enabled at 21:00; off → disabled (its time is kept). Optimistic. */
export function useSetNightlyReminder() {
  const queryClient = useQueryClient();
  return useMutation<DailyLogReminder | null, unknown, boolean, { previous?: DailyLogReminder | null }>({
    mutationFn: async (on) => {
      const body = { reminders: { daily_log: on ? { enabled: true, time: VOICE_REMINDER_TIME } : { enabled: false } } };
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(PATH, body);
      return dailyLogOf(data.data);
    },
    onMutate: async (on) => {
      await queryClient.cancelQueries({ queryKey: KEY });
      const previous = queryClient.getQueryData<DailyLogReminder | null>(KEY);
      const next: DailyLogReminder = { enabled: on, time: on ? VOICE_REMINDER_TIME : (previous?.time ?? null) };
      queryClient.setQueryData<DailyLogReminder | null>(KEY, next);
      return { previous };
    },
    onError: (_e, _on, ctx) => {
      if (ctx && ctx.previous !== undefined) queryClient.setQueryData(KEY, ctx.previous);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['cycle-settings'] });
      void queryClient.invalidateQueries({ queryKey: ['notification-settings'] });
    },
  });
}
