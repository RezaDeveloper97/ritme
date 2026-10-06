'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, ApiError, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { nextPollDelay } from '../model/status';
import { LAB_CONSENT_CODE, type Lab, type MarkerInput } from '../model/types';
import { labKeys } from './keys';
import {
  catalogSchema,
  consentSchema,
  labListSchema,
  labSchema,
  markerDetailSchema,
  statusSchema,
  trendsSchema,
} from './schema';

/*
 * `/api/v1/labs/*` (B-N6-06), Go only. Health data (CLAUDE.md §11): never log a
 * payload or a response. Every write answers with the whole lab, which is put
 * straight into the detail cache.
 */

const notFound = (error: unknown) => error instanceof ApiError && error.response?.status === 404;
const retryOnce = (count: number, error: unknown) => !notFound(error) && count < 1;

/** GET /labs — history (newest first) + upload limits. */
export function useLabs() {
  return useQuery({
    queryKey: labKeys.list(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/labs');
      return labListSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 15_000,
    retry: 1,
  });
}

export async function fetchLab(id: number): Promise<Lab> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/labs/${id}`);
  return labSchema.parse(data.data);
}

/** GET /labs/{id}. 404 `lab_not_found` is not retried. */
export function useLab(id: number | null) {
  return useQuery({
    queryKey: labKeys.detail(id ?? 0),
    queryFn: () => fetchLab(id as number),
    enabled: isAuthenticated() && id != null && id > 0,
    staleTime: 10_000,
    retry: retryOnce,
  });
}

/**
 * GET /labs/{id}/status, polled while the lab is busy ({@link nextPollDelay}):
 * fast at first, slower later, stops when the lab leaves the busy states or
 * after ~10 minutes. Paused while the tab is hidden (TanStack default).
 */
export function useLabStatus(id: number | null) {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: labKeys.status(id ?? 0),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/labs/${id}/status`);
      return statusSchema.parse(data.data);
    },
    enabled: isAuthenticated() && id != null && id > 0,
    staleTime: 0,
    retry: (count, error) => !notFound(error) && count < 3,
    refetchInterval: (q) => nextPollDelay(q.state.data?.status, q.state.dataUpdateCount),
  });
  // the answered-poll counter refetchInterval reads, so the screen knows when polling gave up
  const polls = queryClient.getQueryState(labKeys.status(id ?? 0))?.dataUpdateCount ?? 0;
  return { ...query, polls };
}

/** GET /labs/{id}/markers/{mid} — the marker detail with its trend. */
export function useLabMarker(id: number, markerId: number) {
  return useQuery({
    queryKey: labKeys.marker(id, markerId),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/labs/${id}/markers/${markerId}`);
      return markerDetailSchema.parse(data.data);
    },
    enabled: isAuthenticated() && id > 0 && markerId > 0,
    staleTime: 30_000,
    retry: retryOnce,
  });
}

/** GET /labs/trends — every marker of the verified labs with its series. */
export function useLabTrends(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: labKeys.trends(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/labs/trends');
      return trendsSchema.parse(data.data);
    },
    enabled: isAuthenticated() && (options.enabled ?? true),
    staleTime: 60_000,
    retry: 1,
  });
}

/** GET /labs/markers — the catalog for the «add a missing marker» name suggestions. */
export function useLabCatalog(enabled = true) {
  return useQuery({
    queryKey: labKeys.catalog(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/labs/markers');
      return catalogSchema.parse(data.data);
    },
    enabled: isAuthenticated() && enabled,
    staleTime: 10 * 60_000,
    retry: 1,
  });
}

/** Puts a lab returned by a write into the cache and refreshes the lists. */
function useStoreLab() {
  const queryClient = useQueryClient();
  return (lab: Lab) => {
    queryClient.setQueryData(labKeys.detail(lab.id), lab);
    void queryClient.invalidateQueries({ queryKey: labKeys.list() });
    void queryClient.invalidateQueries({ queryKey: labKeys.trends() });
    void queryClient.invalidateQueries({ queryKey: labKeys.status(lab.id) });
    void queryClient.invalidateQueries({ queryKey: [...labKeys.all, 'marker', lab.id] });
  };
}

/** Snake-case marker body; every key is sent because PUT replaces the row. */
export function toMarkerBody(input: MarkerInput): Record<string, unknown> {
  return {
    name: input.name.trim(),
    value: input.value,
    value_text: input.value === null ? input.valueText : null,
    unit: input.unit,
    ref_low: input.refLow,
    ref_high: input.refHigh,
    ref_text: input.refText,
  };
}

/** POST /labs/{id}/markers — «افزودن شاخص جاافتاده». */
export function useAddLabMarker(id: number) {
  const store = useStoreLab();
  return useMutation<Lab, unknown, MarkerInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(`/labs/${id}/markers`, toMarkerBody(input));
      return labSchema.parse(data.data);
    },
    onSuccess: store,
  });
}

/** PUT /labs/{id}/markers/{mid} — a corrected value. */
export function useUpdateLabMarker(id: number) {
  const store = useStoreLab();
  return useMutation<Lab, unknown, { markerId: number; input: MarkerInput }>({
    mutationFn: async ({ markerId, input }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/labs/${id}/markers/${markerId}`, toMarkerBody(input));
      return labSchema.parse(data.data);
    },
    onSuccess: store,
  });
}

