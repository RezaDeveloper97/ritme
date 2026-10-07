'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { ServicesHub } from '../model/types';
import { serviceHubKeys } from './keys';
import { servicesHubSchema } from './schema';

/** GET /services — the «خدمات» hub (B-N7-01). */
export async function fetchServicesHub(): Promise<ServicesHub> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/services');
  return servicesHubSchema.parse(data.data);
}

export function useServicesHub() {
  return useQuery({
    queryKey: serviceHubKeys.hub(),
    queryFn: fetchServicesHub,
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}
