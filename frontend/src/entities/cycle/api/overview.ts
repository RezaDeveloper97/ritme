'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { cycleKeys } from './queries';

/**
 * `GET /home/cycle-overview` (B-N1-06, Go only) — the numbers the cycle home
 * shows beside `cycle_view`: logging streak, predicted period range, PMS
 * window and the cycle-length summary. The backend owns the math (resolver +
 * metrics); the client only renders. On a backend without the route (Laravel
 * production until the cutover) the query errors and the home falls back to
 * what `cycle_view` already carries.
 *
 * Keyed under {@link cycleKeys} so logging a period (which invalidates
 * `cycleKeys.all`) refreshes it.
 */
export const cycleOverviewKey = () => [...cycleKeys.all, 'overview'] as const;

const rangeSchema = z.object({ start: z.string(), end: z.string() }).nullable();

const overviewSchema = z
  .object({
    date: z.string(),
    logging: z.object({ streak_days: z.number().int(), logged_today: z.boolean() }),
    next_period: rangeSchema,
    pms_window: rangeSchema,
    cycle_length: z.object({
      median: z.number().int(),
      source: z.string(),
      based_on_cycles: z.number().int().nullable(),
      spread_days: z.number().int().nullable(),
      regularity: z.string(),
    }),
  })
  .transform((o) => ({
    date: o.date,
    streakDays: o.logging.streak_days,
    loggedToday: o.logging.logged_today,
    nextPeriod: o.next_period,
    pmsWindow: o.pms_window,
    cycleLength: {
      median: o.cycle_length.median,
      source: o.cycle_length.source,
      basedOnCycles: o.cycle_length.based_on_cycles,
      spreadDays: o.cycle_length.spread_days,
      regularity: o.cycle_length.regularity,
    },
  }));

export type CycleOverview = z.infer<typeof overviewSchema>;

export async function fetchCycleOverview(): Promise<CycleOverview> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/home/cycle-overview');
  return overviewSchema.parse(data.data);
}

export function useCycleOverview(enabled = true) {
  return useQuery({
    queryKey: cycleOverviewKey(),
    queryFn: fetchCycleOverview,
    enabled: enabled && isAuthenticated(),
    staleTime: 60_000,
    retry: false,
  });
}

export { overviewSchema as cycleOverviewSchema };
