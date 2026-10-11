'use client';

import { type QueryClient, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { partnerLabKeys } from './keys';
import { type LabWindow, labRequestSchema, labRequestsSchema, labServicesSchema } from './schema';

const importedSchema = z.object({ lab_id: z.number() });

/** GET /partners/lab-services — the active catalog; an empty list means no «پیشنهاد» card and no form. */
export const useLabServices = () =>
  useQuery({
    queryKey: partnerLabKeys.services(),
    queryFn: async () => labServicesSchema.parse((await apiClient.get<ApiEnvelope<unknown>>('/partners/lab-services')).data.data),
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
  });

/** GET /partners/lab-requests — my requests, newest first (no personal details in the list). */
export const useLabRequests = () =>
  useQuery({
    queryKey: partnerLabKeys.requests(),
    queryFn: async () => labRequestsSchema.parse((await apiClient.get<ApiEnvelope<unknown>>('/partners/lab-requests')).data.data),
    enabled: isAuthenticated(),
  });

/** GET /partners/lab-requests/:id — one request with its details (until the retention purge). */
export const useLabRequest = (id: number) =>
  useQuery({
    queryKey: partnerLabKeys.request(id),
    queryFn: async () => labRequestSchema.parse((await apiClient.get<ApiEnvelope<unknown>>(`/partners/lab-requests/${id}`)).data.data),
    enabled: isAuthenticated() && Number.isFinite(id),
    refetchInterval: 60_000,
  });

export interface LabRequestInput {
  serviceId: number;
  fullName: string;
  phone: string;
  city: string;
  address: string;
  preferredDate: string;
  window: LabWindow;
  note: string;
  consentShare: boolean;
  consentImport: boolean;
}

function invalidate(queryClient: QueryClient, id?: number): void {
  void queryClient.invalidateQueries({ queryKey: partnerLabKeys.requests() });
  if (id !== undefined) void queryClient.invalidateQueries({ queryKey: partnerLabKeys.request(id) });
}

/** POST /partners/lab-requests → 201: the booking form (personal fields go to the API only, never to storage). */
export function useCreateLabRequest() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: LabRequestInput) =>
      labRequestSchema.parse(
        (
          await apiClient.post<ApiEnvelope<unknown>>('/partners/lab-requests', {
            service_id: input.serviceId,
            full_name: input.fullName,
            phone: input.phone,
            city: input.city,
            address: input.address,
            preferred_date: input.preferredDate,
            preferred_window: input.window,
            note: input.note.trim() === '' ? null : input.note,
            consent_share: input.consentShare,
            consent_import: input.consentImport,
          })
        ).data.data,
      ),
    onSuccess: () => invalidate(queryClient),
  });
}

/** POST /partners/lab-requests/:id/cancel (requested or accepted only). */
export function useCancelLabRequest(id: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await apiClient.post(`/partners/lab-requests/${id}/cancel`);
    },
    onSuccess: () => invalidate(queryClient, id),
  });
}

/** POST /partners/lab-requests/:id/import-result {consent} → {lab_id}: copies the result into her labs. */
export function useImportLabResult(id: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (consent: boolean) => {
      const res = await apiClient.post<ApiEnvelope<unknown>>(`/partners/lab-requests/${id}/import-result`, { consent });
      return importedSchema.parse(res.data.data).lab_id;
    },
    onSuccess: () => {
      invalidate(queryClient, id);
      void queryClient.invalidateQueries({ queryKey: ['labs'] });
    },
  });
}
