'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { type PendingConsent, pendingConsentsSchema } from '../model/consents';

/** Query-key factory (CLAUDE.md §8). */
export const pendingConsentKeys = {
  all: ['consents', 'pending'] as const,
};

/** GET /consents/pending — the mandatory consents the signed-in user has not accepted yet (T-M10-23). */
export function usePendingConsents() {
  return useQuery({
    queryKey: pendingConsentKeys.all,
    queryFn: async (): Promise<PendingConsent[]> => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/consents/pending');
      return pendingConsentsSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: Infinity,
    retry: false,
  });
}

/** PUT /consents/{code} for every pending consent, in order, with the version shown. */
export function useAcceptPendingConsents() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, readonly PendingConsent[]>({
    mutationFn: async (pending) => {
      for (const c of pending) {
        await apiClient.put(`/consents/${encodeURIComponent(c.code)}`, { granted: true, version: c.version });
      }
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: pendingConsentKeys.all }),
  });
}
