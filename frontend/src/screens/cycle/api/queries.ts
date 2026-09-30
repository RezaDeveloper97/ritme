'use client';

import { useQuery } from '@tanstack/react-query';

import { cycleKeys } from '@/entities/cycle';
import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import {
  cycleHistorySchema,
  symptomPatternSchema,
  type CycleHistory,
  type SymptomPattern,
} from '../model/schema';

/**
 * Query keys for the two B-N1-08 read models (Go only). Both sit under
 * `cycleKeys.all`, so logging or editing a period (which invalidates it)
 * refreshes the history and the pattern too.
 */
export const cycleScreenKeys = {
  history: () => [...cycleKeys.all, 'history'] as const,
  symptomPattern: () => [...cycleKeys.all, 'symptom-pattern'] as const,
};

/** `GET /cycle/history` — the backend computes every number; we only render. */
export async function fetchCycleHistory(): Promise<CycleHistory> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/cycle/history');
  return cycleHistorySchema.parse(data.data);
}

/** `GET /cycle/symptom-pattern` — heat strips over the typical cycle. */
export async function fetchSymptomPattern(): Promise<SymptomPattern> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/cycle/symptom-pattern');
  return symptomPatternSchema.parse(data.data);
}

export function useCycleHistory() {
  return useQuery({
    queryKey: cycleScreenKeys.history(),
    queryFn: fetchCycleHistory,
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: false,
  });
}

export function useSymptomPattern() {
  return useQuery({
    queryKey: cycleScreenKeys.symptomPattern(),
    queryFn: fetchSymptomPattern,
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
  });
}
