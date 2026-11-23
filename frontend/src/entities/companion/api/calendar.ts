'use client';

import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient, getApiErrorStatus } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { CompanionCalendar } from '../model/calendar';
import { companionSideKeys } from './companion-side';

const isoDay = /^\d{4}-\d{2}-\d{2}$/;

/** Lenient: a malformed list reads as «not shared» (null), the most private reading. */
export const companionCalendarSchema = z
  .object({
    from: z.string(),
    to: z.string(),
    partners: z
      .array(
        z.object({
          link_id: z.number(),
          partner_name: z.string().nullable().catch(null).optional(),
          period_days: z.array(z.string().regex(isoDay)).nullable().catch(null).optional(),
        }),
      )
      .catch([]),
  })
  .transform(
    (d): CompanionCalendar => ({
      from: d.from,
      to: d.to,
      partners: d.partners.map((p) => ({
        linkId: p.link_id,
        partnerName: p.partner_name ?? null,
        periodDays: p.period_days ?? null,
      })),
    }),
  );

export const companionCalendarKeys = {
  range: (from: string, to: string) => [...companionSideKeys.all, 'calendar', from, to] as const,
};

/** GET /companion/calendar?from&to — `Y-m-d`, at most 62 days apart (the server clamps to today … today + 92). */
export async function fetchCompanionCalendar(from: string, to: string): Promise<CompanionCalendar> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/companion/calendar', { params: { from, to } });
  return companionCalendarSchema.parse(data.data);
}

export function useCompanionCalendar(from: string, to: string, enabled = true) {
  return useQuery({
    queryKey: companionCalendarKeys.range(from, to),
    queryFn: () => fetchCompanionCalendar(from, to),
    enabled: enabled && isAuthenticated(),
    staleTime: 5 * 60_000,
    placeholderData: keepPreviousData,
    retry: (count, error) => getApiErrorStatus(error) !== 403 && count < 2,
  });
}
