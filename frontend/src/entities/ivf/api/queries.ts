'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { IvfDoseDay, IvfDoseInput, IvfHome } from '../model/types';
import { ivfKeys } from './keys';
import { ivfHomeSchema, ivfMedsTodaySchema, ivfStagesSchema } from './schema';

/*
 * `/api/v1/ivf*` (CB-IVF-01), Go only. Health data (CLAUDE.md §11): never log
 * a payload or a response.
 */

export async function fetchIvfHome(): Promise<IvfHome> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/ivf');
  return ivfHomeSchema.parse(data.data);
}

/** GET /ivf — the IVF «امروز» (open cycle + timeline, today's doses, next appointment, companion). */
export function useIvfHome() {
  return useQuery({
    queryKey: ivfKeys.home(),
    queryFn: fetchIvfHome,
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

/** GET /catalog/ivf_stages?audience=ttc — admin-editable stage titles + hints (request locale). */
export function useIvfStages(locale: string) {
  return useQuery({
    queryKey: ivfKeys.stages(locale),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/catalog/ivf_stages', {
        params: { audience: 'ttc' },
      });
      return ivfStagesSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 10 * 60_000,
    retry: 1,
  });
}

/** Put a fresh dose day (a log/undo answer) into the cached home, so the row flips at once. */
function useApplyToday() {
  const queryClient = useQueryClient();
  return (today: IvfDoseDay) => {
    queryClient.setQueryData<IvfHome>(ivfKeys.home(), (home) =>
      home && home.today.date === today.date ? { ...home, today } : home,
    );
    void queryClient.invalidateQueries({ queryKey: ivfKeys.all });
  };
}

/** POST /ivf/meds/{id}/doses — marks the dose taken (a care intake; `site` optional). */
export function useLogIvfDose() {
  const apply = useApplyToday();
  return useMutation<IvfDoseDay, unknown, IvfDoseInput>({
    mutationFn: async ({ medId, date, slot, site }) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(`/ivf/meds/${medId}/doses`, {
        date,
        slot,
        site: site ?? null,
      });
      return ivfMedsTodaySchema.parse(data.data);
    },
    onSuccess: apply,
  });
}

/** DELETE /ivf/meds/{id}/doses — undoes a logged dose. */
export function useUnlogIvfDose() {
  const apply = useApplyToday();
  return useMutation<IvfDoseDay, unknown, IvfDoseInput>({
    mutationFn: async ({ medId, date, slot }) => {
      // The shared client sends no DELETE body; the API reads the same keys from the query (Laravel input()).
      const { data } = await apiClient.delete<ApiEnvelope<unknown>>(`/ivf/meds/${medId}/doses`, {
        params: { date, slot },
      });
      return ivfMedsTodaySchema.parse(data.data);
    },
    onSuccess: apply,
  });
}

/**
 * PUT /ivf/cycles/current {notify_companion} — «همدمت هم در جریان باشد».
 * Optimistic: the switch moves at once and rolls back on an error.
 */
export function useSetIvfCompanionNotify() {
  const queryClient = useQueryClient();
  return useMutation<IvfHome, unknown, boolean, { previous?: IvfHome }>({
    mutationFn: async (notify) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>('/ivf/cycles/current', { notify_companion: notify });
      return ivfHomeSchema.parse(data.data);
    },
    onMutate: async (notify) => {
      await queryClient.cancelQueries({ queryKey: ivfKeys.home() });
      const previous = queryClient.getQueryData<IvfHome>(ivfKeys.home());
      if (previous) {
        queryClient.setQueryData<IvfHome>(ivfKeys.home(), {
          ...previous,
          companion: { ...previous.companion, notify },
          cycle: previous.cycle ? { ...previous.cycle, notifyCompanion: notify } : previous.cycle,
        });
      }
      return { previous };
    },
    onError: (_error, _notify, context) => {
      if (context?.previous) queryClient.setQueryData(ivfKeys.home(), context.previous);
    },
    onSuccess: (home) => queryClient.setQueryData(ivfKeys.home(), home),
  });
}

/**
 * POST /ivf/cycles {} — opens a cycle at «آماده‌سازی» today (and switches
 * «IVF/IUI» on server-side; callers refresh the life-stage read they own).
 */
export function useStartIvfCycle() {
  const queryClient = useQueryClient();
  return useMutation<IvfHome, unknown, void>({
    mutationFn: async () => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/ivf/cycles', {});
      return ivfHomeSchema.parse(data.data);
    },
    onSuccess: (home) => {
      queryClient.setQueryData(ivfKeys.home(), home);
      void queryClient.invalidateQueries({ queryKey: ivfKeys.all });
    },
  });
}
