'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { IvfMedInput, IvfMedsView } from '../model/meds';
import type { IvfDoseInput } from '../model/types';
import { ivfKeys } from './keys';
import {
  ivfGuidanceSchema,
  ivfInjectionSitesSchema,
  ivfMedPresetsSchema,
  ivfMedsViewSchema,
  toIvfMedBody,
} from './meds-schema';

/*
 * The injection schedule (CB-IVF-03 on the CB-IVF-01 API), Go only. Every
 * write answers with the whole schedule, so it lands in the cache as is; the
 * IVF home (today's doses) is refetched. Health data: never log a payload.
 */

export async function fetchIvfMeds(): Promise<IvfMedsView> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/ivf/meds');
  return ivfMedsViewSchema.parse(data.data);
}

/** GET /ivf/meds — trigger, today/tomorrow, site rotation, medicines + inventory. */
export function useIvfMeds() {
  return useQuery({
    queryKey: ivfKeys.meds(),
    queryFn: fetchIvfMeds,
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

function useCatalogGroup<T>(group: string, locale: string, schema: z.ZodType<T, z.ZodTypeDef, unknown>) {
  return useQuery({
    queryKey: ivfKeys.catalog(group, locale),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/catalog/${group}`, {
        params: { audience: 'ttc' },
      });
      return schema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 10 * 60_000,
    retry: 1,
  });
}

/** Catalog `ivf_injection_sites` — admin-editable site names + hints (request locale). */
export function useIvfInjectionSites(locale: string) {
  return useCatalogGroup('ivf_injection_sites', locale, ivfInjectionSitesSchema);
}

/** Catalog `ivf_med_presets` — «افزودن دارو از روی نسخه» pre-fills. */
export function useIvfMedPresets(locale: string) {
  return useCatalogGroup('ivf_med_presets', locale, ivfMedPresetsSchema);
}

/** Catalog `ivf_guidance` — trigger timing / site rotation copy. */
export function useIvfGuidance(locale: string) {
  return useCatalogGroup('ivf_guidance', locale, ivfGuidanceSchema);
}

/** Put a write's schedule into the cache and refresh the IVF home. */
function useApplyMeds() {
  const queryClient = useQueryClient();
  return (view: IvfMedsView) => {
    queryClient.setQueryData(ivfKeys.meds(), view);
    void queryClient.invalidateQueries({ queryKey: ivfKeys.home() });
  };
}

function parseView(data: ApiEnvelope<unknown>): IvfMedsView {
  return ivfMedsViewSchema.parse(data.data);
}

/** POST /ivf/meds — adds a medicine to the open cycle. */
export function useAddIvfMed() {
  const apply = useApplyMeds();
  return useMutation<IvfMedsView, unknown, IvfMedInput>({
    mutationFn: async (input) =>
      parseView((await apiClient.post<ApiEnvelope<unknown>>('/ivf/meds', toIvfMedBody(input))).data),
    onSuccess: apply,
  });
}

/** PUT /ivf/meds/{id} — the whole medicine (send `unitsLeft` back to keep the stock count). */
export function useUpdateIvfMed() {
  const apply = useApplyMeds();
  return useMutation<IvfMedsView, unknown, { id: number; input: IvfMedInput }>({
    mutationFn: async ({ id, input }) =>
      parseView((await apiClient.put<ApiEnvelope<unknown>>(`/ivf/meds/${id}`, toIvfMedBody(input))).data),
    onSuccess: apply,
  });
}

/** DELETE /ivf/meds/{id} — removes the medicine (and its care reminder). */
export function useDeleteIvfMed() {
  const apply = useApplyMeds();
  return useMutation<IvfMedsView, unknown, number>({
    mutationFn: async (id) => parseView((await apiClient.delete<ApiEnvelope<unknown>>(`/ivf/meds/${id}`)).data),
    onSuccess: apply,
  });
}

/** POST /ivf/meds/{id}/doses with the injection site — the schedule's answer (new `suggested`). */
export function useLogIvfScheduleDose() {
  const apply = useApplyMeds();
  return useMutation<IvfMedsView, unknown, IvfDoseInput>({
    mutationFn: async ({ medId, date, slot, site }) =>
      parseView(
        (
          await apiClient.post<ApiEnvelope<unknown>>(`/ivf/meds/${medId}/doses`, {
            date,
            slot,
            site: site ?? null,
          })
        ).data,
      ),
    onSuccess: apply,
  });
}

/** DELETE /ivf/meds/{id}/doses — undoes a logged dose (and its site). */
export function useUnlogIvfScheduleDose() {
  const apply = useApplyMeds();
  return useMutation<IvfMedsView, unknown, IvfDoseInput>({
    mutationFn: async ({ medId, date, slot }) =>
      parseView(
        (await apiClient.delete<ApiEnvelope<unknown>>(`/ivf/meds/${medId}/doses`, { params: { date, slot } })).data,
      ),
    onSuccess: apply,
  });
}

/**
 * PUT /care/medications/{reminderId} {is_active} — CB-IVF-06b: pause / resume a
 * cycle medicine (the IVF med IS a care medication reminder; `PUT /ivf/meds/{id}`
 * keeps the switch as is). A paused medicine leaves today's doses and the
 * reminders. Every IVF read is refreshed.
 */
export function useSetIvfMedActive() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, { reminderId: number; active: boolean }>({
    mutationFn: async ({ reminderId, active }) => {
      await apiClient.put<ApiEnvelope<unknown>>(`/care/medications/${reminderId}`, { is_active: active });
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ivfKeys.all }),
  });
}
