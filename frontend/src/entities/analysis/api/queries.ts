'use client';

import { keepPreviousData, useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { AnalysisRangeKey, AnalysisSummary } from '../model/types';
import { analysisKeys } from './keys';
import { analysisSummarySchema } from './schema';

/*
 * Reads for `/api/v1/analysis/*` (Go only, B-N3-07). Disabled until a token
 * exists. The texts inside come rendered in the request language
 * (`Accept-Language` from the shared client). Never log what these return (§11).
 */

/** GET /analysis/summary?range= — the hub. */
export async function fetchAnalysisSummary(range: AnalysisRangeKey): Promise<AnalysisSummary> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/analysis/summary', { params: { range } });
  return analysisSummarySchema.parse(data.data);
}

/**
 * The hub for `range`. The previous range's data stays on screen while the new
 * one loads (`isPlaceholderData`), so the range tabs never flash a skeleton.
 */
export function useAnalysisSummary(range: AnalysisRangeKey, options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: analysisKeys.summary(range),
    queryFn: () => fetchAnalysisSummary(range),
    enabled: isAuthenticated() && options.enabled !== false,
    placeholderData: keepPreviousData,
    staleTime: 60_000,
    retry: false,
  });
}
