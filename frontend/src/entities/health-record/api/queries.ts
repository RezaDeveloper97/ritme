'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { BasicsInput, ConditionsInput, HealthRecord, PregnancyEntry, PregnancyInput } from '../model/types';
import { healthRecordKeys } from './keys';
import { healthRecordSchema, pregnancyEntrySchema } from './schema';

/*
 * `/api/v1/health-record*` (bloom B-N6-03), Go only, owner-only. Health data of the most sensitive kind (§11): never
 * log a payload, a response or an error body, and never put any of it in a URL.
 */

export async function fetchHealthRecord(): Promise<HealthRecord> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/health-record');
  return healthRecordSchema.parse(data.data);
}

/** GET /health-record — every section of the owner's record. */
export function useHealthRecord() {
  return useQuery({
    queryKey: healthRecordKeys.record(),
    queryFn: fetchHealthRecord,
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

/** PUT /health-record/basics — blood type and / or allergies; answers the whole record. */
export function useSaveRecordBasics() {
  const queryClient = useQueryClient();
  return useMutation<HealthRecord, unknown, BasicsInput>({
    mutationFn: async (input) => {
      const body: Record<string, unknown> = {};
      if (input.bloodType !== undefined) body.blood_type = input.bloodType;
      if (input.allergies !== undefined) body.allergies = input.allergies;
      const { data } = await apiClient.put<ApiEnvelope<unknown>>('/health-record/basics', body);
      return healthRecordSchema.parse(data.data);
    },
    onSuccess: (record) => queryClient.setQueryData(healthRecordKeys.record(), record),
  });
}

/**
 * PUT /onboarding/steps/conditions — the record's conditions are the onboarding Conditions answers (B-N2-01); the
 * stored medications list is sent back unchanged so this edit never clears it.
 */
export function useSaveRecordConditions() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, ConditionsInput & { medications: string[] | null }>({
    mutationFn: async (input) => {
      await apiClient.put<ApiEnvelope<unknown>>('/onboarding/steps/conditions', {
        chronic_illnesses: input.chronicIllnesses,
        gyn_conditions: input.gynConditions,
        medications: input.medications,
      });
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: healthRecordKeys.all }),
  });
}

function pregnancyBody(input: PregnancyInput): Record<string, unknown> {
  return {
    outcome: input.outcome,
    ended_on: input.endedOn,
    baby_count: input.outcome === 'ended' ? null : input.babyCount,
  };
}

/** POST /health-record/pregnancies (create) or PUT /health-record/pregnancies/{id} (replace). */
export function useSaveRecordPregnancy() {
  const queryClient = useQueryClient();
  return useMutation<PregnancyEntry, unknown, { id: number | null; input: PregnancyInput }>({
    mutationFn: async ({ id, input }) => {
      const { data } =
        id === null
          ? await apiClient.post<ApiEnvelope<unknown>>('/health-record/pregnancies', pregnancyBody(input))
          : await apiClient.put<ApiEnvelope<unknown>>(`/health-record/pregnancies/${id}`, pregnancyBody(input));
      return pregnancyEntrySchema.parse(data.data);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: healthRecordKeys.all }),
  });
}

/** DELETE /health-record/pregnancies/{id}. */
export function useDeleteRecordPregnancy() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, number>({
    mutationFn: async (id) => {
      await apiClient.delete(`/health-record/pregnancies/${id}`);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: healthRecordKeys.all }),
  });
}
