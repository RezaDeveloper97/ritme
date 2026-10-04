'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { applyConsent, type Consent, type ConsentCode, consentChangeBody, consentsSchema } from '../model/consents';

/** Query-key factory (CLAUDE.md §8). */
export const consentKeys = {
  all: ['consents'] as const,
};

const PATH = '/profile/consents';

/** GET /profile/consents. */
export function useConsents() {
  return useQuery({
    queryKey: consentKeys.all,
    queryFn: async (): Promise<Consent[]> => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(PATH);
      return consentsSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: 1,
  });
}

interface ConsentChange {
  code: ConsentCode;
  granted: boolean;
  /** Version of the text shown (from GET). */
  version: number;
}

/** PUT /profile/consents for one switch; optimistic, rolled back on failure. */
export function useUpdateConsent() {
  const queryClient = useQueryClient();
  const key = consentKeys.all;
  return useMutation<Consent[], unknown, ConsentChange, { previous?: Consent[] }>({
    mutationKey: [...key, 'update'],
    mutationFn: async ({ code, granted, version }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(PATH, consentChangeBody(code, granted, version));
      return consentsSchema.parse(data.data);
    },
    onMutate: async ({ code, granted }) => {
      await queryClient.cancelQueries({ queryKey: key });
      const previous = queryClient.getQueryData<Consent[]>(key);
      if (previous) queryClient.setQueryData(key, applyConsent(previous, code, granted, new Date().toISOString()));
      return { previous };
    },
    onError: (_e, _v, context) => {
      if (context?.previous) queryClient.setQueryData(key, context.previous);
    },
    onSuccess: (saved) => {
      if (queryClient.isMutating({ mutationKey: [...key, 'update'] }) <= 1) queryClient.setQueryData(key, saved);
    },
  });
}
