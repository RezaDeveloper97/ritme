'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';

import { type InfoPage, type InfoPageGroup, infoPageSchema } from '../model/info-page';

/** Query-key factory (CLAUDE.md §8); keyed by locale because the copy comes localized. */
export const infoPageKeys = {
  all: ['info-page'] as const,
  page: (group: InfoPageGroup, locale: string) => [...infoPageKeys.all, group, locale] as const,
};

/** GET /info-pages/{group} — public, admin-edited, so cached for an hour. */
export function useInfoPage(group: InfoPageGroup, locale: string) {
  return useQuery({
    queryKey: infoPageKeys.page(group, locale),
    queryFn: async (): Promise<InfoPage> => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/info-pages/${group}`);
      return infoPageSchema.parse(data.data);
    },
    staleTime: 60 * 60_000,
    retry: 1,
  });
}
