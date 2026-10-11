'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { PolicyPayload } from '../model/types';
import { insuranceKeys } from './keys';
import { claimsSchema, insurersSchema, policyStateSchema } from './schema';

/*
 * `/api/v1/insurance` (CB-INS-01, Go only). Self-tracking: nothing here is sent to an insurer. Health and financial
 * data (§11): never log a payload or a response.
 */

/** GET /insurance/policy — her policy with coverage usage, or `policy: null` (setup form). */
export function useInsurancePolicy() {
  return useQuery({
    queryKey: insuranceKeys.policy(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/insurance/policy');
      return policyStateSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}

/** GET /insurance/insurers — the admin catalog (names only). */
export function useInsurers(enabled = true) {
  return useQuery({
    queryKey: insuranceKeys.insurers(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/insurance/insurers');
      return insurersSchema.parse(data.data);
    },
    enabled: enabled && isAuthenticated(),
    staleTime: 10 * 60_000,
    retry: 1,
  });
}

/** GET /insurance/claims — the policy-period claims; the home shows the latest few. */
export function useRecentClaims(enabled = true) {
  return useQuery({
    queryKey: insuranceKeys.recentClaims(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/insurance/claims');
      return claimsSchema.parse(data.data);
    },
    enabled: enabled && isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}

/** PUT /insurance/policy — create or replace; the response carries the whole policy state. */
export function useSaveInsurancePolicy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (payload: PolicyPayload) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>('/insurance/policy', payload);
      return policyStateSchema.parse(data.data);
    },
    onSuccess: (state) => {
      queryClient.setQueryData(insuranceKeys.policy(), state);
      void queryClient.invalidateQueries({ queryKey: insuranceKeys.recentClaims() });
    },
  });
}
