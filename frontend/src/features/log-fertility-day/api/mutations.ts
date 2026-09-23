'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { cycleKeys } from '@/entities/cycle';
import {
  type FertilityDay,
  type FertilityDayInput,
  fertilityDaySchema,
  fertilityKeys,
} from '@/entities/fertility';
import { healthLogKeys } from '@/entities/health-log';
import { messageKeys } from '@/entities/message';

import { toFertilityDayBody } from '../model/body';

/*
 * «ذخیره» on the TTC day log (`/fertility/log`, T-M5-06).
 *
 * One PUT writes both `fertility_logs` (LH, mucus, BBT time) and the shared
 * `daily_health_logs` columns (BBT, intercourse, symptoms, note), and the
 * server runs the health-log side effects (period-start check, recalculation
 * mark). So a save refreshes:
 *   - every fertility read (tiles, day, BBT chart, insights);
 *   - the cycle / home reads (`cycleKeys`, the daily message) — the chance
 *     level and the home ring come from the cycle view;
 *   - the health-log caches, which read the same columns on the legacy log page.
 *
 * Privacy (§11): the day is health data — API only, never logged.
 */

export interface SaveFertilityDayVars {
  /** `Y-m-d` of the day being logged (not in the future — the API 422s). */
  date: string;
  input: FertilityDayInput;
}

export function useSaveFertilityDay() {
  const queryClient = useQueryClient();
  return useMutation<FertilityDay, unknown, SaveFertilityDayVars>({
    mutationFn: async ({ date, input }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(
        `/fertility/days/${date}`,
        toFertilityDayBody(input),
      );
      return fertilityDaySchema.parse(data.data);
    },
    onSuccess: (day, { date }) => {
      // The PUT answers with the merged day — adopt it before the refetch lands.
      queryClient.setQueryData(fertilityKeys.day(date), day);
      void queryClient.invalidateQueries({ queryKey: fertilityKeys.all });
      void queryClient.invalidateQueries({ queryKey: cycleKeys.all });
      void queryClient.invalidateQueries({ queryKey: messageKeys.all });
      void queryClient.invalidateQueries({ queryKey: healthLogKeys.all });
    },
  });
}
