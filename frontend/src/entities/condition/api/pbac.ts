'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { PBAC_CLOTS, PBAC_MAX_PADS, type PbacDay, type PbacEntry } from '../model/pbac';
import { conditionKeys } from './keys';

/*
 * `GET|PUT /api/v1/conditions/pbac/{date}` (CB-COND-01), Go only. Pad counts are health data (§11): never log a
 * payload, a response or an error body.
 */

export const pbacKeys = {
  all: () => [...conditionKeys.all, 'pbac'] as const,
  day: (date: string) => [...pbacKeys.all(), date] as const,
};

const count = z.number().int().min(0).max(PBAC_MAX_PADS).catch(0);
const score = z.number().int().min(0).catch(0);
const text = z.string().trim().min(1).nullable().catch(null);

const periodSchema = z
  .object({
    start: z.string(),
    end: z.string(),
    day: z.number().int().min(1),
    score,
    days_logged: z.number().int().min(0).catch(0),
  })
  .transform((p) => ({ start: p.start, end: p.end, day: p.day, score: p.score, daysLogged: p.days_logged }));

const alertSchema = z.object({ code: z.string(), title: text, body: text });

export const pbacDaySchema = z
  .object({
    date: z.string().regex(/^\d{4}-\d{2}-\d{2}$/),
    light: count,
    medium: count,
    heavy: count,
    clots: z.enum(PBAC_CLOTS).nullable().catch(null),
    flooding: z.boolean().catch(false),
    score,
    period: periodSchema.nullable().catch(null),
    alert_threshold: z.number().int().min(1).catch(100),
    alert: alertSchema.nullable().catch(null),
  })
  .transform(
    (d): PbacDay => ({
      date: d.date,
      light: d.light,
      medium: d.medium,
      heavy: d.heavy,
      clots: d.clots,
      flooding: d.flooding,
      score: d.score,
      period: d.period,
      alertThreshold: d.alert_threshold,
      alert: d.alert,
    }),
  );

/** GET /conditions/pbac/{date} — the day's pads with its period score. */
export function usePbacDay(date: string) {
  return useQuery<PbacDay>({
    queryKey: pbacKeys.day(date),
    queryFn: async () =>
      pbacDaySchema.parse((await apiClient.get<ApiEnvelope<unknown>>(`/conditions/pbac/${date}`)).data.data),
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

/** PUT /conditions/pbac/{date} — partial save (an omitted key stays; needs the heavy_bleeding enrolment); answers the day. */
export function useSavePbacDay() {
  const queryClient = useQueryClient();
  return useMutation<PbacDay, unknown, { date: string; entry: Partial<PbacEntry> }>({
    mutationFn: async ({ date, entry }) =>
      pbacDaySchema.parse((await apiClient.put<ApiEnvelope<unknown>>(`/conditions/pbac/${date}`, entry)).data.data),
    onSuccess: (day) => {
      queryClient.setQueryData(pbacKeys.day(day.date), day);
      // Other days of the same period carry the period score; the flow may change this cycle's nudges.
      void queryClient.invalidateQueries({ queryKey: pbacKeys.all(), predicate: (q) => q.queryKey[2] !== day.date });
      void queryClient.invalidateQueries({ queryKey: conditionKeys.nudges() });
    },
  });
}
