'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { PelvicAlerts, PelvicDiary, PelvicDiaryPatch, PelvicOverview, PelvicSessionInput } from '../model/types';
import { pelvicKeys } from './keys';
import { pelvicAlertsSchema, pelvicDiarySchema, pelvicOverviewSchema } from './schema';

/*
 * `/api/v1/pelvic*` (CB-PELV-01), Go only. Leaks, night waking and UTI symptoms are health data (§11): never log a
 * payload, a response or an error body, never put them in a URL (the diary date is the only path segment).
 */

export async function fetchPelvic(): Promise<PelvicOverview> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/pelvic');
  return pelvicOverviewSchema.parse(data.data);
}

/** GET /pelvic — the program (null until started) and today's bladder diary. */
export function usePelvic() {
  return useQuery({
    queryKey: pelvicKeys.overview(),
    queryFn: fetchPelvic,
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

/** Program and session writes answer the full overview. */
function useOverviewWrite<V>(send: (vars: V) => Promise<ApiEnvelope<unknown>>) {
  const queryClient = useQueryClient();
  return useMutation<PelvicOverview, unknown, V>({
    mutationFn: async (vars) => pelvicOverviewSchema.parse((await send(vars)).data),
    onSuccess: (overview) => queryClient.setQueryData(pelvicKeys.overview(), overview),
  });
}

/** POST /pelvic/program — starts (or restarts) the 8 weeks today. */
export function useStartPelvicProgram() {
  return useOverviewWrite<void>(async () => (await apiClient.post<ApiEnvelope<unknown>>('/pelvic/program', {})).data);
}

/** POST /pelvic/sessions — adds a guided session to today. */
export function useSavePelvicSession() {
  return useOverviewWrite<PelvicSessionInput>(
    async ({ setsCompleted, durationSec }) =>
      (
        await apiClient.post<ApiEnvelope<unknown>>('/pelvic/sessions', {
          sets_completed: setsCompleted,
          duration_sec: durationSec,
        })
      ).data,
  );
}

/** PUT /pelvic/diary/{date} — only the sent keys change; the saved day replaces the overview's `diary_today`. */
export function useSavePelvicDiary() {
  const queryClient = useQueryClient();
  return useMutation<PelvicDiary, unknown, { date: string; patch: PelvicDiaryPatch }>({
    mutationFn: async ({ date, patch }) =>
      pelvicDiarySchema.parse(
        (await apiClient.put<ApiEnvelope<unknown>>(`/pelvic/diary/${date}`, patch)).data.data,
      ),
    onSuccess: (diary) =>
      queryClient.setQueryData<PelvicOverview>(pelvicKeys.overview(), (old) =>
        old && old.diaryToday.date === diary.date ? { ...old, diaryToday: diary } : old,
      ),
  });
}

/** GET /catalog/pelvic_alerts — admin copy of the UTI warning and the suitability note (content, not health data). */
export function usePelvicAlerts(locale: string) {
  return useQuery<PelvicAlerts>({
    queryKey: pelvicKeys.alerts(locale),
    queryFn: async () =>
      pelvicAlertsSchema.parse((await apiClient.get<ApiEnvelope<unknown>>('/catalog/pelvic_alerts')).data.data),
    enabled: isAuthenticated(),
    staleTime: 10 * 60_000,
    retry: 1,
  });
}
