'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { ClaimEventPayload, ClaimPayload, PolicyPayload } from '../model/types';
import { insuranceKeys } from './keys';
import { centresSchema, claimDetailSchema, claimsSchema, insurersSchema, policyHistorySchema, policyStateSchema } from './schema';

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

/** GET /insurance/policies — her policy history (switcher), newest period first. */
export function useInsurancePolicies(enabled = true) {
  return useQuery({
    queryKey: insuranceKeys.policies(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/insurance/policies');
      return policyHistorySchema.parse(data.data);
    },
    enabled: enabled && isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}

/** GET /insurance/policies/{id} — one policy of her history (null id = disabled; the current one reads /policy). */
export function useInsurancePolicyById(id: number | null) {
  return useQuery({
    queryKey: insuranceKeys.policyById(id ?? 0),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/insurance/policies/${id}`);
      return policyStateSchema.parse(data.data);
    },
    enabled: isAuthenticated() && id !== null && id > 0,
    staleTime: 60_000,
    retry: 1,
  });
}

/** Any policy write refreshes the current policy, the history and every per-policy view. */
function invalidatePolicies(queryClient: ReturnType<typeof useQueryClient>) {
  void queryClient.invalidateQueries({ queryKey: insuranceKeys.policy() });
  void queryClient.invalidateQueries({ queryKey: insuranceKeys.policies() });
  void queryClient.invalidateQueries({ queryKey: [...insuranceKeys.all, 'claims'] });
}

/** POST /insurance/policies/{id}/renew — the next policy (members and coverages copied); answers its state. */
export function useRenewInsurancePolicy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(`/insurance/policies/${id}/renew`, {});
      return policyStateSchema.parse(data.data);
    },
    onSuccess: (state) => {
      if (state.policy) queryClient.setQueryData(insuranceKeys.policyById(state.policy.id), state);
      invalidatePolicies(queryClient);
    },
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

/** GET /insurance/claims — the claims of a policy (default the current one); the home shows the latest few. */
export function useRecentClaims(enabled = true, policyId: number | null = null) {
  return useQuery({
    queryKey: insuranceKeys.recentClaims(policyId),
    queryFn: async () => {
      const params = policyId ? { policy_id: String(policyId) } : undefined;
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/insurance/claims', { params });
      return claimsSchema.parse(data.data);
    },
    enabled: enabled && isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}

/** GET /insurance/centres — contracted centres, filtered server-side by type and name/service text. */
export function useCentres(type: string, q: string) {
  return useQuery({
    queryKey: insuranceKeys.centres(type, q),
    queryFn: async () => {
      const params: Record<string, string> = {};
      if (type) params.type = type;
      if (q) params.q = q;
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/insurance/centres', { params });
      return centresSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    placeholderData: (prev) => prev,
    retry: 1,
  });
}

/**
 * PUT /insurance/policy (the current one; creates her first) or PUT /insurance/policies/{id} (a given policy of her
 * history, e.g. a renewal not started yet) — the response carries the whole policy state.
 */
export function useSaveInsurancePolicy(policyId: number | null = null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (payload: PolicyPayload) => {
      const url = policyId ? `/insurance/policies/${policyId}` : '/insurance/policy';
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(url, payload);
      return policyStateSchema.parse(data.data);
    },
    onSuccess: (state) => {
      if (policyId) queryClient.setQueryData(insuranceKeys.policyById(policyId), state);
      else queryClient.setQueryData(insuranceKeys.policy(), state);
      invalidatePolicies(queryClient);
    },
  });
}

/** GET /insurance/claims/{id} — payable breakdown, timeline and attached record documents. */
export function useClaim(id: number) {
  return useQuery({
    queryKey: insuranceKeys.claim(id),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/insurance/claims/${id}`);
      return claimDetailSchema.parse(data.data);
    },
    enabled: isAuthenticated() && Number.isFinite(id) && id > 0,
    staleTime: 30_000,
    retry: 1,
  });
}

/** A claim change refreshes the list, the home summary and the policy's usage / needs-action count. */
function useClaimWrite<V>(call: (v: V) => Promise<unknown>, id?: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (v: V) => claimDetailSchema.parse(await call(v)),
    onSuccess: (claim) => {
      queryClient.setQueryData(insuranceKeys.claim(id ?? claim.id), claim);
      invalidatePolicies(queryClient);
    },
  });
}

/** POST /insurance/claims — self-tracked, nothing is sent to the insurer. */
export function useCreateClaim() {
  return useClaimWrite<ClaimPayload>(async (payload) => (await apiClient.post<ApiEnvelope<unknown>>('/insurance/claims', payload)).data.data);
}

/** POST /insurance/claims/{id}/events — a step she maintains; the status follows the latest one. */
export function useAddClaimEvent(id: number) {
  return useClaimWrite<ClaimEventPayload>(
    async (payload) => (await apiClient.post<ApiEnvelope<unknown>>(`/insurance/claims/${id}/events`, payload)).data.data,
    id,
  );
}

/** POST /insurance/claims/{id}/documents — one of her own record documents. */
export function useAttachClaimDocument(id: number) {
  return useClaimWrite<{ documentId: number; label?: string | null }>(
    async (v) =>
      (await apiClient.post<ApiEnvelope<unknown>>(`/insurance/claims/${id}/documents`, { document_id: v.documentId, label: v.label ?? null })).data.data,
    id,
  );
}

/** DELETE /insurance/claims/{id}/documents/{documentId}. */
export function useDetachClaimDocument(id: number) {
  return useClaimWrite<number>(async (documentId) => (await apiClient.delete<ApiEnvelope<unknown>>(`/insurance/claims/${id}/documents/${documentId}`)).data.data, id);
}

/** DELETE /insurance/claims/{id}. */
export function useDeleteClaim(id: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await apiClient.delete(`/insurance/claims/${id}`);
    },
    onSuccess: () => {
      queryClient.removeQueries({ queryKey: insuranceKeys.claim(id) });
      invalidatePolicies(queryClient);
    },
  });
}
