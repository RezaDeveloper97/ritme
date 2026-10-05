'use client';

import { keepPreviousData, useQuery } from '@tanstack/react-query';

import { childKeys } from '@/entities/child';
import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { LearnView } from '../model/types';
import { learnViewSchema } from './schema';

/* `/api/v1/children/{id}/learn` (B-N5-02), Go only — tips for the child's age, by topic. */

export const learnKeys = {
  list: (id: number, topic: string | null) => [...childKeys.detail(id), 'learn', topic ?? 'all'] as const,
};

/** GET /children/{id}/learn?topic= (`null` / «همه» = every topic, with this week's pick). */
export function useChildLearn(id: number, topic: string | null) {
  return useQuery<LearnView>({
    queryKey: learnKeys.list(id, topic),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/children/${id}/learn`, {
        params: { topic: topic ?? undefined },
      });
      return learnViewSchema.parse(data.data);
    },
    enabled: isAuthenticated() && Number.isFinite(id) && id > 0,
    placeholderData: keepPreviousData,
    staleTime: 5 * 60_000,
    retry: 1,
  });
}
