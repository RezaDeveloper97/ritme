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
 * POST /health-logs — create or update a day's log (the endpoint upserts by
 * `log_date`). On success we invalidate that day so the form reflects the saved
 * state. Cycle recalculation is a separate concern and stays out of this slice
 * (no sibling-entity coupling — §12).
 */
export function useSaveHealthLog() {
  const queryClient = useQueryClient();
  return useMutation<HealthLogInput, unknown, HealthLogInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/health-logs', input);
      return dailyHealthLogSchema.parse(data.data);
    },
    onSuccess: (_result, input) => {
      queryClient.invalidateQueries({ queryKey: healthLogKeys.date(input.log_date) });
    },
  });
}
