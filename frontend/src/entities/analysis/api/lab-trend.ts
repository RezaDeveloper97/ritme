'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { LabTrendList, LabTrendMarker } from '../model/lab-trend';
import { analysisKeys } from './keys';
import { phraseSchema } from './schema';

/** Output type `T`, any input (the transforms rename snake_case keys). */
type Parser<T> = z.ZodType<T, z.ZodTypeDef, unknown>;

const direction = z.enum(['rising', 'falling', 'stable']).nullable().catch(null);

const markerSchema: Parser<LabTrendMarker> = z
  .object({
    marker: z.string(),
    name: z.string(),
    unit: z.string().nullable().catch(null),
    reports: z.number(),
    ready: z.boolean(),
    reports_needed: z.number().catch(0),
    direction,
    finding: phraseSchema.nullable().catch(null),
  })
  .transform((m) => ({
    marker: m.marker,
    name: m.name,
    unit: m.unit,
    reports: m.reports,
    ready: m.ready,
    reportsNeeded: m.reports_needed,
    direction: m.direction,
    finding: m.finding,
  }));

/**
 * `data` of `GET /analysis/labs` without `marker` (backend-go/internal/analysis/labtrend.go): the `{plus, locked,
 * ready, data}` section; a locked (free) reader gets no data.
 */
export const labTrendListSchema: Parser<LabTrendList> = z
  .object({
    plus: z.boolean(),
    locked: z.boolean(),
    ready: z.boolean(),
    data: z
      .object({
        min_reports: z.number().catch(2),
        markers: z.array(markerSchema.nullable().catch(null)),
        note: phraseSchema.nullable().catch(null),
      })
      .nullable()
      .catch(null),
  })
  .transform((r) => ({
    locked: r.locked,
    ready: r.ready && !r.locked && r.data != null,
    minReports: r.data?.min_reports ?? 2,
    markers: r.data ? r.data.markers.filter((m): m is LabTrendMarker => m != null) : [],
    note: r.data?.note ?? null,
  }));

/** GET /analysis/labs — every marker's trend status (Plus). Never log the result (§11). */
export async function fetchLabTrendList(): Promise<LabTrendList> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/analysis/labs');
  return labTrendListSchema.parse(data.data);
}

/** The lab-trend list of An_Labs (§4.1 «روند آزمایش»: a marker in ≥ 2 confirmed reports). */
export function useLabTrendList(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: analysisKeys.labTrend(null),
    queryFn: fetchLabTrendList,
    enabled: isAuthenticated() && options.enabled !== false,
    staleTime: 60_000,
    retry: false,
  });
}
