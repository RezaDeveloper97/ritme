'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, ApiError, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { deleteChildPhoto } from '../model/photo';
import type { Child, ChildHome, ChildInput, ChildrenList } from '../model/types';
import { childKeys } from './keys';
import { childHomeSchema, childrenListSchema, childSchema } from './schema';

/*
 * `/api/v1/children*` (B-N5-02), Go only. Health data (CLAUDE.md §11): never
 * log a payload or a response. The photo never goes to the server
 * (`model/photo.ts`).
 */

export async function fetchChildren(): Promise<ChildrenList> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/children');
  return childrenListSchema.parse(data.data);
}

/** GET /children — own children (youngest first), then those a spouse-owner shares (read-only). */
export function useChildren(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: childKeys.list(),
    queryFn: fetchChildren,
    enabled: isAuthenticated() && (options.enabled ?? true),
    staleTime: 60_000,
    retry: 1,
  });
}

export async function fetchChild(id: number): Promise<ChildHome> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/children/${id}`);
  return childHomeSchema.parse(data.data);
}

/** GET /children/{id} — the child home payload. 404 `child_not_found` is not retried. */
export function useChild(id: number | null) {
  return useQuery({
    queryKey: childKeys.detail(id ?? 0),
    queryFn: () => fetchChild(id as number),
    enabled: isAuthenticated() && id != null && id > 0,
    staleTime: 30_000,
    retry: (count, error) => !(error instanceof ApiError && error.response?.status === 404) && count < 1,
  });
}

/** Snake-case body; every key is sent so `null` clears it on an update. */
export function toChildBody(input: ChildInput): Record<string, unknown> {
  return {
    name: input.name.trim(),
    birth_date: input.birthDate,
    sex: input.sex,
    delivery_type: input.deliveryType,
    birth_weight_kg: input.birthWeightKg,
    birth_length_cm: input.birthLengthCm,
    birth_head_cm: input.birthHeadCm,
  };
}

/** POST /children (≤ 10 per owner, 422 `children_limit`). */
export function useCreateChild() {
  const queryClient = useQueryClient();
  return useMutation<Child, unknown, ChildInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/children', toChildBody(input));
      return childSchema.parse(data.data);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: childKeys.all });
    },
  });
}

/** PUT /children/{id} — owner only (403 `child_read_only` for a spouse). */
export function useUpdateChild(id: number) {
  const queryClient = useQueryClient();
  return useMutation<Child, unknown, ChildInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/children/${id}`, toChildBody(input));
      return childSchema.parse(data.data);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: childKeys.all });
    },
  });
}

/** DELETE /children/{id} — also drops the on-device photo. */
export function useDeleteChild(id: number) {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, void>({
    mutationFn: async () => {
      await apiClient.delete(`/children/${id}`);
      await deleteChildPhoto(id);
    },
    onSuccess: () => {
      queryClient.removeQueries({ queryKey: childKeys.detail(id) });
      void queryClient.invalidateQueries({ queryKey: childKeys.list() });
    },
  });
}

/** First validation message of `field` on a 422, if any. */
export function childFieldError(error: unknown, field: string): string | undefined {
  if (!(error instanceof ApiError)) return undefined;
  const body = error.response?.data as ApiEnvelope<unknown> | undefined;
  const list = body && typeof body === 'object' ? body.errors?.[field] : undefined;
  return Array.isArray(list) && typeof list[0] === 'string' ? list[0] : undefined;
}
