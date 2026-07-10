'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient, getApiErrorStatus } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { HealthLogEnums, HealthLogInput } from '../model/types';
import { dailyHealthLogSchema, healthLogEnumsSchema } from './schema';

/**
 * Query-key factory for daily health logs (CLAUDE.md §8). Each day is keyed by
 * its API date so days cache independently; all reads and post-mutation
 * invalidation go through here — never hand-written arrays.
 */
export const healthLogKeys = {
  all: ['health-log'] as const,
  enums: () => [...healthLogKeys.all, 'enums'] as const,
  date: (date: string) => [...healthLogKeys.all, 'date', date] as const,
};

/** GET /health-logs/enums — option lists that drive the log form. */
export async function fetchHealthLogEnums(): Promise<HealthLogEnums> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/health-logs/enums');
  return healthLogEnumsSchema.parse(data.data);
}

/**
 * Reads the form enums (§8 — server state stays in TanStack Query). Enums rarely
 * change, so we cache them for the session and never refetch on focus.
 */
export function useHealthLogEnums() {
  return useQuery({
    queryKey: healthLogKeys.enums(),
    queryFn: fetchHealthLogEnums,
    enabled: isAuthenticated(),
    staleTime: Infinity,
    gcTime: Infinity,
    retry: false,
  });
}

/**
 * GET /health-logs/{date} — the saved log for a day, or `null` when nothing has
 * been recorded yet (the API answers 404). Never logs the payload (§11).
 */
export async function fetchHealthLog(date: string): Promise<HealthLogInput | null> {
  try {
    const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/health-logs/${date}`);
    return dailyHealthLogSchema.parse(data.data);
  } catch (error) {
    if (getApiErrorStatus(error) === 404) return null;
    throw error;
  }
}

/** Reads a day's saved log so the form can prefill (empty day → `null`). */
export function useHealthLog(date: string) {
  return useQuery({
    queryKey: healthLogKeys.date(date),
    queryFn: () => fetchHealthLog(date),
    enabled: isAuthenticated() && date.length > 0,
    staleTime: 60_000,
    retry: false,
  });
}

/**
 * Merge a patch onto a cached day's log. `null` clears a field (the user
 * deselected it); `undefined`/`null` values never linger in the draft (§8 —
 * cache mirrors the saved shape). `log_date` is always preserved.
 */
function mergeLogPatch(
  base: HealthLogInput | null | undefined,
  patch: HealthLogInput,
): HealthLogInput {
  const next = { ...(base ?? { log_date: patch.log_date }), log_date: patch.log_date } as unknown as Record<string, unknown>;
  for (const [key, value] of Object.entries(patch)) {
    if (key === 'log_date') continue;
    if (value === null || value === undefined) delete next[key];
    else next[key] = value;
  }
  return next as unknown as HealthLogInput;
}

/**
 * POST /health-logs — create or update a day's log (the endpoint upserts by
 * `log_date` and merges only the fields present in the body, so we send small
 * per-field patches for instant auto-save). Optimistic: the cache updates the
 * moment the user taps an option, then reconciles with the server's saved
 * state; a failed save rolls back. Cycle recalculation is the backend's job and
 * stays out of this slice (no sibling-entity coupling — §12).
 */
export function useSaveHealthLog() {
  const queryClient = useQueryClient();
  return useMutation<
    HealthLogInput,
    unknown,
    HealthLogInput,
    { key: readonly unknown[]; previous: HealthLogInput | null | undefined }
  >({
    mutationFn: async (input) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/health-logs', input);
      return dailyHealthLogSchema.parse(data.data);
    },
    onMutate: async (input) => {
      const key = healthLogKeys.date(input.log_date);
      await queryClient.cancelQueries({ queryKey: key });
      const previous = queryClient.getQueryData<HealthLogInput | null>(key);
      queryClient.setQueryData<HealthLogInput | null>(key, (old) => mergeLogPatch(old, input));
      return { key, previous };
    },
    onError: (_error, _input, context) => {
      if (context) queryClient.setQueryData(context.key, context.previous);
    },
    onSuccess: (result, input) => {
      // Adopt the server's canonical saved state (nulls already stripped). Key on
      // the request's date — the exact string the cache and screen already use.
      queryClient.setQueryData(healthLogKeys.date(input.log_date), result);
    },
  });
}
