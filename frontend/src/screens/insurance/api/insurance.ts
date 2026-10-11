'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';

import { type InsurancePageData, insurancePageSchema } from '../model/insurance';

export const insuranceKeys = {
  all: ['insurance-page'] as const,
  page: (locale: string) => [...insuranceKeys.all, locale] as const,
};

/** GET /info-pages/insurance — public admin content, cached for an hour. */
export function useInsurancePage(locale: string) {
  return useQuery({
    queryKey: insuranceKeys.page(locale),
    queryFn: async (): Promise<InsurancePageData> => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/info-pages/insurance');
      return insurancePageSchema.parse(data.data);
    },
    staleTime: 60 * 60_000,
    retry: 1,
  });
}
