'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { PAIN_ANALGESIC_EFFECTS, type PainCatalogGroup, type PainDay, type PainDayInput } from '../model/pain';
import type { ConditionCatalogItem } from '../model/types';
import { conditionKeys } from './keys';
import { conditionCatalogSchema } from './schema';

/*
 * `GET|PUT /api/v1/conditions/pain/{date}` (CB-COND-01, Go only) — the endometriosis pain diary. Which conditions she
 * lives with and her pain are health data (§11): never log a payload, a response or an error body.
 */

const codes = z
  .array(z.unknown())
  .nullable()
  .catch([])
  .transform((l) => (l ?? []).filter((c): c is string => typeof c === 'string' && c.length > 0));
const clock = z.string().regex(/^\d{2}:\d{2}$/).nullable().catch(null);

export const painDaySchema = z
  .object({
    date: z.string(),
    score: z.number().int().min(0).max(10).nullable().catch(null),
    locations: codes,
    relief: codes,
    types: codes,
    associated: codes,
    missed_activity: z.boolean().nullable().catch(null),
    analgesic: z.string().trim().min(1).nullable().catch(null),
    analgesic_time: clock,
    analgesic_effect: z.enum(PAIN_ANALGESIC_EFFECTS).nullable().catch(null),
  })
  .transform(
    (d): PainDay => ({
      date: d.date,
      score: d.score,
      locations: d.locations,
      relief: d.relief,
      types: d.types,
      associated: d.associated,
      missedActivity: d.missed_activity,
      analgesic: d.analgesic,
      analgesicTime: d.analgesic_time,
      analgesicEffect: d.analgesic_effect,
    }),
  );

/** Query keys of the pain diary (under the condition entity's root, so leaving a program can drop them). */
export const painKeys = {
  day: (date: string) => [...conditionKeys.all, 'pain', date] as const,
};

/** GET /conditions/pain/{date}. */
export function usePainDay(date: string) {
  return useQuery<PainDay>({
    queryKey: painKeys.day(date),
    queryFn: async () =>
      painDaySchema.parse((await apiClient.get<ApiEnvelope<unknown>>(`/conditions/pain/${date}`)).data.data),
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

/** PUT /conditions/pain/{date} — partial update; answers the saved day. Pain also feeds the nudges. */
export function useSavePainDay(date: string) {
  const queryClient = useQueryClient();
  return useMutation<PainDay, unknown, PainDayInput>({
    mutationFn: async (input) =>
      painDaySchema.parse((await apiClient.put<ApiEnvelope<unknown>>(`/conditions/pain/${date}`, input)).data.data),
    onSuccess: (day) => {
      queryClient.setQueryData(painKeys.day(date), day);
      void queryClient.invalidateQueries({ queryKey: conditionKeys.nudges() });
    },
  });
}

/** GET /catalog/pain_types | pain_associated — admin-editable chip labels in the request locale. */
export function usePainCatalog(group: PainCatalogGroup, locale: string) {
  return useQuery<ConditionCatalogItem[]>({
    queryKey: conditionKeys.catalog(group, locale),
    queryFn: async () =>
      conditionCatalogSchema.parse((await apiClient.get<ApiEnvelope<unknown>>(`/catalog/${group}`)).data.data),
    enabled: isAuthenticated(),
    staleTime: 10 * 60_000,
    retry: 1,
  });
}
