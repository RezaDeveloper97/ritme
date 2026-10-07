'use client';

import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { menopauseSectionSchema, type MenopauseSection } from '@/entities/health-record';
import { menopauseKeys } from '@/entities/menopause';
import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

/** «۱ ماه · ۳ ماه · ۶ ماه» (CB-MENO-03 `?months=`). */
export const MENO_REPORT_MONTHS = [1, 3, 6] as const;
export type MenoReportMonths = (typeof MENO_REPORT_MONTHS)[number];

export interface MenoReport {
  months: number;
  empty: boolean;
  report: MenopauseSection;
}

const menoReportSchema = z
  .object({ months: z.number(), empty: z.boolean(), report: menopauseSectionSchema })
  .transform((d): MenoReport => d);

/**
 * GET /menopause/report?months= — the owner's preview of the menopause doctor report (CB-MENO-03): the same data as
 * the builder's `menopause` section. Symptom labels are localized by `Accept-Language` (hence the locale in the key).
 * Health data (§11): never logged.
 */
export function useMenopauseReport(months: MenoReportMonths, locale: string) {
  return useQuery({
    queryKey: [...menopauseKeys.all, 'report', months, locale] as const,
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/menopause/report', { params: { months } });
      return menoReportSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    placeholderData: keepPreviousData,
    staleTime: 30_000,
    retry: 1,
  });
}
