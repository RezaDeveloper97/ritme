'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { postpartumKeys } from '@/entities/postpartum';
import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

/** One EPDS check of the history — never the answers (B-N5-01). */
export interface EpdsPoint {
  kind: 'short' | 'full';
  takenOn: string;
  week: number | null;
  total: number;
  max: number;
  urgent: boolean;
}

export interface EpdsHistory {
  /** Newest first, as the API sends them. */
  checks: EpdsPoint[];
  thresholds: { shortElevated: number; fullPossible: number; fullLikely: number };
}

const checkSchema = z
  .object({
    kind: z.enum(['short', 'full']),
    taken_on: z.string(),
    week: z.number().int().nullable().catch(null),
    total: z.number().int(),
    max: z.number().int(),
    urgent: z.boolean().catch(false),
  })
  .transform(
    (d): EpdsPoint => ({ kind: d.kind, takenOn: d.taken_on, week: d.week, total: d.total, max: d.max, urgent: d.urgent }),
  );

export const epdsHistorySchema = z
  .object({
    checks: z
      .array(z.unknown())
      .catch([])
      .transform((list) =>
        list.flatMap((raw) => {
          const parsed = checkSchema.safeParse(raw);
          return parsed.success ? [parsed.data] : [];
        }),
      ),
    thresholds: z
      .object({ short_elevated: z.number(), full_possible: z.number(), full_likely: z.number() })
      .catch({ short_elevated: 6, full_possible: 10, full_likely: 13 }),
  })
  .transform(
    (d): EpdsHistory => ({
      checks: d.checks,
      thresholds: {
        shortElevated: d.thresholds.short_elevated,
        fullPossible: d.thresholds.full_possible,
        fullLikely: d.thresholds.full_likely,
      },
    }),
  );

export async function fetchEpdsHistory(): Promise<EpdsHistory> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/postpartum/epds', { params: { limit: 26 } });
  return epdsHistorySchema.parse(data.data);
}

/**
 * GET /postpartum/epds — the scored checks for the hub's trend. Keyed on
 * `postpartumKeys.history()`, which the EPDS submit already invalidates.
 * Never log it (CLAUDE.md §11).
 */
export function useEpdsHistory() {
  return useQuery({
    queryKey: postpartumKeys.history(),
    queryFn: fetchEpdsHistory,
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}
