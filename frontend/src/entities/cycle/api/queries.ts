'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import {
  cycleCalculationEnvelopeSchema,
  cycleMonthEnvelopeSchema,
  cycleStatusSchema,
} from './schema';

/**
 * Query-key factory for cycle calculations (CLAUDE.md §8). All cache reads and
 * post-mutation invalidation go through this — never hand-written arrays.
 */
export const cycleKeys = {
  all: ['cycle'] as const,
  today: () => [...cycleKeys.all, 'today'] as const,
  status: () => [...cycleKeys.all, 'status'] as const,
  date: (date: string) => [...cycleKeys.all, 'date', date] as const,
  month: (year: number, month: number) =>
    [...cycleKeys.all, 'month', year, month] as const,
};

type CycleCalculationEnvelope = z.infer<typeof cycleCalculationEnvelopeSchema>;
type CycleMonthEnvelope = z.infer<typeof cycleMonthEnvelopeSchema>;
type CycleStatus = z.infer<typeof cycleStatusSchema>;

/** GET /cycle/today — today's phase, fertility window and probability. */
export async function fetchCycleToday(): Promise<CycleCalculationEnvelope> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/cycle/today');
  return cycleCalculationEnvelopeSchema.parse(data.data);
}

/**
 * Reads today's cycle calculation (§8 — server state stays in TanStack Query).
 * Disabled until a token exists so it never fires on public screens. If the
 * backend is still recalculating, callers can poll via {@link useCycleStatus}.
 */
export function useCycleToday() {
  return useQuery({
    queryKey: cycleKeys.today(),
    queryFn: fetchCycleToday,
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
  });
}

/** GET /cycle/status — whether calculations are currently being processed. */
export async function fetchCycleStatus(): Promise<CycleStatus> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/cycle/status');
  return cycleStatusSchema.parse(data.data);
}

/**
 * Polls the calculation status while the backend is recalculating, so the home
 * screen can show a "updating your cycle" state and refresh when it settles.
 */
export function useCycleStatus(options?: { poll?: boolean }) {
  return useQuery({
    queryKey: cycleKeys.status(),
    queryFn: fetchCycleStatus,
    enabled: isAuthenticated(),
    refetchInterval: options?.poll ? 4_000 : false,
    retry: false,
  });
}

/** GET /cycle/month/{year}/{month} — a month of calculations for the calendar. */
export async function fetchCycleMonth(
  year: number,
  month: number,
): Promise<CycleMonthEnvelope> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(
    `/cycle/month/${year}/${month}`,
  );
  return cycleMonthEnvelopeSchema.parse(data.data);
}

/** Reads a full month of calculations (week strip / calendar coloring). */
export function useCycleMonth(year: number, month: number) {
  return useQuery({
    queryKey: cycleKeys.month(year, month),
    queryFn: () => fetchCycleMonth(year, month),
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
  });
}

/**
 * POST /cycle/recalculate — ask the backend to recompute the user's cycle after
 * new data. Invalidates every cycle key so the UI refetches the fresh result.
 */
export function useRecalculateCycle() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, void>({
    mutationFn: async () => {
      await apiClient.post('/cycle/recalculate');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: cycleKeys.all });
    },
  });
}
