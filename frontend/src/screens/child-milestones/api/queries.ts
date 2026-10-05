'use client';

import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { childKeys } from '@/entities/child';
import { type ApiEnvelope, apiClient, getApiErrorCode } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { MilestonesView } from '../model/types';
import { milestonesSchema } from './schema';

/* `/api/v1/children/{id}/milestones` (B-N5-02), Go only. Never log a payload (§11). */

export const milestoneKeys = {
  all: (id: number) => [...childKeys.detail(id), 'milestones'] as const,
  /** `month` null = the band of the child's age. */
  band: (id: number, month: number | null) => [...milestoneKeys.all(id), month ?? 'current'] as const,
};

/** GET /children/{id}/milestones?month= */
export function useMilestones(id: number, month: number | null) {
  return useQuery<MilestonesView>({
    queryKey: milestoneKeys.band(id, month),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/children/${id}/milestones`, {
        params: { month: month ?? undefined },
      });
      return milestonesSchema.parse(data.data);
    },
    enabled: isAuthenticated() && Number.isFinite(id) && id > 0,
    placeholderData: keepPreviousData,
    staleTime: 30_000,
    retry: 1,
  });
}

/** PUT /children/{id}/milestones/{code} {checked}. Owner only (403 `child_read_only`). */
export function useCheckMilestone(id: number) {
  const queryClient = useQueryClient();
  return useMutation<MilestonesView, unknown, { code: string; checked: boolean }>({
    mutationFn: async ({ code, checked }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/children/${id}/milestones/${encodeURIComponent(code)}`, {
        checked,
      });
      return milestonesSchema.parse(data.data);
    },
    onSuccess: () => {
      // The band views and the child home's «x از y» summary.
      void queryClient.invalidateQueries({ queryKey: childKeys.detail(id) });
    },
  });
}

export function isReadOnlyError(error: unknown): boolean {
  return getApiErrorCode(error) === 'child_read_only';
}
