'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type {
  ConditionCatalogGroup,
  ConditionCatalogItem,
  ConditionNudge,
  ConditionProgram,
  ConditionsOverview,
} from '../model/types';
import { conditionKeys } from './keys';
import { conditionCatalogSchema, conditionNudgesSchema, conditionsOverviewSchema } from './schema';

/*
 * `/api/v1/conditions*` (CB-COND-01) and `/api/v1/messages/nudges` (CB-COND-06b), Go only. Enrolment says which
 * conditions she lives with (§11): never log a payload, a response or an error body, never put it in a URL.
 */

export async function fetchConditions(): Promise<ConditionsOverview> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/conditions');
  return conditionsOverviewSchema.parse(data.data);
}

/** GET /conditions — every program in hub order with her enrolment. */
export function useConditions() {
  return useQuery({
    queryKey: conditionKeys.overview(),
    queryFn: fetchConditions,
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

/** Both writes answer the full overview; nudges depend on enrolment, so they are refetched too. */
function useEnrolmentWrite(send: (program: ConditionProgram) => Promise<ApiEnvelope<unknown>>) {
  const queryClient = useQueryClient();
  return useMutation<ConditionsOverview, unknown, ConditionProgram>({
    mutationFn: async (program) => conditionsOverviewSchema.parse((await send(program)).data),
    onSuccess: (overview) => {
      queryClient.setQueryData(conditionKeys.overview(), overview);
      void queryClient.invalidateQueries({ queryKey: conditionKeys.nudges() });
    },
  });
}

/** POST /conditions/enrolments — joins today (joining again keeps the first date). */
export function useEnrolCondition() {
  return useEnrolmentWrite(
    async (program) => (await apiClient.post<ApiEnvelope<unknown>>('/conditions/enrolments', { program })).data,
  );
}

/** DELETE /conditions/enrolments/{program} — leaves; the diary data stays. */
export function useLeaveCondition() {
  return useEnrolmentWrite(
    async (program) => (await apiClient.delete<ApiEnvelope<unknown>>(`/conditions/enrolments/${program}`)).data,
  );
}

/** GET /messages/nudges — this cycle's heavy-pain / heavy-flow suggestions (possibly none). */
export function useConditionNudges() {
  return useQuery<ConditionNudge[]>({
    queryKey: conditionKeys.nudges(),
    queryFn: async () => conditionNudgesSchema.parse((await apiClient.get<ApiEnvelope<unknown>>('/messages/nudges')).data.data),
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: false,
  });
}

/** GET /catalog/<condition group> — admin-editable copy in the request locale (content, not health data). */
export function useConditionCatalog(group: ConditionCatalogGroup, locale: string) {
  return useQuery<ConditionCatalogItem[]>({
    queryKey: conditionKeys.catalog(group, locale),
    queryFn: async () =>
      conditionCatalogSchema.parse((await apiClient.get<ApiEnvelope<unknown>>(`/catalog/${group}`)).data.data),
    enabled: isAuthenticated(),
    staleTime: 10 * 60_000,
    retry: 1,
  });
}
