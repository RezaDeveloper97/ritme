'use client';

import { useQuery } from '@tanstack/react-query';

import { analysisKeys } from '@/entities/analysis';
import { type ApiEnvelope, apiClient, getApiErrorCode } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { type PregnancyAnalysis, pregnancyAnalysisSchema } from './schema';

/** `GET /analysis/pregnancy` (B-N3-12) — one payload for the hub and the weight-gain screen. Never log it (§11). */
export async function fetchPregnancyAnalysis(): Promise<PregnancyAnalysis> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/analysis/pregnancy');
  return pregnancyAnalysisSchema.parse(data.data);
}

/** Keyed under `analysisKeys.all`, so a day log or a Plus change refreshes it with the rest of the tab. */
export const pregnancyAnalysisKey = analysisKeys.report('pregnancy', 'all');

export function usePregnancyAnalysis() {
  return useQuery({
    queryKey: pregnancyAnalysisKey,
    queryFn: fetchPregnancyAnalysis,
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: false,
  });
}

/** 409 `pregnancy_not_active`: pregnancy mode is off or undated → the setup prompt instead of an error. */
export function isPregnancyInactive(error: unknown): boolean {
  return getApiErrorCode(error) === 'pregnancy_not_active';
}
