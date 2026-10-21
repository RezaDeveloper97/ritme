'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  healthLogKeys,
  logKeys,
  saveLogDay,
  type LogDay,
  type LogDayChanges,
  type LogDayValues,
} from '@/entities/health-log';
import { wellbeingKeys } from '@/entities/wellbeing';
import { invalidateCycleDependents } from '@/shared/api';
import { track } from '@/shared/analytics';

export interface SaveLogDayInput {
  date: string;
  /** Only the params that changed (`diffDay`) — one PUT per save. */
  changes: LogDayChanges;
  /** The whole edited day, painted into the cache before the server answers. */
  draft: LogDayValues;
  /** `category.param` keys confirmed from voice-log suggestions (B-N3-05) — stored with `source: voice`. */
  voiceParams?: string[];
}

/**
 * Saves the log sheet: one `PUT /logs/days/{date}`. Optimistic — the day's cache shows the draft at once
 * and rolls back on failure. The backend syncs v2 entries into the legacy `daily_health_logs` row, which
 * home, calendar, cycle engine, fertility and messages still read, so all of those are refreshed after.
 */
export function useSaveLogDay() {
  const queryClient = useQueryClient();
  return useMutation<LogDay, unknown, SaveLogDayInput, { previous: LogDay | undefined }>({
    mutationFn: ({ date, changes, voiceParams }) => saveLogDay(date, changes, voiceParams),
    onMutate: async ({ date, draft }) => {
      const key = logKeys.day(date);
      await queryClient.cancelQueries({ queryKey: key });
      const previous = queryClient.getQueryData<LogDay>(key);
      queryClient.setQueryData<LogDay>(key, { date, categories: draft });
      return { previous };
    },
    onError: (_error, { date }, context) => {
      queryClient.setQueryData(logKeys.day(date), context?.previous);
    },
    onSuccess: (day, { date }) => {
      queryClient.setQueryData(logKeys.day(date), day);
      track({ name: 'log_saved', props: { source: 'full' } });
      void queryClient.invalidateQueries({ queryKey: logKeys.ranges() });
      void queryClient.invalidateQueries({ queryKey: healthLogKeys.all });
      invalidateCycleDependents(queryClient);
      void queryClient.invalidateQueries({ queryKey: wellbeingKeys.all });
    },
  });
}
