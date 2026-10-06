'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, ApiError, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { DEFAULT_THRESHOLDS } from '../model/classify';
import type {
  GlucoseFilter,
  PlanItem,
  ReadingInput,
  ReportRange,
  SavedReading,
  VitalReading,
  VitalReport,
  VitalsHub,
  VitalsPlan,
  VitalThresholds,
  VitalType,
} from '../model/types';
import { vitalsKeys } from './keys';
import { hubSchema, parseReport, planSchema, readingsSchema, savedSchema, thresholdsSchema } from './schema';

/*
 * `/api/v1/vitals*` (B-N6-01), Go only. Health data (CLAUDE.md §11): never log
 * a payload or a response; values never ride in a URL (the report filter is a
 * context name, not a value).
 */

export async function fetchVitalsHub(): Promise<VitalsHub> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/vitals');
  return hubSchema.parse(data.data);
}

/** GET /vitals — latest per type, this week's plan progress, recent readings. */
export function useVitalsHub() {
  return useQuery({ queryKey: vitalsKeys.hub(), queryFn: fetchVitalsHub, enabled: isAuthenticated(), staleTime: 30_000, retry: 1 });
}

export async function fetchThresholds(): Promise<VitalThresholds> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/vitals/thresholds');
  return thresholdsSchema.parse(data.data);
}

/** GET /vitals/thresholds — the bands the server classifies with (falls back to the bundled copy). */
export function useVitalThresholds(): VitalThresholds {
  const q = useQuery({
    queryKey: vitalsKeys.thresholds(),
    queryFn: fetchThresholds,
    enabled: isAuthenticated(),
    staleTime: 60 * 60_000,
    retry: 1,
  });
  return q.data ?? DEFAULT_THRESHOLDS;
}

export async function fetchReport(type: VitalType, range: ReportRange, filter: GlucoseFilter | null): Promise<VitalReport> {
  const params: Record<string, string> = { range };
  if (type === 'glucose' && filter && filter !== 'all') params.filter = filter;
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/vitals/reports/${type}`, { params });
  return parseReport(type, data.data);
}

/** GET /vitals/reports/{type}?range[&filter] — averages, distribution, morning vs night, daily series. */
export function useVitalReport(type: VitalType, range: ReportRange, filter: GlucoseFilter | null = null) {
  return useQuery({
    queryKey: vitalsKeys.report(type, range, filter),
    queryFn: () => fetchReport(type, range, filter),
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
    placeholderData: (prev) => (prev && prev.type === type ? prev : undefined),
  });
}

export async function fetchReadings(type: VitalType, from: string, to: string): Promise<VitalReading[]> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/vitals/readings', { params: { type, from, to } });
  return readingsSchema.parse(data.data);
}

/** GET /vitals/readings?type&from&to — every reading of the range (timed + log sheet), newest first. */
export function useVitalReadings(type: VitalType, from: string | null, to: string | null) {
  return useQuery({
    queryKey: vitalsKeys.readings(type, from ?? '', to ?? ''),
    queryFn: () => fetchReadings(type, from ?? '', to ?? ''),
    enabled: isAuthenticated() && !!from && !!to,
    staleTime: 30_000,
    retry: 1,
  });
}

export async function fetchPlan(): Promise<VitalsPlan> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/vitals/plan');
  return planSchema.parse(data.data);
}

/** GET /vitals/plan — the weekly measurement plan, its slots and the week's progress. */
export function useVitalsPlan(enabled = true) {
  return useQuery({ queryKey: vitalsKeys.plan(), queryFn: fetchPlan, enabled: enabled && isAuthenticated(), staleTime: 30_000, retry: 1 });
}

/** Body of PUT /vitals/plan. */
export function toPlanBody(items: readonly PlanItem[]): Record<string, unknown> {
  return {
    items: items.map((i) => ({ type: i.type, slot: i.slot, days: [...i.days].sort((a, b) => a - b), remind_at: i.remindAt })),
  };
}

/** PUT /vitals/plan — replaces the plan (≤ 8 items, one per type + slot). */
export function useSavePlan() {
  const queryClient = useQueryClient();
  return useMutation<VitalsPlan, unknown, readonly PlanItem[]>({
    mutationFn: async (items) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>('/vitals/plan', toPlanBody(items));
      return planSchema.parse(data.data);
    },
    onSuccess: (plan) => {
      queryClient.setQueryData(vitalsKeys.plan(), plan);
      void queryClient.invalidateQueries({ queryKey: vitalsKeys.hub() });
    },
  });
}

/** Snake-case body of POST /vitals/readings. */
export function toReadingBody(input: ReadingInput): Record<string, unknown> {
  const common: Record<string, unknown> = { type: input.type };
  if (input.measuredAt) common.measured_at = input.measuredAt;
  if (input.note) common.note = input.note;
  switch (input.type) {
    case 'bp':
      return {
        ...common,
        systolic: input.systolic,
        diastolic: input.diastolic,
        ...(input.pulse ? { pulse: input.pulse } : {}),
        ...(input.arm ? { arm: input.arm } : {}),
        ...(input.position ? { position: input.position } : {}),
      };
    case 'glucose':
      return { ...common, value: input.value, unit: input.unit, context: input.context, ...(input.method ? { method: input.method } : {}) };
    case 'hr':
      return { ...common, bpm: input.bpm, context: input.context };
  }
}

/**
 * POST /vitals/readings → `{reading, alert}`. The reading is stored either way;
 * `alert` (BP > 180/120, glucose < 54 mg/dL) is the urgent modal to show.
 * Deliberately no `onMutate` cache: the values live only in the request.
 */
export function useCreateReading() {
  const queryClient = useQueryClient();
  return useMutation<SavedReading, unknown, ReadingInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/vitals/readings', toReadingBody(input));
      return savedSchema.parse(data.data);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: vitalsKeys.all });
    },
  });
}

/** DELETE /vitals/readings/{id} (log-sheet values are read-only and have no id). */
export function useDeleteReading() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, number>({
    mutationFn: async (id) => {
      await apiClient.delete(`/vitals/readings/${id}`);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: vitalsKeys.all });
    },
  });
}

/** Validation messages of a 422, field → first message. */
export function fieldErrors(error: unknown): Record<string, string> {
  if (!(error instanceof ApiError)) return {};
  const body = error.response?.data as ApiEnvelope<unknown> | undefined;
  const errors = body && typeof body === 'object' ? body.errors : undefined;
  const out: Record<string, string> = {};
  if (errors && typeof errors === 'object') {
    for (const [k, v] of Object.entries(errors as Record<string, unknown>)) {
      if (Array.isArray(v) && typeof v[0] === 'string') out[k] = v[0];
    }
  }
  return out;
}
