'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { childKeys } from '@/entities/child';
import { type ApiEnvelope, apiClient, getApiErrorCode } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { VaccineSchedule } from '../model/types';
import { vaccineScheduleSchema } from './schema';

/*
 * `/api/v1/children/{id}/vaccines*` (B-N5-02), Go only. Every write answers with
 * the whole schedule, which replaces the cache; the child's other reads (home
 * card, list chips, reminders) are invalidated. Never log a payload (§11).
 */

export const vaccineKeys = {
  schedule: (id: number) => [...childKeys.detail(id), 'vaccines'] as const,
};

async function fetchSchedule(id: number): Promise<VaccineSchedule> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/children/${id}/vaccines`);
  return vaccineScheduleSchema.parse(data.data);
}

/** GET /children/{id}/vaccines — the national schedule by the birth date with the recorded doses. */
export function useVaccineSchedule(id: number) {
  return useQuery<VaccineSchedule>({
    queryKey: vaccineKeys.schedule(id),
    queryFn: () => fetchSchedule(id),
    enabled: isAuthenticated() && Number.isFinite(id) && id > 0,
    staleTime: 30_000,
    retry: 1,
  });
}

export type VaccineWrite =
  | { kind: 'visit'; code: string; givenOn: string; note: string }
  | { kind: 'dose'; code: string; givenOn: string; note: string }
  | { kind: 'unmark'; code: string };

/** POST …/vaccines/visits/{visit} · PUT …/vaccines/{code} · DELETE …/vaccines/{code}. Owner only. */
export function useVaccineWrite(id: number) {
  const queryClient = useQueryClient();
  return useMutation<VaccineSchedule, unknown, VaccineWrite>({
    mutationFn: async (w) => {
      const base = `/children/${id}/vaccines`;
      const body = w.kind === 'unmark' ? undefined : { given_on: w.givenOn, note: w.note.trim() || null };
      const { data } =
        w.kind === 'visit'
          ? await apiClient.post<ApiEnvelope<unknown>>(`${base}/visits/${encodeURIComponent(w.code)}`, body)
          : w.kind === 'dose'
            ? await apiClient.put<ApiEnvelope<unknown>>(`${base}/${encodeURIComponent(w.code)}`, body)
            : await apiClient.delete<ApiEnvelope<unknown>>(`${base}/${encodeURIComponent(w.code)}`);
      return vaccineScheduleSchema.parse(data.data);
    },
    onSuccess: (schedule) => {
      queryClient.setQueryData(vaccineKeys.schedule(id), schedule);
      void queryClient.invalidateQueries({
        queryKey: childKeys.all,
        predicate: (q) => q.queryKey[3] !== 'vaccines',
      });
    },
  });
}

/** 403 `child_read_only`: a spouse viewing a shared child. */
export function isReadOnlyError(error: unknown): boolean {
  return getApiErrorCode(error) === 'child_read_only';
}
