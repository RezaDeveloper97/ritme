'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { ClaimEventPayload, ClaimPayload, PolicyPayload } from '../model/types';
import { insuranceKeys } from './keys';
import { centresSchema, claimDetailSchema, claimsSchema, insurersSchema, policyStateSchema } from './schema';

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
      void queryClient.invalidateQueries({ queryKey: insuranceKeys.recentClaims() });
      void queryClient.invalidateQueries({ queryKey: insuranceKeys.policy() });
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
      void queryClient.invalidateQueries({ queryKey: insuranceKeys.recentClaims() });
      void queryClient.invalidateQueries({ queryKey: insuranceKeys.policy() });
    },
  });
}
