'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type {
  BbtRange,
  FertilityBbt,
  FertilityDay,
  FertilityInsights,
  FertilityToday,
} from '../model/types';
import { fertilityKeys } from './keys';
import {
  fertilityBbtSchema,
  fertilityDaySchema,
  fertilityInsightsSchema,
  fertilityTodaySchema,
} from './schema';

/*
 * Reads for `/api/v1/fertility/*` (CLAUDE.md §8 — server state in TanStack
 * Query). Every hook is disabled until a token exists so public screens never
 * fire it. Personal health data (§11): never log what these return.
 */

/** GET /fertility/today — the three home tiles + today's chance level. */
export async function fetchFertilityToday(): Promise<FertilityToday> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/fertility/today');
  return fertilityTodaySchema.parse(data.data);
}

export function useFertilityToday(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: fertilityKeys.today(),
    queryFn: fetchFertilityToday,
    enabled: isAuthenticated() && options.enabled !== false,
    staleTime: 60_000,
    retry: false,
  });
}

/** GET /fertility/days/{date} — the merged day the log screen edits. */
export async function fetchFertilityDay(date: string): Promise<FertilityDay> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/fertility/days/${date}`);
  return fertilityDaySchema.parse(data.data);
}

export function useFertilityDay(date: string) {
  return useQuery({
    queryKey: fertilityKeys.day(date),
    queryFn: () => fetchFertilityDay(date),
    enabled: isAuthenticated() && date.length > 0,
    staleTime: 60_000,
    retry: false,
  });
}

/** GET /fertility/bbt?range=1|3|6 — chart points, coverline, window, stats. */
export async function fetchFertilityBbt(range: BbtRange = 1): Promise<FertilityBbt> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/fertility/bbt', {
    params: { range },
  });
  const parsed = fertilityBbtSchema.parse(data.data);
  // The server may omit `range`; the request is the truth for the cache entry.
  return { ...parsed, range };
}

export function useFertilityBbt(range: BbtRange = 1) {
  return useQuery({
    queryKey: fertilityKeys.bbt(range),
    queryFn: () => fetchFertilityBbt(range),
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: false,
  });
}

/** GET /fertility/insights — fertile window, confidence, evidence, history. */
export async function fetchFertilityInsights(): Promise<FertilityInsights> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/fertility/insights');
  return fertilityInsightsSchema.parse(data.data ?? {});
}

export function useFertilityInsights() {
  return useQuery({
    queryKey: fertilityKeys.insights(),
    queryFn: fetchFertilityInsights,
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
  });
}
