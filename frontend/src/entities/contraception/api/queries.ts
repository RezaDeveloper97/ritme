'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { ContraceptionOverview, MethodPayload } from '../model/types';
import { contraceptionKeys } from './keys';
import { contraceptionOverviewSchema } from './schema';

/*
 * `/api/v1/contraception` (CB-CONTRA-01, Go only). Every write responds with
 * the whole screen, which replaces the cached overview. Health data (§11):
 * never log a payload or a response.
 */

const PATH = '/contraception';

async function parse(promise: Promise<{ data: ApiEnvelope<unknown> }>): Promise<ContraceptionOverview> {
  const { data } = await promise;
  return contraceptionOverviewSchema.parse(data.data);
}

/** GET /contraception — switch, method, today's pack and the method's care reminders. */
export function fetchContraception(): Promise<ContraceptionOverview> {
  return parse(apiClient.get<ApiEnvelope<unknown>>(PATH));
}

export function useContraception() {
  return useQuery({
    queryKey: contraceptionKeys.overview(),
    queryFn: fetchContraception,
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}

function useOverviewWrite<V>(write: (vars: V) => Promise<ContraceptionOverview>) {
  const queryClient = useQueryClient();
  return useMutation<ContraceptionOverview, unknown, V>({
    mutationFn: write,
    onSuccess: (overview) => {
      queryClient.setQueryData(contraceptionKeys.overview(), overview);
    },
  });
}

/**
 * PUT /contraception/method — full replace; also switches «track
 * contraception» on and sets the one pill reminder. Callers refresh the
 * life-stage cache (entities/user) themselves — sibling slices don't import
 * each other.
 */
export function useSaveContraceptionMethod() {
  return useOverviewWrite((payload: MethodPayload) =>
    parse(apiClient.put<ApiEnvelope<unknown>>(`${PATH}/method`, payload)),
  );
}

/** DELETE /contraception/method — stop tracking: method + its reminders go, the pill log stays. */
export function useStopContraception() {
  return useOverviewWrite(() => parse(apiClient.delete<ApiEnvelope<unknown>>(`${PATH}/method`)));
}

/** POST /contraception/pills — log a pill (default: today, taken). */
export function useLogPill() {
  return useOverviewWrite((body: { date?: string; status?: 'taken' | 'missed' } = {}) =>
    parse(apiClient.post<ApiEnvelope<unknown>>(`${PATH}/pills`, body)),
  );
}

/** DELETE /contraception/pills/{date} — undo a pill log. */
export function useUndoPill() {
  return useOverviewWrite((date: string) =>
    parse(apiClient.delete<ApiEnvelope<unknown>>(`${PATH}/pills/${encodeURIComponent(date)}`)),
  );
}
