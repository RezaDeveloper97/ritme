'use client';

import { keepPreviousData, useQuery } from '@tanstack/react-query';
import type { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { BodyReport, CorrelationsReport, CycleReport, PeriodReport, ReportName, SymptomsReport } from '../model/reports';
import type { AnalysisRangeKey } from '../model/types';
import { analysisKeys } from './keys';
import { bodyReportSchema, correlationsReportSchema, cycleReportSchema, periodReportSchema, symptomsReportSchema } from './reports';

/*
 * Reads of the detail reports (B-N3-09), same rules as the hub: disabled
 * without a token, rendered in the request language, the previous range stays
 * on screen while the next one loads. Never log what these return (§11).
 */

type Parser<T> = z.ZodType<T, z.ZodTypeDef, unknown>;

async function fetchReport<T>(name: ReportName, range: AnalysisRangeKey, schema: Parser<T>): Promise<T> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/analysis/${name}`, { params: { range } });
  return schema.parse(data.data);
}

function useReport<T>(name: ReportName, range: AnalysisRangeKey, schema: Parser<T>) {
  return useQuery({
    queryKey: analysisKeys.report(name, range),
    queryFn: () => fetchReport(name, range, schema),
    enabled: isAuthenticated(),
    placeholderData: keepPreviousData,
    staleTime: 60_000,
    retry: false,
  });
}

/** GET /analysis/cycle?range= (An_Cycle). */
export const useCycleReport = (range: AnalysisRangeKey) => useReport<CycleReport>('cycle', range, cycleReportSchema);
/** GET /analysis/period?range= (An_Period). */
export const usePeriodReport = (range: AnalysisRangeKey) => useReport<PeriodReport>('period', range, periodReportSchema);
/** GET /analysis/symptoms?range= (An_Symptoms). */
export const useSymptomsReport = (range: AnalysisRangeKey) => useReport<SymptomsReport>('symptoms', range, symptomsReportSchema);
/** GET /analysis/correlations?range= (An_Correlations, Plus — 200 `locked` for a free user). */
export const useCorrelationsReport = (range: AnalysisRangeKey) =>
  useReport<CorrelationsReport>('correlations', range, correlationsReportSchema);
/** GET /analysis/body?range= (An_Body). */
export const useBodyReport = (range: AnalysisRangeKey) => useReport<BodyReport>('body', range, bodyReportSchema);
