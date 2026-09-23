'use client';

import { type QueryClient, useMutation, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import {
  type CheckupDetail,
  type CheckupItem,
  type CheckupSettings,
  checkupItemSchema,
  checkupKeys,
} from '@/entities/checkup';

import { type CustomCheckupInput, type CustomCheckupPatch, toCustomCheckupBody } from '../model/body';

/*
 * The user's own plan: custom checkup types (POST/PUT/DELETE
 * /checkups/custom[/{id}]) and the per-type switches (PUT /checkups/{id}/settings
 * — «فعال» and the reminder bell). Every mutation invalidates through
 * `checkupKeys`. Privacy (§11): titles and notes go to the API only.
 */

function invalidate(queryClient: QueryClient, id?: number): void {
  void queryClient.invalidateQueries({ queryKey: checkupKeys.listAll() });
  void queryClient.invalidateQueries({ queryKey: checkupKeys.home() });
  if (id !== undefined) void queryClient.invalidateQueries({ queryKey: checkupKeys.detail(id) });
}

/** The response is the (recomputed) item, bare or as `{item}`; unreadable → null. */
function parseItem(raw: unknown): CheckupItem | null {
  const candidate =
    typeof raw === 'object' && raw !== null && 'item' in raw ? (raw as { item: unknown }).item : raw;
  const parsed = checkupItemSchema.safeParse(candidate);
  return parsed.success ? parsed.data : null;
}

/** POST /checkups/custom → 201. */
export function useCreateCustomCheckup() {
  const queryClient = useQueryClient();
  return useMutation<CheckupItem | null, unknown, CustomCheckupInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(
        '/checkups/custom',
        toCustomCheckupBody(input),
      );
      return parseItem(data.data);
    },
    onSuccess: () => invalidate(queryClient),
  });
}

export interface UpdateCustomCheckupVars {
  id: number;
  patch: CustomCheckupPatch;
}

/** PUT /checkups/custom/{id} — partial. */
export function useUpdateCustomCheckup() {
  const queryClient = useQueryClient();
  return useMutation<CheckupItem | null, unknown, UpdateCustomCheckupVars>({
    mutationFn: async ({ id, patch }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(
        `/checkups/custom/${id}`,
        toCustomCheckupBody(patch),
      );
      return parseItem(data.data);
    },
    onSuccess: (_data, { id }) => invalidate(queryClient, id),
  });
}

/**
 * DELETE /checkups/custom/{id}. The server cascades the type's records; their
 * on-device reports are not known here (only the latest 2 are cached) — see
 * the T-M4-05 notes on sweeping orphaned files.
 */
export function useDeleteCustomCheckup() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, number>({
    mutationFn: async (id) => {
      await apiClient.delete(`/checkups/custom/${id}`);
    },
    onSuccess: (_data, id) => {
      queryClient.removeQueries({ queryKey: checkupKeys.detail(id) });
      invalidate(queryClient);
      void queryClient.invalidateQueries({ queryKey: checkupKeys.recordsAll() });
    },
  });
}

export interface UpdateCheckupSettingsVars extends Partial<CheckupSettings> {
  /** The checkup type id. */
  id: number;
}

/**
 * PUT /checkups/{id}/settings `{enabled, remind}` — the plan-settings switches
 * and the detail's bell. The cached detail flips instantly and rolls back on
 * failure; the list and home card refetch (a disabled type leaves the plan).
 */
export function useUpdateCheckupSettings() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, UpdateCheckupSettingsVars, { previous?: CheckupDetail }>({
    mutationFn: async ({ id, ...settings }) => {
      const body: Record<string, boolean> = {};
      if (settings.enabled !== undefined) body.enabled = settings.enabled;
      if (settings.remind !== undefined) body.remind = settings.remind;
      await apiClient.put(`/checkups/${id}/settings`, body);
    },
    onMutate: async ({ id, ...settings }) => {
      const key = checkupKeys.detail(id);
      await queryClient.cancelQueries({ queryKey: key });
      const previous = queryClient.getQueryData<CheckupDetail>(key);
      if (previous) {
        const next: CheckupSettings = {
          enabled: settings.enabled ?? previous.settings.enabled,
          remind: settings.remind ?? previous.settings.remind,
        };
        queryClient.setQueryData<CheckupDetail>(key, { ...previous, settings: next });
      }
      return { previous };
    },
    onError: (_error, { id }, context) => {
      if (context?.previous) queryClient.setQueryData(checkupKeys.detail(id), context.previous);
    },
    onSettled: (_data, _error, { id }) => invalidate(queryClient, id),
  });
}
