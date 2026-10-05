'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ChildMeasurement, childKeys } from '@/entities/child';
import { type ApiEnvelope, apiClient, getApiErrorCode } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { toMeasurementBody } from '../model/form';
import type { GrowthIndicator, GrowthSeries, MeasurementForm, MeasurementList } from '../model/types';
import { growthSeriesSchema, measurementListSchema, measurementSchema } from './schema';

/*
 * `/api/v1/children/{id}/measurements` + `/growth` (B-N5-02), Go only. Health
 * data (CLAUDE.md §11): never log a payload or a response. Keys hang under the
 * child's `detail(id)`, so a save also refreshes the child home's measurements.
 */

export const growthKeys = {
  measurements: (id: number) => [...childKeys.detail(id), 'measurements'] as const,
  series: (id: number, indicator: GrowthIndicator) => [...childKeys.detail(id), 'growth', indicator] as const,
};

const enabled = (id: number) => isAuthenticated() && Number.isFinite(id) && id > 0;

/** GET /children/{id}/measurements — newest first, the birth values last. */
export function useMeasurements(id: number) {
  return useQuery<MeasurementList>({
    queryKey: growthKeys.measurements(id),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/children/${id}/measurements`);
      return measurementListSchema.parse(data.data);
    },
    enabled: enabled(id),
    staleTime: 30_000,
    retry: 1,
  });
}

/** GET /children/{id}/growth?indicator= — WHO P3…P97 per month + the child's points. */
export function useGrowthSeries(id: number, indicator: GrowthIndicator) {
  return useQuery<GrowthSeries>({
    queryKey: growthKeys.series(id, indicator),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/children/${id}/growth`, {
        params: { indicator },
      });
      return growthSeriesSchema.parse(data.data);
    },
    enabled: enabled(id),
    staleTime: 60_000,
    retry: 1,
  });
}

/** POST (no `measurementId`) or PUT /children/{id}/measurements/{mid}. Owner only. */
export function useSaveMeasurement(id: number) {
  const queryClient = useQueryClient();
  return useMutation<ChildMeasurement, unknown, { measurementId: number | null; form: MeasurementForm }>({
    mutationFn: async ({ measurementId, form }) => {
      const body = toMeasurementBody(form);
      const { data } =
        measurementId === null
          ? await apiClient.post<ApiEnvelope<unknown>>(`/children/${id}/measurements`, body)
          : await apiClient.put<ApiEnvelope<unknown>>(`/children/${id}/measurements/${measurementId}`, body);
      return measurementSchema.parse(data.data);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: childKeys.all });
    },
  });
}

/** DELETE /children/{id}/measurements/{mid}. */
export function useDeleteMeasurement(id: number) {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, number>({
    mutationFn: async (measurementId) => {
      await apiClient.delete(`/children/${id}/measurements/${measurementId}`);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: childKeys.all });
    },
  });
}

/** 403 `child_read_only`: a spouse viewing a shared child. */
export function isReadOnlyError(error: unknown): boolean {
  return getApiErrorCode(error) === 'child_read_only';
}
