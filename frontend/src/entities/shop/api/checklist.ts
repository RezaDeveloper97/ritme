'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient, getApiErrorStatus } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import {
  type ChecklistGroupKey,
  type ChecklistState,
  shopChecklistItemEnvelopeSchema,
  shopChecklistSchema,
  supplyReminderSchema,
} from './checklist-schema';
import { shopKeys } from './keys';

/**
 * GET /shop/checklist[?for_user_id=] — the layette list with her state. `forUserId` = a companion reading her list
 * (`read_only: true`). 404 = teen-mode account (no shop); 403 = no live pregnancy grant.
 */
export const useShopChecklist = (forUserId: number | null = null) =>
  useQuery({
    queryKey: shopKeys.checklist(forUserId),
    queryFn: async () =>
      shopChecklistSchema.parse(
        (await apiClient.get<ApiEnvelope<unknown>>('/shop/checklist', { params: forUserId ? { for_user_id: forUserId } : undefined }))
          .data.data,
      ),
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: (count, error) => ![403, 404].includes(getApiErrorStatus(error) ?? 0) && count < 2,
  });

function useInvalidateChecklist() {
  const qc = useQueryClient();
  return () => qc.invalidateQueries({ queryKey: shopKeys.checklistAll() });
}

/** PUT /shop/checklist/items/:id — her state (+ quantity) on a template item. */
export function useSetChecklistItemState() {
  const invalidate = useInvalidateChecklist();
  return useMutation({
    mutationFn: async (v: { layetteItemId: number; state: ChecklistState; quantity?: number | null }) => {
      await apiClient.put(`/shop/checklist/items/${v.layetteItemId}`, { state: v.state, quantity: v.quantity ?? null });
    },
    onSuccess: invalidate,
  });
}

export interface CustomItemInput {
  title?: string;
  group?: ChecklistGroupKey;
  state?: ChecklistState;
  quantity?: number | null;
}

/** POST /shop/checklist/custom — her own row. */
export function useAddCustomItem() {
  const invalidate = useInvalidateChecklist();
  return useMutation({
    mutationFn: async (input: CustomItemInput) =>
      shopChecklistItemEnvelopeSchema.parse((await apiClient.post<ApiEnvelope<unknown>>('/shop/checklist/custom', input)).data.data).item,
    onSuccess: invalidate,
  });
}

/** PATCH /shop/checklist/custom/:id. */
export function useUpdateCustomItem() {
  const invalidate = useInvalidateChecklist();
  return useMutation({
    mutationFn: async (v: { id: number } & CustomItemInput) => {
      const { id, ...body } = v;
      await apiClient.patch(`/shop/checklist/custom/${id}`, body);
    },
    onSuccess: invalidate,
  });
}

/** DELETE /shop/checklist/custom/:id (template rows are only ever marked `not_needed`). */
export function useDeleteCustomItem() {
  const invalidate = useInvalidateChecklist();
  return useMutation({
    mutationFn: async (id: number) => {
      await apiClient.delete(`/shop/checklist/custom/${id}`);
    },
    onSuccess: invalidate,
  });
}

/** GET /shop/supply-reminder — off by default; the read re-syncs the care reminder to the current prediction. */
export const useSupplyReminder = () =>
  useQuery({
    queryKey: shopKeys.supplyReminder(),
    queryFn: async () => supplyReminderSchema.parse((await apiClient.get<ApiEnvelope<unknown>>('/shop/supply-reminder')).data.data),
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: (count, error) => getApiErrorStatus(error) !== 404 && count < 2,
  });

/** PUT /shop/supply-reminder. */
export function useSaveSupplyReminder() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (v: { enabled: boolean; leadDays?: number }) =>
      supplyReminderSchema.parse(
        (await apiClient.put<ApiEnvelope<unknown>>('/shop/supply-reminder', { enabled: v.enabled, lead_days: v.leadDays })).data.data,
      ),
    onSuccess: (data) => qc.setQueryData(shopKeys.supplyReminder(), data),
  });
}