/** DELETE /labs/{id}/markers/{mid} — a misread row. */
export function useDeleteLabMarker(id: number) {
  const store = useStoreLab();
  return useMutation<Lab, unknown, number>({
    mutationFn: async (markerId) => {
      const { data } = await apiClient.delete<ApiEnvelope<unknown>>(`/labs/${id}/markers/${markerId}`);
      return labSchema.parse(data.data);
    },
    onSuccess: store,
  });
}

/** POST /labs/{id}/verify (202) — values confirmed, the explanation is prepared. */
export function useVerifyLab(id: number) {
  const store = useStoreLab();
  return useMutation<Lab, unknown, void>({
    mutationFn: async () => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(`/labs/${id}/verify`);
      return labSchema.parse(data.data);
    },
    onSuccess: store,
  });
}

/** POST /labs/{id}/feedback {helpful} — «این تحلیل مفید بود؟». */
export function useLabFeedback(id: number) {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, boolean>({
    mutationFn: async (helpful) => {
      await apiClient.post(`/labs/${id}/feedback`, { helpful });
    },
    onSuccess: (_, helpful) => {
      queryClient.setQueryData<Lab>(labKeys.detail(id), (lab) => (lab ? { ...lab, feedback: { helpful } } : lab));
    },
  });
}

/** DELETE /labs/{id} — the lab, its values and its files. */
export function useDeleteLab(id: number) {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, void>({
    mutationFn: async () => {
      await apiClient.delete(`/labs/${id}`);
    },
    onSuccess: () => {
      queryClient.removeQueries({ queryKey: labKeys.detail(id) });
      void queryClient.invalidateQueries({ queryKey: labKeys.all });
    },
  });
}

/** GET /consents/ai_lab_analysis — the consent text of the version in force and whether it is granted. */
export function useLabConsent(enabled = true) {
  return useQuery({
    queryKey: labKeys.consent(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/consents/${LAB_CONSENT_CODE}`);
      return consentSchema.parse(data.data);
    },
    enabled: isAuthenticated() && enabled,
    staleTime: 60_000,
    retry: 1,
  });
}

/** PUT /consents/ai_lab_analysis {granted, version} — 409 `consent_version_stale` when a newer text is in force. */
export function useSetLabConsent() {
  const queryClient = useQueryClient();
  return useMutation<unknown, unknown, { granted: boolean; version: number }>({
    mutationFn: async ({ granted, version }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(
        `/consents/${LAB_CONSENT_CODE}`,
        granted ? { granted, version } : { granted },
      );
      return consentSchema.parse(data.data);
    },
    onSuccess: (consent) => {
      queryClient.setQueryData(labKeys.consent(), consent);
      // the privacy screen lists consents under ['consents', …]
      void queryClient.invalidateQueries({ queryKey: ['consents'], predicate: (q) => q.queryKey[1] !== 'ai_lab_analysis' });
    },
    onError: () => void queryClient.invalidateQueries({ queryKey: labKeys.consent() }),
  });
}

/** Re-reads a lab into the cache (the processing screen calls it before handing over, so the next screen never sees a stale status). */
export function useRefreshLab() {
  const queryClient = useQueryClient();
  return (id: number) =>
    queryClient.fetchQuery({ queryKey: labKeys.detail(id), queryFn: () => fetchLab(id), staleTime: 0 });
}
