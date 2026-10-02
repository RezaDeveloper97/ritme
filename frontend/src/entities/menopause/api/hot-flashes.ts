'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { HotFlashDetails, MenopauseCatalogItem, MenopauseFlashDay } from '../model/types';
import { menopauseKeys } from './keys';
import { catalogItemSchema, menopauseFlashSchema } from './schema';

/*
 * The hot-flash timer screen (CB-MENO-07, nbl_Meno_HotFlash) on
 * `GET /menopause/hot-flashes` (CB-MENO-02) and its tip from the `meno_tips`
 * catalog. Health data (§11): never log a payload or a response.
 */

/** Snake-case details of a start / stop / edit (`null` severity clears it). */
export function toHotFlashDetailsBody(details: HotFlashDetails): Record<string, unknown> {
  return { severity: details.severity, sweat: details.sweat, triggers: details.triggers };
}

/** Drops the flashes that fail to parse instead of failing the day. */
const flashListSchema = z
  .array(z.unknown())
  .catch([])
  .transform((rows) =>
    rows.flatMap((row) => {
      const parsed = menopauseFlashSchema.safeParse(row);
      return parsed.success ? [parsed.data] : [];
    }),
  );

export const menopauseFlashDaySchema = z
  .object({
    date: z.string(),
    count: z.number().int().catch(0),
    night_count: z.number().int().catch(0),
    avg_duration_s: z.number().nullable().catch(null),
    running: menopauseFlashSchema.nullable().catch(null),
    items: flashListSchema,
  })
  .transform(
    (d): MenopauseFlashDay => ({
      date: d.date,
      count: d.count,
      nightCount: d.night_count,
      avgDurationS: d.avg_duration_s,
      running: d.running,
      items: d.items,
    }),
  );

export async function fetchHotFlashDay(date: string | null): Promise<MenopauseFlashDay> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/menopause/hot-flashes', {
    params: date ? { date } : undefined,
  });
  return menopauseFlashDaySchema.parse(data.data);
}

/** GET /menopause/hot-flashes — the day's flashes, tiles and the timer still running (`date` null = today). */
export function useHotFlashDay(date: string | null = null) {
  return useQuery({
    queryKey: menopauseKeys.hotFlashDay(date),
    queryFn: () => fetchHotFlashDay(date),
    enabled: isAuthenticated(),
    staleTime: 15_000,
    retry: 1,
  });
}

const tipsSchema = z
  .object({ items: z.array(z.unknown()).catch([]) })
  .transform((d) =>
    d.items.flatMap((row): MenopauseCatalogItem[] => {
      const parsed = catalogItemSchema.safeParse(row);
      return parsed.success ? [parsed.data] : [];
    }),
  );

export async function fetchMenopauseTips(): Promise<MenopauseCatalogItem[]> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/catalog/meno_tips', { params: { audience: 'menopause' } });
  return tipsSchema.parse(data.data);
}

/**
 * GET /catalog/meno_tips — the admin-editable menopause tips (localized by
 * `Accept-Language`, hence the locale in the key); content, not health data.
 */
export function useMenopauseTips(locale: string) {
  return useQuery({
    queryKey: menopauseKeys.tips(locale),
    queryFn: fetchMenopauseTips,
    enabled: isAuthenticated(),
    staleTime: 10 * 60_000,
    retry: 1,
  });
}
