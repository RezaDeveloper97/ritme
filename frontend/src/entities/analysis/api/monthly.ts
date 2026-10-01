'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import {
  MONTHLY_CALENDARS,
  MONTHLY_METRIC_KEYS,
  type BloodPressureValue,
  type MonthlyCalendar,
  type MonthlyMetric,
  type MonthlyMetricKey,
  type MonthlyReport,
} from '../model/monthly';
import { analysisKeys } from './keys';
import { phraseSchema } from './schema';

/** Output type `T`, any input (the transforms rename snake_case keys). */
type Parser<T> = z.ZodType<T, z.ZodTypeDef, unknown>;

const bpSchema: Parser<BloodPressureValue> = z.object({ systolic: z.number(), diastolic: z.number() });
const metricValue = z.union([z.number(), bpSchema]).nullable().catch(null);

/** A row with a key this bundle doesn't know is dropped (a newer backend may add one). */
const metricsSchema: Parser<MonthlyMetric[]> = z
  .array(z.object({ key: z.string(), value: metricValue, delta: metricValue, unit: z.string().nullable().catch(null) }))
  .transform((rows) =>
    rows.filter((r): r is typeof r & { key: MonthlyMetricKey } => (MONTHLY_METRIC_KEYS as readonly string[]).includes(r.key)),
  );

/** `data` of `GET /analysis/monthly/:ym` (backend-go/internal/analysis/monthly.go). */
export const monthlyReportSchema: Parser<MonthlyReport> = z
  .object({
    month: z.object({
      key: z.string(),
      calendar: z.enum(MONTHLY_CALENDARS),
      from: z.string(),
      to: z.string(),
      complete: z.boolean(),
    }),
    headline: phraseSchema,
    summary: z.object({ parts: z.array(phraseSchema), text: z.string() }),
    metrics: metricsSchema,
    top_symptoms: z.array(z.object({ key: z.string(), label: z.string(), days: z.number(), cycles: z.number() })),
    suggestion: phraseSchema,
    pdf: z.object({ plus: z.boolean(), locked: z.boolean() }),
  })
  .transform((r) => ({
    month: r.month,
    headline: r.headline,
    summary: r.summary,
    metrics: r.metrics,
    topSymptoms: r.top_symptoms,
    suggestion: r.suggestion,
    pdf: r.pdf,
  }));

/** GET /analysis/monthly/:ym?calendar= — `ym` is `YYYY-MM` in that calendar. Never log the result (§11). */
export async function fetchMonthlyReport(ym: string, calendar: MonthlyCalendar): Promise<MonthlyReport> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/analysis/monthly/${encodeURIComponent(ym)}`, {
    params: { calendar },
  });
  return monthlyReportSchema.parse(data.data);
}

/**
 * The monthly report (An_Monthly, B-N3-10). No placeholder data: a month
 * switch shows the skeleton rather than last month's figures under the new
 * month's title. `enabled: false` for an `ym` the screen already knows is invalid.
 */
export function useMonthlyReport(ym: string, calendar: MonthlyCalendar, options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: analysisKeys.monthly(ym, calendar),
    queryFn: () => fetchMonthlyReport(ym, calendar),
    enabled: isAuthenticated() && options.enabled !== false,
    staleTime: 60_000,
    retry: false,
  });
}
