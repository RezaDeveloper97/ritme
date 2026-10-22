'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { type ConsentEvent, consentHistorySchema } from '../model/history';
import { consentKeys } from './consents';

/** GET /consents/history — the user's own consent ledger, newest first (T-M10-23). */
export function useConsentHistory() {
  return useQuery({
    queryKey: [...consentKeys.all, 'history'] as const,
    queryFn: async (): Promise<ConsentEvent[]> => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/consents/history');
      return consentHistorySchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}
